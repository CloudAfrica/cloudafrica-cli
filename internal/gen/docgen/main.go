// Command docgen writes the command reference under docs/reference from the
// CLI's command tree. Run it with `go generate ./...`.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/CloudAfrica/cloudafrica-cli/internal/cli"
)

func main() {
	out := flag.String("out", "../../docs/reference", "the reference directory")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "docgen:", err)
		os.Exit(1)
	}
}

// root is the command tree the reference documents, with the completion
// command cobra otherwise adds only when it runs.
func root() *cobra.Command {
	app := &cli.App{Stdout: io.Discard, Stderr: io.Discard, Getenv: func(string) string { return "" }}
	r := app.Root()
	r.DisableAutoGenTag = true
	r.InitDefaultCompletionCmd()
	return r
}

// run removes the ca*.md pages in dir, so a removed command loses its page,
// and writes the reference. It removes nothing else.
func run(dir string) error {
	old, err := filepath.Glob(filepath.Join(dir, "ca*.md"))
	if err != nil {
		return err
	}
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return doc.GenMarkdownTree(root(), dir)
}
