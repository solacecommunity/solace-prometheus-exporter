// Command semp1fixtures regenerates the synthetic SEMP v1 replies used by the golden tests in internal/semp.
//
// Run it via `go generate ./internal/semp` after adding a getter or changing a getter's reply struct, then
// regenerate the golden files and review their diff (see AGENTS.md, section Testing).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"solace_exporter/internal/semp/semp1fixture"
)

func main() {
	sempDir := flag.String("semp-dir", "internal/semp", "directory containing the get*Semp1.go getters")
	outDir := flag.String("out", "internal/semp/testdata/semp1", "directory to write the <Getter>.xml replies to")
	flag.Parse()

	if err := run(*sempDir, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "semp1fixtures:", err)
		os.Exit(1)
	}
}

func run(sempDir, outDir string) error {
	fixtures, err := semp1fixture.Generate(sempDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(fixtures))
	for name := range fixtures {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(outDir, name), fixtures[name], 0o644); err != nil { //nolint:gosec // test fixtures, not secrets
			return err
		}
	}
	fmt.Printf("semp1fixtures: wrote %d replies to %s\n", len(names), outDir)
	return nil
}
