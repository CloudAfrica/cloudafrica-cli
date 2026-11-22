// Package gen turns the API's openapi.json plus the hand-written overlay
// (internal/cli/commands.yaml) into the command table the executor runs. It
// refuses a table with a clash, a gap or a stale overlay entry, so a new API
// operation fails `go generate` until it is covered or excluded.
package gen

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"

	"sigs.k8s.io/yaml"

	"github.com/CloudAfrica/cloudafrica-cli/internal/cmdspec"
)

// Overlay is commands.yaml.
type Overlay struct {
	Groups      map[string]string            `json:"groups"`
	Nouns       map[string]string            `json:"nouns"`
	Resources   map[string]cmdspec.Resource  `json:"resources"`
	Params      map[string]ParamOverlay      `json:"params"`
	Handwritten map[string]string            `json:"handwritten"`
	Exclude     []Rule                       `json:"exclude"`
	Pending     []Rule                       `json:"pending"`
	Operations  map[string]OpOverlay         `json:"operations"`
	Columns     map[string]cmdspec.ColumnSet `json:"columns"`
}

// ParamOverlay is keyed by the URL up to and including the parameter.
type ParamOverlay struct {
	Resource string `json:"resource"`
	Convert  string `json:"convert"`
	Help     string `json:"help"`
}

// Rule matches operations by id or by path prefix.
type Rule struct {
	Operation string `json:"operation"`
	Prefix    string `json:"prefix"`
	Slice     int    `json:"slice"`
	Reason    string `json:"reason"`
}

// OpOverlay overrides one operation.
type OpOverlay struct {
	Path       string                 `json:"path"`
	Positional string                 `json:"positional"`
	Wait       string                 `json:"wait"`
	Flags      map[string]FlagOverlay `json:"flags"`
	AtLeastOne []string               `json:"atLeastOne"`
	Example    string                 `json:"example"`
}

// FlagOverlay renames, scales or hides a query or body field.
type FlagOverlay struct {
	// Hide leaves the field to --data, for a schema artefact such as a getter.
	Hide  bool   `json:"hide"`
	Name  string `json:"name"`
	Scale int64  `json:"scale"`
	Help  string `json:"help"`
}

// Report counts what the table covers.
type Report struct {
	Generated, Handwritten, Excluded int
	Pending                          map[int]int
}

func (r Report) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d generated, %d hand-written, %d excluded", r.Generated, r.Handwritten, r.Excluded)
	for _, s := range sortedKeys(r.Pending) {
		fmt.Fprintf(&b, ", %d pending slice %d", r.Pending[s], s)
	}
	return b.String()
}

// reserved are the executor's and the root's own flags.
var reserved = []string{"timeout", "no-wait", "yes", "data", "limit", "cursor", "output", "profile", "quiet", "help"}

type doc struct {
	Paths      map[string]map[string]json.RawMessage `json:"paths"`
	Components struct {
		Schemas map[string]*schema `json:"schemas"`
	} `json:"components"`
}

type schema struct {
	Ref         string             `json:"$ref"`
	Type        any                `json:"type"`
	Enum        []any              `json:"enum"`
	Properties  map[string]*schema `json:"properties"`
	Required    []string           `json:"required"`
	Items       *schema            `json:"items"`
	AllOf       []*schema          `json:"allOf"`
	OneOf       []*schema          `json:"oneOf"`
	AnyOf       []*schema          `json:"anyOf"`
	Description string             `json:"description"`
	Pattern     string             `json:"pattern"`
	Format      string             `json:"format"`
}

type operation struct {
	OperationID string      `json:"operationId"`
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Parameters  []parameter `json:"parameters"`
	RequestBody *struct {
		Required bool                                `json:"required"`
		Content  map[string]struct{ Schema *schema } `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Headers map[string]any                      `json:"headers"`
		Content map[string]struct{ Schema *schema } `json:"content"`
	} `json:"responses"`
	Security []map[string][]string `json:"security"`
}

type parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required"`
	Description string  `json:"description"`
	Schema      *schema `json:"schema"`
}

// Output is the generated table.
type Output struct {
	Ops       []cmdspec.Op
	Groups    []cmdspec.Group
	Columns   map[string]cmdspec.ColumnSet
	Resources map[string]cmdspec.Resource
	Scopes    []string
}

// Generate builds the table from the document and the overlay. On failure the
// error lists every problem found and the table is what could be built.
func Generate(docJSON, overlayYAML []byte) (*Output, Report, error) {
	var d doc
	if err := json.Unmarshal(docJSON, &d); err != nil {
		return nil, Report{}, fmt.Errorf("openapi.json: %w", err)
	}
	ov, err := ParseOverlay(overlayYAML)
	if err != nil {
		return nil, Report{}, err
	}
	g := &generator{d: &d, ov: ov, rep: Report{Pending: map[int]int{}}}
	out := g.run()
	if len(g.errs) > 0 {
		sort.Strings(g.errs)
		return out, g.rep, errors.New(strings.Join(g.errs, "\n"))
	}
	return out, g.rep, nil
}

// ParseOverlay reads commands.yaml, refusing unknown keys.
func ParseOverlay(b []byte) (*Overlay, error) {
	j, err := yaml.YAMLToJSON(b)
	if err != nil {
		return nil, fmt.Errorf("commands.yaml: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(j))
	dec.DisallowUnknownFields()
	var ov Overlay
	if err := dec.Decode(&ov); err != nil {
		return nil, fmt.Errorf("commands.yaml: %w", err)
	}
	return &ov, nil
}

type generator struct {
	d    *doc
	ov   *Overlay
	rep  Report
	errs []string
}

func (g *generator) fail(format string, args ...any) {
	g.errs = append(g.errs, fmt.Sprintf(format, args...))
}

var methods = []string{"get", "post", "put", "patch", "delete"}

func (g *generator) run() *Output {
	out := &Output{Columns: g.ov.Columns, Resources: g.ov.Resources}
	seen := map[string]bool{}
	usedColumns := map[string]bool{}
	scopes := map[string]bool{}
	type pathOwner struct{ path, op string }
	var owners []pathOwner

	for _, url := range sortedKeys(g.d.Paths) {
		item := g.d.Paths[url]
		for _, m := range methods {
			raw, ok := item[m]
			if !ok {
				continue
			}
			var op operation
			if err := json.Unmarshal(raw, &op); err != nil {
				g.fail("%s %s: %v", m, url, err)
				continue
			}
			seen[op.OperationID] = true
			oo := g.ov.Operations[op.OperationID]
			if hw, ok := g.ov.Handwritten[op.OperationID]; ok {
				g.rep.Handwritten++
				owners = append(owners, pathOwner{hw, op.OperationID})
				addScopes(scopes, op)
				continue
			}
			if r, ok := g.match(g.ov.Exclude, op.OperationID, url); ok {
				if r.Reason == "" {
					g.fail("%s: an exclusion needs a reason", op.OperationID)
				}
				g.rep.Excluded++
				continue
			}
			if r, ok := g.match(g.ov.Pending, op.OperationID, url); ok {
				if r.Slice == 0 {
					g.fail("%s: a pending entry needs its slice", op.OperationID)
				}
				g.rep.Pending[r.Slice]++
				addScopes(scopes, op)
				continue
			}
			c, ok := g.build(strings.ToUpper(m), url, op, oo)
			if !ok {
				continue
			}
			if c.Columns != "" {
				usedColumns[c.Columns] = true
			}
			addScopes(scopes, op)
			g.rep.Generated++
			out.Ops = append(out.Ops, c)
			owners = append(owners, pathOwner{strings.Join(c.Path, " "), op.OperationID})
		}
	}

	// Clashes: two commands on one path, or a command that is also a group.
	byPath := map[string]string{}
	for _, o := range owners {
		if prev, ok := byPath[o.path]; ok {
			g.fail("command %q: both %s and %s", o.path, prev, o.op)
		}
		byPath[o.path] = o.op
	}
	groups := map[string]bool{}
	for _, op := range out.Ops {
		parts := op.Path
		for i := 1; i < len(parts); i++ {
			groups[strings.Join(parts[:i], " ")] = true
		}
	}
	for _, p := range sortedKeys(groups) {
		if op, ok := byPath[p]; ok {
			g.fail("command %q (%s) is also a group: rename it in commands.yaml", p, op)
		}
		short := g.ov.Groups[p]
		if short == "" {
			parts := strings.Fields(p)
			short = "Manage " + parts[len(parts)-1] + "s"
		}
		out.Groups = append(out.Groups, cmdspec.Group{Path: strings.Fields(p), Short: short})
	}
	for _, p := range sortedKeys(g.ov.Groups) {
		if !groups[p] {
			g.fail("commands.yaml groups: %q is not a group of generated commands", p)
		}
	}

	// Stale overlay entries.
	for _, id := range sortedKeys(g.ov.Operations) {
		if !seen[id] {
			g.fail("commands.yaml operations: %s is not in openapi.json", id)
		}
	}
	for _, id := range sortedKeys(g.ov.Handwritten) {
		if !seen[id] {
			g.fail("commands.yaml handwritten: %s is not in openapi.json", id)
		}
	}
	for _, rules := range [][]Rule{g.ov.Exclude, g.ov.Pending} {
		for _, r := range rules {
			if r.Operation != "" && !seen[r.Operation] {
				g.fail("commands.yaml: %s is not in openapi.json", r.Operation)
			}
			if r.Prefix != "" && !g.anyPath(r.Prefix) {
				g.fail("commands.yaml: no path starts with %s", r.Prefix)
			}
		}
	}
	for _, name := range sortedKeys(g.ov.Columns) {
		if _, ok := g.d.Components.Schemas[name]; !ok {
			g.fail("commands.yaml columns: schema %s is not in openapi.json", name)
		}
	}
	for _, name := range sortedKeys(g.ov.Resources) {
		r := g.ov.Resources[name]
		if r.Noun == "" || r.Match == "" {
			g.fail("commands.yaml resources %s: noun and match are required", name)
		}
		if _, ok := g.d.Paths[r.List]; !ok {
			g.fail("commands.yaml resources %s: list %s is not in openapi.json", name, r.List)
		}
		if _, err := regexp.Compile(r.IDPattern); err != nil {
			g.fail("commands.yaml resources %s: idPattern: %v", name, err)
		}
	}
	for _, name := range sortedKeys(g.ov.Columns) {
		if !usedColumns[name] {
			g.fail("commands.yaml columns: %s is used by no command", name)
		}
	}
	for _, p := range sortedKeys(g.ov.Params) {
		if !g.anyPath(p) {
			g.fail("commands.yaml params: no path starts with %s", p)
		}
		if r := g.ov.Params[p].Resource; r != "" {
			if _, ok := g.ov.Resources[r]; !ok {
				g.fail("commands.yaml params %s: no resource %q", p, r)
			}
		}
	}

	for s := range scopes {
		out.Scopes = append(out.Scopes, s)
	}
	sort.Strings(out.Scopes)
	sort.Slice(out.Ops, func(i, j int) bool {
		return strings.Join(out.Ops[i].Path, " ") < strings.Join(out.Ops[j].Path, " ")
	})
	return out
}

func addScopes(into map[string]bool, op operation) {
	for _, req := range op.Security {
		for _, ss := range req {
			for _, s := range ss {
				if strings.HasPrefix(s, "read:") || strings.HasPrefix(s, "write:") {
					into[s] = true
				}
			}
		}
	}
}

func (g *generator) anyPath(prefix string) bool {
	for p := range g.d.Paths {
		if p == prefix || strings.HasPrefix(p, prefix+"/") || (strings.HasSuffix(prefix, "/") && strings.HasPrefix(p, prefix)) {
			return true
		}
	}
	return false
}

func (g *generator) match(rules []Rule, id, url string) (Rule, bool) {
	for _, r := range rules {
		if r.Operation == id {
			return r, true
		}
	}
	for _, r := range rules {
		if r.Prefix != "" && (url == r.Prefix || strings.HasPrefix(url, strings.TrimSuffix(r.Prefix, "/")+"/")) {
			return r, true
		}
	}
	return Rule{}, false
}

func isParam(seg string) bool { return strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") }

// Singular singularises a kebab-case path segment.
func Singular(s string) string {
	switch {
	case strings.HasSuffix(s, "ies"):
		return strings.TrimSuffix(s, "ies") + "y"
	case strings.HasSuffix(s, "sses"), strings.HasSuffix(s, "xes"):
		return strings.TrimSuffix(s, "es")
	case strings.HasSuffix(s, "ss"):
		return s
	case strings.HasSuffix(s, "s"):
		return strings.TrimSuffix(s, "s")
	}
	return s
}

func (g *generator) noun(seg string) string {
	if a, ok := g.ov.Nouns[seg]; ok {
		return a
	}
	return Singular(seg)
}

func (g *generator) resolve(s *schema) *schema {
	for i := 0; s != nil && i < 10; i++ {
		switch {
		case s.Ref != "":
			s = g.d.Components.Schemas[strings.TrimPrefix(s.Ref, "#/components/schemas/")]
		case len(s.AllOf) == 1:
			s = s.AllOf[0]
		default:
			return s
		}
	}
	return s
}

func refName(s *schema) string {
	if s == nil {
		return ""
	}
	if s.Ref != "" {
		return strings.TrimPrefix(s.Ref, "#/components/schemas/")
	}
	if len(s.AllOf) == 1 {
		return refName(s.AllOf[0])
	}
	return ""
}

func typeOf(s *schema) string {
	switch t := s.Type.(type) {
	case string:
		return t
	case []any:
		for _, v := range t {
			if v != "null" {
				return fmt.Sprint(v)
			}
		}
	}
	return ""
}

func jsonSchema(content map[string]struct{ Schema *schema }) *schema {
	for _, ct := range []string{"application/json", "*/*"} {
		if c, ok := content[ct]; ok {
			return c.Schema
		}
	}
	return nil
}

func (g *generator) build(method, url string, op operation, oo OpOverlay) (cmdspec.Op, bool) {
	c := cmdspec.Op{OperationID: op.OperationID, Method: method, URL: url,
		Short: firstLine(op.Summary), Long: strings.TrimSpace(op.Description), Example: strings.TrimSpace(oo.Example)}
	if c.Short == "" {
		c.Short = op.OperationID
	}
	for _, req := range op.Security {
		for _, ss := range req {
			c.Scopes = append(c.Scopes, ss...)
		}
	}
	segs := strings.Split(strings.TrimPrefix(url, "/v1/"), "/")

	// Derived path: nouns, then the verb.
	var nouns []string
	verb := ""
	for i, s := range segs {
		if isParam(s) {
			continue
		}
		if s == "actions" && i+1 < len(segs) {
			verb = segs[i+1]
			break
		}
		nouns = append(nouns, g.noun(s))
	}
	ok200 := op.Responses["200"]
	body200 := g.resolve(jsonSchema(ok200.Content))
	if body200 != nil && strings.HasPrefix(refName(jsonSchema(ok200.Content)), "CursorPage") {
		c.List = true
	}
	if verb == "" {
		last := segs[len(segs)-1]
		switch method {
		case "GET":
			verb = "get"
			if c.List {
				verb = "list"
			}
		case "POST":
			verb = "create"
		case "PATCH":
			verb = "update"
		case "PUT":
			verb = "set"
			if isParam(last) {
				verb = "update"
			}
		case "DELETE":
			verb = "delete"
		}
	}
	c.Path = append(nouns, verb)
	if oo.Path != "" {
		c.Path = strings.Fields(oo.Path)
	}

	// Path parameters: the final noun's is the positional; the rest are flags.
	params := map[string]parameter{}
	for _, p := range op.Parameters {
		params[p.In+":"+p.Name] = p
	}
	final := ""
	if len(c.Path) >= 2 {
		final = c.Path[len(c.Path)-2]
	}
	posName := ""
	for i := 0; i+1 < len(segs); i++ {
		if segs[i] == "actions" {
			break
		}
		if !isParam(segs[i]) && isParam(segs[i+1]) && g.noun(segs[i]) == final {
			posName = strings.Trim(segs[i+1], "{}")
		}
	}
	switch oo.Positional {
	case "":
	case "-":
		posName = ""
	default:
		posName = oo.Positional
	}
	foundPos := posName == ""
	flagNames := map[string]string{}
	claim := func(flag, what string) {
		if prev, ok := flagNames[flag]; ok {
			g.fail("%s: flag --%s is both %s and %s", op.OperationID, flag, prev, what)
		}
		if slices.Contains(reserved, flag) {
			g.fail("%s: flag --%s (%s) collides with a built-in flag; rename it in commands.yaml", op.OperationID, flag, what)
		}
		flagNames[flag] = what
	}
	for i, s := range segs {
		if !isParam(s) {
			continue
		}
		name := strings.Trim(s, "{}")
		prefix := "/v1/" + strings.Join(segs[:i+1], "/")
		po := g.ov.Params[prefix]
		pd := params["path:"+name]
		p := cmdspec.Param{Name: name, Resource: po.Resource, Convert: po.Convert, Help: firstLine(pd.Description)}
		if po.Help != "" {
			p.Help = po.Help
		}
		if pd.Schema != nil && pd.Schema.Pattern != "" {
			if re, err := regexp.Compile(pd.Schema.Pattern); err == nil && re.MatchString("a/b") {
				p.Verbatim = true
			}
		}
		if name == posName {
			c.Positional = p
			foundPos = true
			continue
		}
		if i == 0 {
			g.fail("%s: parameter %s has no noun to name its flag", op.OperationID, name)
			continue
		}
		p.Flag = g.noun(segs[i-1])
		claim(p.Flag, "path parameter "+name)
		c.Params = append(c.Params, p)
	}
	if !foundPos {
		g.fail("%s: positional %q is not a path parameter", op.OperationID, posName)
	}

	usedFlags := map[string]bool{}
	field := func(name string, s *schema, required bool, desc, where string) (cmdspec.Field, bool) {
		rs := g.resolve(s)
		if rs == nil {
			return cmdspec.Field{}, false
		}
		t := typeOf(rs)
		if t != "string" && t != "integer" && t != "number" && t != "boolean" {
			return cmdspec.Field{}, false
		}
		f := cmdspec.Field{Name: name, Flag: Kebab(name), Type: t, Required: required, Help: firstLine(desc)}
		if t == "integer" && rs.Format == "int32" {
			f.Format = "int32"
		}
		if f.Help == "" {
			f.Help = firstLine(rs.Description)
		}
		for _, e := range rs.Enum {
			if e != nil {
				f.Enum = append(f.Enum, fmt.Sprint(e))
			}
		}
		if fo, ok := oo.Flags[name]; ok {
			usedFlags[name] = true
			if fo.Hide {
				return cmdspec.Field{}, false
			}
			if fo.Name != "" {
				f.Flag = fo.Name
			}
			f.Scale = fo.Scale
			if fo.Help != "" {
				f.Help = fo.Help
			}
			if f.Scale != 0 && t != "integer" {
				g.fail("%s: scale on %s, which is not an integer", op.OperationID, name)
			}
		}
		claim(f.Flag, where+" "+name)
		return f, true
	}
	for _, p := range op.Parameters {
		if p.In != "query" {
			continue
		}
		if c.List && (p.Name == "cursor" || p.Name == "limit") {
			c.Paged = true
			continue
		}
		if f, ok := field(p.Name, p.Schema, p.Required, p.Description, "query parameter"); ok {
			c.Query = append(c.Query, f)
		} else {
			g.fail("%s: query parameter %s is not a scalar", op.OperationID, p.Name)
		}
	}
	if op.RequestBody != nil {
		c.HasBody = true
		bs := g.resolve(jsonSchema(op.RequestBody.Content))
		if bs == nil {
			g.fail("%s: the request body is not JSON", op.OperationID)
		} else {
			for _, name := range sortedKeys(bs.Properties) {
				ps := bs.Properties[name]
				if f, ok := field(name, ps, slices.Contains(bs.Required, name), ps.Description, "body field"); ok {
					c.Body = append(c.Body, f)
				} else {
					c.BodyNested = true
				}
			}
		}
	}
	for _, name := range sortedKeys(oo.Flags) {
		if !usedFlags[name] {
			g.fail("%s: commands.yaml flags: %s is not a scalar query or body field", op.OperationID, name)
		}
	}
	for _, f := range oo.AtLeastOne {
		if _, ok := flagNames[f]; !ok {
			g.fail("%s: atLeastOne: no flag --%s", op.OperationID, f)
		}
	}
	c.AtLeastOne = oo.AtLeastOne

	// Async.
	if r, ok := op.Responses["202"]; ok {
		_, loc := r.Headers["Location"]
		switch {
		case oo.Wait == "self":
			if c.Positional.Name == "" {
				g.fail("%s: wait: self needs a positional operation id", op.OperationID)
			}
			c.Async = cmdspec.Self
		case loc || refName(jsonSchema(r.Content)) == "OperationAccepted":
			c.Async = cmdspec.Location
		default:
			g.fail("%s: a 202 with no Location or OperationAccepted; set wait: in commands.yaml", op.OperationID)
		}
	} else if oo.Wait != "" {
		g.fail("%s: wait: set on an operation with no 202", op.OperationID)
	}

	// Confirmation: deletes and protected actions.
	c.Confirm = method == "DELETE" || strings.HasSuffix(url, "/actions/delete")
	for _, s := range c.Scopes {
		if strings.HasPrefix(s, "protected:") {
			c.Confirm = true
		}
	}

	// Columns.
	for _, code := range []string{"200", "201"} {
		r, ok := op.Responses[code]
		if !ok {
			continue
		}
		s := jsonSchema(r.Content)
		name := refName(s)
		if c.List {
			name = refName(g.resolve(s).Properties["data"].Items)
		}
		if name == "" {
			continue
		}
		if _, ok := g.ov.Columns[name]; ok {
			c.Columns = name
		} else if method == "GET" {
			g.fail("%s: no column set for %s in commands.yaml", op.OperationID, name)
		}
	}
	return c, true
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// Kebab turns a camelCase name into kebab-case.
func Kebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('-')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

func sortedKeys[K interface{ ~string | ~int }, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// Source renders the table as Go source in package pkg.
func Source(out *Output, pkg string) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by internal/gen from the SDK's openapi.json and commands.yaml; DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package %s\n\nimport \"github.com/CloudAfrica/cloudafrica-cli/internal/cmdspec\"\n\n", pkg)
	fmt.Fprintf(&b, "var genOps = []cmdspec.Op{\n")
	for _, op := range out.Ops {
		fmt.Fprintf(&b, "%#v,\n", op)
	}
	fmt.Fprintf(&b, "}\n\nvar genGroups = []cmdspec.Group{\n")
	for _, g := range out.Groups {
		fmt.Fprintf(&b, "%#v,\n", g)
	}
	fmt.Fprintf(&b, "}\n\nvar genColumns = map[string]cmdspec.ColumnSet{\n")
	for _, k := range sortedKeys(out.Columns) {
		fmt.Fprintf(&b, "%q: %#v,\n", k, out.Columns[k])
	}
	fmt.Fprintf(&b, "}\n\nvar genResources = map[string]cmdspec.Resource{\n")
	for _, k := range sortedKeys(out.Resources) {
		fmt.Fprintf(&b, "%q: %#v,\n", k, out.Resources[k])
	}
	fmt.Fprintf(&b, "}\n\nvar genScopes = %#v\n", out.Scopes)
	src, err := format.Source(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting the generated table: %w", err)
	}
	return src, nil
}
