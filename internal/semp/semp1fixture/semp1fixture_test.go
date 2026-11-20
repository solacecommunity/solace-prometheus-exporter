package semp1fixture

import (
	"os"
	"path/filepath"
	"testing"
)

const getterSource = `package semp

func (semp *Semp) GetThingSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Thing []struct {
					Name    string  ` + "`xml:\"name\"`" + `
					Enabled bool    ` + "`xml:\"enabled\"`" + `
					Rate    float64 ` + "`xml:\"stats>rate\"`" + `
					Kind    string  ` + "`xml:\"kind,attr\"`" + `
				} ` + "`xml:\"thing\"`" + `
			} ` + "`xml:\"show\"`" + `
		} ` + "`xml:\"rpc\"`" + `
		types.Reply
	}
	return 1, nil
}
`

func TestGenerateFillsEveryFieldWithUniqueValuesAndRepeatsTheLastListElement(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "getThingSemp1.go"), []byte(getterSource), 0o600); err != nil {
		t.Fatal(err)
	}

	fixtures, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Leaves are numbered in field order; a bool takes the next number and is true when it is even.
	want := `<rpc-reply semp-version="soltr/10_0VMR"><rpc><show>` +
		`<thing kind="4"><name>1</name><enabled>true</enabled><stats><rate>3</rate></stats></thing>` +
		`<thing kind="8"><name>5</name><enabled>true</enabled><stats><rate>7</rate></stats></thing>` +
		`<thing kind="8"><name>5</name><enabled>true</enabled><stats><rate>7</rate></stats></thing>` +
		`</show></rpc>` + MoreCookiePlaceholder + `<execute-result code="ok"/></rpc-reply>` + "\n"
	if got := string(fixtures["GetThingSemp1.xml"]); got != want {
		t.Errorf("reply =\n%s\nwant\n%s", got, want)
	}
}
