// Package semp1fixture derives deterministic, synthetic SEMP v1 replies for the golden tests in internal/semp.
//
// For every GetXxxSemp1 getter it reads the xml struct tags of the getter's local `Data` type and builds a reply
// that fills every field: each leaf gets a unique, increasing number (so an emitted metric value can be traced back to
// one field), booleans alternate between false and true, and struct slices get three elements whose third repeats the
// second (exercising the getters' skipping of consecutive duplicates on paged replies). The shared more-cookie and
// execute-result elements are left to the test harness: the reply contains a {{MORE_COOKIE}} placeholder and an "ok"
// execute-result.
//
// Generation is deterministic: unchanged getters always produce byte-identical replies.
package semp1fixture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// MoreCookiePlaceholder marks where the test harness inserts an optional more-cookie element.
const MoreCookiePlaceholder = "{{MORE_COOKIE}}"

// replyTypeName is the name of the local reply type every SEMP v1 getter declares.
const replyTypeName = "Data"

// Generate parses the get*Semp1.go files in sempDir and returns one reply per getter, keyed by "<Getter>.xml".
func Generate(sempDir string) (map[string][]byte, error) {
	files, err := filepath.Glob(filepath.Join(sempDir, "get*Semp1.go"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	fixtures := make(map[string][]byte)
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil {
				continue
			}
			reply, found, err := replyFor(fn)
			if err != nil {
				return nil, fmt.Errorf("%s: %s: %w", filepath.Base(file), fn.Name.Name, err)
			}
			if found {
				fixtures[fn.Name.Name+".xml"] = reply
			}
		}
	}
	return fixtures, nil
}

// replyFor builds the reply for the Data type declared in fn; found is false if fn declares none.
func replyFor(fn *ast.FuncDecl) (reply []byte, found bool, err error) {
	var data *ast.StructType
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != replyTypeName {
			return data == nil
		}
		data, _ = spec.Type.(*ast.StructType)
		return false
	})
	if data == nil {
		return nil, false, nil
	}

	g := &generator{}
	root := &element{name: "rpc-reply", attrs: []string{`semp-version="soltr/10_0VMR"`}}
	if err := g.fill(root, data); err != nil {
		return nil, true, err
	}
	var sb strings.Builder
	root.render(&sb)
	xml := strings.Replace(sb.String(), "</rpc-reply>",
		MoreCookiePlaceholder+`<execute-result code="ok"/></rpc-reply>`, 1)
	return []byte(xml + "\n"), true, nil
}

// element is an XML element under construction.
type element struct {
	name     string
	attrs    []string
	children []*element
	text     string
}

// child returns the child element with the given name, creating it on first use.
func (e *element) child(name string) *element {
	for _, c := range e.children {
		if c.name == name {
			return c
		}
	}
	c := &element{name: name}
	e.children = append(e.children, c)
	return c
}

func (e *element) render(sb *strings.Builder) {
	sb.WriteString("<" + e.name)
	for _, attr := range e.attrs {
		sb.WriteString(" " + attr)
	}
	sb.WriteString(">" + e.text)
	for _, c := range e.children {
		c.render(sb)
	}
	sb.WriteString("</" + e.name + ">")
}

// generator hands out the unique leaf values.
type generator struct {
	counter int
}

func (g *generator) next() int {
	g.counter++
	return g.counter
}

func (g *generator) leaf(typeName string) string {
	if typeName == "bool" {
		if g.next()%2 == 0 {
			return "true"
		}
		return "false"
	}
	// Numbers and strings alike: a unique number parses as a number and is a traceable label value.
	return strconv.Itoa(g.next())
}

// fill adds the fields of st to parent.
func (g *generator) fill(parent *element, st *ast.StructType) error {
	for _, field := range st.Fields.List {
		path, flags := xmlTag(field)
		if path == "-" {
			continue
		}
		typ := field.Type
		if star, ok := typ.(*ast.StarExpr); ok {
			typ = star.X
		}
		if _, shared := typ.(*ast.SelectorExpr); shared {
			continue // types.MoreCookie / types.ExecuteResult / types.Reply are added by the test harness
		}
		switch {
		case flags["attr"]:
			parent.attrs = append(parent.attrs, fmt.Sprintf("%s=%q", path, g.leaf(identName(typ))))
			continue
		case flags["chardata"] || flags["innerxml"]:
			parent.text = g.leaf(identName(typ))
			continue
		}

		segments := strings.Split(path, ">")
		holder := parent
		for _, segment := range segments[:len(segments)-1] {
			holder = holder.child(segment)
		}
		name := segments[len(segments)-1]

		if array, ok := typ.(*ast.ArrayType); ok {
			var second *element
			for i := 0; i < 2; i++ {
				el := &element{name: name}
				if err := g.value(el, array.Elt); err != nil {
					return err
				}
				holder.children = append(holder.children, el)
				second = el
			}
			holder.children = append(holder.children, second) // consecutive duplicate on purpose
			continue
		}
		if err := g.value(holder.child(name), typ); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) value(el *element, typ ast.Expr) error {
	switch t := typ.(type) {
	case *ast.StructType:
		return g.fill(el, t)
	case *ast.Ident:
		el.text = g.leaf(t.Name)
		return nil
	case *ast.StarExpr:
		return g.value(el, t.X)
	default:
		return fmt.Errorf("unsupported field type %T", typ)
	}
}

// xmlTag returns the element path and flags of a field's xml tag; the path defaults to the field name.
func xmlTag(field *ast.Field) (path string, flags map[string]bool) {
	tag := ""
	if field.Tag != nil {
		unquoted, err := strconv.Unquote(field.Tag.Value)
		if err == nil {
			tag = reflect.StructTag(unquoted).Get("xml")
		}
	}
	parts := strings.Split(tag, ",")
	path = parts[0]
	if path == "" && len(field.Names) > 0 {
		path = field.Names[0].Name
	}
	flags = make(map[string]bool, len(parts)-1)
	for _, flag := range parts[1:] {
		flags[flag] = true
	}
	return path, flags
}

func identName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
