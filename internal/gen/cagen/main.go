// Command cagen writes internal/cli/commands_gen.go from the SDK's openapi.json
// and internal/cli/commands.yaml. Run it with `go generate ./...`.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/CloudAfrica/cloudafrica-cli/internal/gen"
)

func main() {
	overlay := flag.String("overlay", "commands.yaml", "the overlay")
	out := flag.String("out", "commands_gen.go", "the generated file")
	pkg := flag.String("pkg", "cli", "the generated file's package")
	doc := flag.String("doc", "", "openapi.json (default: the SDK module's)")
	list := flag.Bool("list", false, "print each generated command and its operation, then exit")
	flag.Parse()
	if err := run(*overlay, *out, *pkg, *doc, *list); err != nil {
		fmt.Fprintln(os.Stderr, "cagen:", err)
		os.Exit(1)
	}
}

func run(overlayPath, outPath, pkg, docPath string, list bool) error {
	if docPath == "" {
		dir, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/CloudAfrica/cloudafrica-go").Output()
		if err != nil {
			return fmt.Errorf("locating the SDK: %w", err)
		}
		docPath = filepath.Join(strings.TrimSpace(string(dir)), "openapi.json")
	}
	docJSON, err := os.ReadFile(docPath)
	if err != nil {
		return err
	}
	ov, err := os.ReadFile(overlayPath)
	if err != nil {
		return err
	}
	table, report, err := gen.Generate(docJSON, ov)
	if list && table != nil {
		for _, op := range table.Ops {
			pos := ""
			if op.Positional.Name != "" {
				pos = " <" + op.Positional.Name + ">"
			}
			var flags []string
			for _, p := range op.Params {
				flags = append(flags, "--"+p.Flag)
			}
			fmt.Printf("%-44s %-7s %-60s %s %s\n", strings.Join(op.Path, " ")+pos, op.Method, op.URL, op.OperationID, strings.Join(flags, " "))
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s\n(%s)", err, report)
	}
	src, err := gen.Source(table, pkg)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "cagen:", report)
	return os.WriteFile(outPath, src, 0o644)
}
