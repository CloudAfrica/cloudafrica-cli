package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// handWritten is every hand-written page that names commands.
func handWritten(t *testing.T) []string {
	t.Helper()
	pages, err := filepath.Glob("../../../docs/*.md")
	if err != nil {
		t.Fatal(err)
	}
	pages = append(pages, "../../../README.md", "../../../CONTRIBUTING.md")
	var kept []string
	for _, p := range pages {
		if filepath.Base(p) != "traps.md" {
			kept = append(kept, p)
		}
	}
	return kept
}

var inlineCa = regexp.MustCompile("`(ca [^`]+)`")

// commandLines returns each `ca ...` line in a fenced block and each `ca ...`
// code span, with backslash continuations joined.
func commandLines(t *testing.T, page string) []string {
	t.Helper()
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	fenced := false
	pending := ""
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
			continue
		}
		if !fenced {
			for _, m := range inlineCa.FindAllStringSubmatch(line, -1) {
				lines = append(lines, m[1])
			}
			continue
		}
		line = pending + line
		pending = ""
		if strings.HasSuffix(line, `\`) {
			pending = strings.TrimSuffix(line, `\`) + " "
			continue
		}
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "ca ") {
			lines = append(lines, t)
		}
	}
	return lines
}

func sub(c *cobra.Command, name string) *cobra.Command {
	for _, s := range c.Commands() {
		if s.Name() == name || s.HasAlias(name) {
			return s
		}
	}
	return nil
}

// resolve walks a command line's words to the commands it names; a word
// `a|b` names alternatives. It returns those commands and the flags given.
func resolve(root *cobra.Command, line string) ([]*cobra.Command, []string, string) {
	cmds := []*cobra.Command{root}
	words := strings.Fields(line)[1:]
	i := 0
	for ; i < len(words); i++ {
		w := words[i]
		if w == "|" || strings.HasPrefix(w, "-") || strings.HasPrefix(w, "<") || strings.HasPrefix(w, "[") {
			break
		}
		var next []*cobra.Command
		for _, c := range cmds {
			for _, alt := range strings.Split(w, "|") {
				if s := sub(c, alt); s != nil {
					next = append(next, s)
				} else if strings.Contains(w, "|") {
					return nil, nil, "no command " + alt + " under " + c.CommandPath()
				}
			}
		}
		if len(next) == 0 {
			break
		}
		cmds = next
	}
	for _, c := range cmds {
		if !c.Runnable() && !(i < len(words) && strings.HasPrefix(words[i], "<")) {
			return nil, nil, c.CommandPath() + " is a group, not a command"
		}
	}
	var flags []string
	for _, w := range words[i:] {
		if w == "|" || w == ">" {
			break
		}
		w = strings.Trim(w, "[]")
		if strings.HasPrefix(w, "-") {
			flags = append(flags, strings.SplitN(w, "=", 2)[0])
		}
	}
	return cmds, flags, ""
}

func hasFlag(c *cobra.Command, f string) bool {
	if f == "--help" || f == "-h" {
		return true // cobra adds it when the command runs
	}
	fs := c.Flags()
	if strings.HasPrefix(f, "--") {
		return fs.Lookup(f[2:]) != nil || c.InheritedFlags().Lookup(f[2:]) != nil
	}
	s := strings.TrimPrefix(f, "-")
	return fs.ShorthandLookup(s) != nil || c.InheritedFlags().ShorthandLookup(s) != nil
}

func TestDocumentedCommandsExist(t *testing.T) {
	r := root()
	seen := map[string]bool{}
	for _, page := range handWritten(t) {
		for _, line := range commandLines(t, page) {
			cmds, flags, problem := resolve(r, line)
			if problem != "" {
				t.Errorf("%s: %q: %s", filepath.Base(page), line, problem)
				continue
			}
			for _, c := range cmds {
				for p := c; p.HasParent(); p = p.Parent() {
					if p.Parent() == r {
						seen[p.Name()] = true
					}
				}
				for _, f := range flags {
					if !hasFlag(c, f) {
						t.Errorf("%s: %q: %s has no flag %s", filepath.Base(page), line, c.CommandPath(), f)
					}
				}
			}
		}
	}
	for _, c := range r.Commands() {
		if !c.Hidden && c.Name() != "help" && !seen[c.Name()] {
			t.Errorf("ca %s is on no hand-written page", c.Name())
		}
	}
}
