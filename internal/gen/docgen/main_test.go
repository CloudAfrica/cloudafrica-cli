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
	pages = append(pages, "../../../README.md")
	var kept []string
	for _, p := range pages {
		if filepath.Base(p) != "traps.md" {
			kept = append(kept, p)
		}
	}
	return kept
}

var inlineCa = regexp.MustCompile("`(ca [^`]+)`")

// fencedCa finds a `ca ...` invocation at the start of a line, inside $(...),
// or on the right of an assignment or && — up to the end of that command.
var fencedCa = regexp.MustCompile(`(?:^|\$\(|=\s*|&&\s*|;\s*)(ca\s[^)&;|]*)`)

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
		for _, m := range fencedCa.FindAllStringSubmatch(strings.TrimSpace(line), -1) {
			lines = append(lines, strings.TrimSpace(m[1]))
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
		if strings.HasPrefix(w, "-") && i == 0 {
			return nil, nil, "put flags after the command, so the test can resolve it"
		}
		if w == "|" || strings.HasPrefix(w, "-") || strings.HasPrefix(w, "<") || strings.HasPrefix(w, "[") {
			break
		}
		var next []*cobra.Command
		for _, c := range cmds {
			for _, alt := range strings.Split(w, "|") {
				if s := sub(c, alt); s != nil {
					if s.Name() != alt {
						return nil, nil, alt + " is an alias: write ca " + strings.TrimPrefix(s.CommandPath(), "ca ")
					}
					next = append(next, s)
				} else if strings.Contains(w, "|") {
					return nil, nil, "no command " + alt + " under " + c.CommandPath()
				}
			}
		}
		if len(next) == 0 {
			for _, c := range cmds {
				if c.HasAvailableSubCommands() {
					return nil, nil, "no command " + w + " under " + c.CommandPath()
				}
			}
			break
		}
		cmds = next
	}
	for _, c := range cmds {
		if !c.Runnable() && (i >= len(words) || !strings.HasPrefix(words[i], "<")) {
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

func TestRunReplacesOnlyReferencePages(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"keep.txt", "notes.md", "ca_removed-command.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(dir); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"keep.txt", "notes.md", "ca.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "ca_removed-command.md")); !os.IsNotExist(err) {
		t.Errorf("stale page kept: %v", err)
	}
}

func TestCommandLinesFindsCapturedInvocations(t *testing.T) {
	page := filepath.Join(t.TempDir(), "p.md")
	body := "```\nX=$(ca auth elevate) || exit\n$env:X = ca auth elevate\nca vm list -o json | jq .\n```\n"
	if err := os.WriteFile(page, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(commandLines(t, page), "\n")
	want := "ca auth elevate\nca auth elevate\nca vm list -o json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A documented line that reaches a command through an alias is refused and names the canonical path.
func TestResolveRefusesAnAlias(t *testing.T) {
	root := &cobra.Command{Use: "ca"}
	vms := &cobra.Command{Use: "vms", Aliases: []string{"vm"}}
	vms.AddCommand(&cobra.Command{Use: "list", Run: func(*cobra.Command, []string) {}})
	root.AddCommand(vms)
	if _, _, problem := resolve(root, "ca vms list"); problem != "" {
		t.Fatalf("ca vms list: %s", problem)
	}
	_, _, problem := resolve(root, "ca vm list")
	if !strings.Contains(problem, "vm is an alias") || !strings.Contains(problem, "write ca vms") {
		t.Errorf("ca vm list: got %q, want an alias refusal naming ca vms", problem)
	}
}

var referenceLink = regexp.MustCompile(`reference/(ca[A-Za-z0-9_.-]*\.md)`)

// Every link from a hand-written page to a generated reference page names a page that exists.
func TestReferenceLinksExist(t *testing.T) {
	links := 0
	for _, p := range handWritten(t) {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range referenceLink.FindAllStringSubmatch(string(b), -1) {
			links++
			if _, err := os.Stat(filepath.Join("../../../docs/reference", m[1])); err != nil {
				t.Errorf("%s links reference/%s, which does not exist", filepath.Base(p), m[1])
			}
		}
	}
	if links == 0 {
		t.Fatal("found no reference links: the pattern or the page list is wrong")
	}
}
