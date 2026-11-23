// Command docgen writes the command reference under docs/reference from the
// CLI's command tree. Run it with `go generate ./...`.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

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

// run rewrites dir from scratch, so a removed command loses its page.
func run(dir string) error {
	app := &cli.App{Stdout: io.Discard, Stderr: io.Discard, Getenv: func(string) string { return "" }}
	root := app.Root()
	root.DisableAutoGenTag = true
	root.InitDefaultCompletionCmd()
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return doc.GenMarkdownTree(root, dir)
}
