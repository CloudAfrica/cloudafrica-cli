package cli

import (
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// v090Command is one leaf of the v0.9.0 tree (testdata/v0.9.0-commands.tsv): its path, the operation it ran
// ("-" for a hand-written command), whether it took an argument, and its own flags.
type v090Command struct {
	path  []string
	op    string
	args  string
	flags []string
}

func v090Commands(t *testing.T) []v090Command {
	t.Helper()
	b, err := os.ReadFile("testdata/v0.9.0-commands.tsv")
	if err != nil {
		t.Fatal(err)
	}
	var cmds []v090Command
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Split(line, "\t")
		c := v090Command{path: strings.Fields(f[0]), op: f[1], args: f[2]}
		if f[3] != "" {
			c.flags = strings.Split(f[3], ",")
		}
		cmds = append(cmds, c)
	}
	return cmds
}

func treeRoot() *cobra.Command {
	a := &App{Stdout: io.Discard, Stderr: io.Discard, Getenv: func(string) string { return "" }}
	return a.Root()
}

// resolveOld walks an old path through names and aliases, recording each word that was an alias.
func resolveOld(root *cobra.Command, path []string, viaAlias map[*cobra.Command]map[string]bool) *cobra.Command {
	cur := root
	for _, w := range path {
		var next *cobra.Command
		for _, c := range cur.Commands() {
			if c.Name() == w || c.HasAlias(w) {
				next = c
			}
		}
		if next == nil {
			return nil
		}
		if next.Name() != w {
			if viaAlias[next] == nil {
				viaAlias[next] = map[string]bool{}
			}
			viaAlias[next][w] = true
		}
		cur = next
	}
	return cur
}

func ownFlags(c *cobra.Command) []string {
	var fl []string
	c.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name != "help" {
			fl = append(fl, f.Name)
		}
	})
	slices.Sort(fl)
	return fl
}

// moved lists the v0.9.0 commands that no old name reaches (CA-5660 §3): each one's new path ("" for a
// removed command), and, for addVmDisk, the flag it gained when its argument became one. The release
// notes' breaking changes are written from this map.
var moved = map[string]struct {
	to, addedFlag string
}{
	"api":                                   {},
	"vm add-disk":                           {"vms disks create", "vm"},
	"vpc vip create":                        {"vips create", ""},
	"vpc vip delete":                        {"vips delete", ""},
	"vpc vip firewall-rule create":          {"vips firewall-rules create", ""},
	"vpc vip firewall-rule delete":          {"vips firewall-rules delete", ""},
	"vpc vip firewall-rule list":            {"vips firewall-rules list", ""},
	"vpc vip firewall-rule set-description": {"vips firewall-rules update", ""},
	"vpc vip public-ip attach":              {"vips public-ip attach", ""},
	"vpc vip public-ip detach":              {"vips public-ip detach", ""},
}

// Every v0.9.0 command still runs under its old name, runs the same operation and takes the same
// argument and flags; a moved one does so at its new path, and nothing else moved.
func TestV090NamesStillRunTheSameCommand(t *testing.T) {
	root := treeRoot()
	viaAlias := map[*cobra.Command]map[string]bool{}
	for _, old := range v090Commands(t) {
		name := strings.Join(old.path, " ")
		c := resolveOld(root, old.path, viaAlias)
		if m, ok := moved[name]; ok {
			if c != nil && !c.HasSubCommands() {
				t.Errorf("ca %s still resolves; drop it from moved", name)
			}
			if m.to == "" {
				continue
			}
			c, _, _ = root.Find(strings.Fields(m.to))
			if c == nil || c.CommandPath() != "ca "+m.to {
				t.Errorf("ca %s moved to ca %s, which does not exist", name, m.to)
				continue
			}
			if m.addedFlag != "" {
				old.flags = append(slices.Clone(old.flags), m.addedFlag)
				slices.Sort(old.flags)
				old.args = "0"
			}
		}
		if c == nil || c.HasSubCommands() {
			t.Errorf("ca %s no longer resolves to a command", name)
			continue
		}
		if got := c.Annotations[operationIDKey]; old.op != "-" && got != old.op {
			t.Errorf("ca %s now runs %q, not %s", name, got, old.op)
		}
		if old.op == "-" && c.Name() != old.path[len(old.path)-1] {
			t.Errorf("ca %s resolves to the hand-written %s", name, c.CommandPath())
		}
		args := "0"
		if strings.Contains(c.Use, " ") {
			args = "1"
		}
		if args != old.args {
			t.Errorf("ca %s took %s arguments in v0.9.0, now %s", name, old.args, args)
		}
		if got := ownFlags(c); !slices.Equal(got, old.flags) {
			t.Errorf("ca %s flags = %v, v0.9.0 had %v", name, got, old.flags)
		}
	}
}

// Every alias is a v0.9.0 name some old path uses, apart from the hand-written aliases v0.9.0 had.
func TestEveryAliasIsAV090Name(t *testing.T) {
	root := treeRoot()
	viaAlias := map[*cobra.Command]map[string]bool{}
	for _, old := range v090Commands(t) {
		resolveOld(root, old.path, viaAlias)
	}
	kept := map[string]bool{"ca operations op": true, "ca k8s kubernetes": true}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, al := range c.Aliases {
			if !viaAlias[c][al] && !kept[c.CommandPath()+" "+al] {
				t.Errorf("%s: alias %q is no v0.9.0 name; list it in commands.yaml aliases:", c.CommandPath(), al)
			}
		}
		for _, s := range c.Commands() {
			walk(s)
		}
	}
	walk(root)
}

func TestNoTwoSiblingsAnswerToOneWord(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		seen := map[string]string{}
		for _, s := range c.Commands() {
			for _, w := range append([]string{s.Name()}, s.Aliases...) {
				if prev, ok := seen[w]; ok {
					t.Errorf("%s %s: both %s and %s", c.CommandPath(), w, prev, s.CommandPath())
				}
				seen[w] = s.CommandPath()
			}
			walk(s)
		}
	}
	walk(treeRoot())
}

func TestEveryTopLevelCommandHasASection(t *testing.T) {
	root := treeRoot()
	for _, c := range root.Commands() {
		if c.GroupID == "" || !root.ContainsGroup(c.GroupID) {
			t.Errorf("ca %s has no help section (GroupID %q)", c.Name(), c.GroupID)
		}
	}
}

// A positional is named for the singular noun, under a plural group.
func TestPositionalNamesAreSingular(t *testing.T) {
	root := treeRoot()
	for path, want := range map[string]string{
		"vms nics detach":            "detach NIC_ID",
		"operations get":             "get OPERATION_ID",
		"vms firewall-rules delete":  "delete RULE_ID",
		"vips firewall-rules delete": "delete RULE_ID",
		"vms public-ips detach":      "detach PUBLIC_IP_ID",
	} {
		c, _, err := root.Find(strings.Fields(path))
		if err != nil || c.Use != want {
			t.Errorf("ca %s: use %q, want %q (%v)", path, c.Use, want, err)
		}
	}
}

// A prompt names the group by its v0.9.0 singular, and a singleton group by its own name.
func TestConfirmQuestionNamesTheGroupInTheSingular(t *testing.T) {
	ops := map[string]string{
		"updateBillingDetails": "Set details?",
	}
	for _, op := range genOps {
		want, ok := ops[op.OperationID]
		if !ok {
			continue
		}
		delete(ops, op.OperationID)
		if got := confirmQuestion(op, nil); got != want {
			t.Errorf("%s asks %q, want %q", op.OperationID, got, want)
		}
	}
	for id := range ops {
		t.Errorf("no operation %s", id)
	}
	for path, want := range map[string]string{"vms": "vm", "ips": "ip-address", "billing details": "details"} {
		if got := groupNoun(strings.Fields(path)); got != want {
			t.Errorf("groupNoun(%s) = %q, want %q", path, got, want)
		}
	}
}

// help and completion are created on first use, so this runs the root before looking.
func TestHelpAndCompletionAreInTheCLISection(t *testing.T) {
	root := treeRoot()
	root.SetArgs([]string{"--help"})
	root.SetOut(io.Discard)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"help", "completion"} {
		c, _, err := root.Find([]string{name})
		if err != nil || c.Name() != name {
			t.Fatalf("no %s command: %v", name, err)
		}
		if c.GroupID != "cli" {
			t.Errorf("ca %s is in section %q, want cli", name, c.GroupID)
		}
	}
}
