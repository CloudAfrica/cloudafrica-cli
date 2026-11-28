package gen

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/CloudAfrica/cloudafrica-cli/internal/cmdspec"
)

// fixtureDoc is a small openapi.json: a list, a get, a create, an update, a
// delete, an action, a sub-resource, a 202 and an excluded and pending path.
func fixtureDoc() map[string]any {
	ref := func(n string) map[string]any { return map[string]any{"$ref": "#/components/schemas/" + n} }
	content := func(s any) map[string]any {
		return map[string]any{"application/json": map[string]any{"schema": s}}
	}
	ok := func(s any) map[string]any { return map[string]any{"200": map[string]any{"content": content(s)}} }
	sec := func(s ...string) []any { return []any{map[string]any{"oauth": s}} }
	pathID := func(name string) map[string]any {
		return map[string]any{"name": name, "in": "path", "required": true, "schema": map[string]any{"type": "string"}}
	}
	accepted := map[string]any{"202": map[string]any{"headers": map[string]any{"Location": map[string]any{}}}}
	return map[string]any{
		"paths": map[string]any{
			"/v1/widgets": map[string]any{
				"get": map[string]any{"operationId": "listWidgets", "summary": "List widgets\nmore", "security": sec("read:w"),
					"parameters": []any{
						map[string]any{"name": "cursor", "in": "query", "schema": map[string]any{"type": "string"}},
						map[string]any{"name": "limit", "in": "query", "description": "Page size, a whole number from 1 to 50; defaults to 50.", "schema": map[string]any{"type": "integer"}},
						map[string]any{"name": "colour", "in": "query", "schema": map[string]any{"type": "string", "enum": []any{"red", "blue"}}},
					},
					"responses": ok(ref("CursorPageWidget"))},
				"post": map[string]any{"operationId": "createWidget", "security": sec("write:w"),
					"requestBody": map[string]any{"content": content(ref("CreateWidget"))},
					"responses":   map[string]any{"201": map[string]any{"content": content(ref("Widget"))}}},
			},
			"/v1/widgets/{id}": map[string]any{
				"get":    map[string]any{"operationId": "getWidget", "parameters": []any{pathID("id")}, "responses": ok(ref("Widget"))},
				"patch":  map[string]any{"operationId": "updateWidget", "parameters": []any{pathID("id")}, "responses": ok(ref("Widget"))},
				"delete": map[string]any{"operationId": "deleteWidget", "parameters": []any{pathID("id")}, "responses": accepted},
			},
			"/v1/widgets/{id}/actions/spin": map[string]any{
				"post": map[string]any{"operationId": "spinWidget", "security": sec("protected:w"),
					"parameters": []any{pathID("id")}, "responses": accepted},
			},
			"/v1/widgets/{id}/gears/{gearId}": map[string]any{
				"put": map[string]any{"operationId": "setWidgetGear", "parameters": []any{pathID("id"), pathID("gearId")},
					"responses": map[string]any{"204": map[string]any{}}},
			},
			"/v1/jobs/{id}": map[string]any{
				"get": map[string]any{"operationId": "getJob", "parameters": []any{
					map[string]any{"name": "id", "in": "path", "schema": map[string]any{"type": "string", "pattern": "^[a-z/]+$"}}},
					"responses": ok(ref("Widget"))},
			},
			"/v1/secret": map[string]any{
				"get": map[string]any{"operationId": "getSecret", "responses": ok(ref("Widget"))},
			},
			"/v1/later": map[string]any{
				"get": map[string]any{"operationId": "listLater", "security": sec("read:later"), "responses": ok(ref("Widget"))},
			},
		},
		"components": map[string]any{"schemas": map[string]any{
			"Widget": map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}},
			"CursorPageWidget": map[string]any{"type": "object", "properties": map[string]any{
				"data": map[string]any{"type": "array", "items": ref("Widget")}}},
			"CreateWidget": map[string]any{"type": "object", "required": []any{"name"}, "properties": map[string]any{
				"name":   map[string]any{"type": "string"},
				"ramMib": map[string]any{"type": "integer", "format": "int32"},
				"tags":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}},
		}},
	}
}

const fixtureOverlay = `
groups:
  widgets: Manage widgets
sections:
  widgets: compute
  jobs: cli
resources:
  widget: {noun: widget, list: /v1/widgets, match: name}
params:
  /v1/widgets/{id}: {resource: widget}
exclude:
  - {operation: getSecret, reason: not for the CLI}
pending:
  - {prefix: /v1/later, slice: 2}
columns:
  Widget:
    columns:
      - {header: ID, path: id}
operations:
  createWidget:
    flags:
      ramMib: {name: ram-gib, scale: 1024}
`

// at walks nested objects of a fixture document.
func at(m map[string]any, keys ...string) map[string]any {
	for _, k := range keys {
		next, ok := m[k].(map[string]any)
		if !ok {
			panic("fixture has no object at " + k)
		}
		m = next
	}
	return m
}

func generate(t *testing.T, d map[string]any, overlay string) (*Output, Report, error) {
	t.Helper()
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return Generate(b, []byte(overlay))
}

func opByID(t *testing.T, out *Output, id string) cmdspec.Op {
	t.Helper()
	for _, op := range out.Ops {
		if op.OperationID == id {
			return op
		}
	}
	t.Fatalf("no op %s", id)
	return cmdspec.Op{}
}

func TestGenerateDerivesTheFixture(t *testing.T) {
	out, rep, err := generate(t, fixtureDoc(), fixtureOverlay)
	if err != nil {
		t.Fatal(err)
	}
	if rep.String() != "8 generated, 0 hand-written, 1 excluded, 1 pending slice 2" {
		t.Errorf("report = %s", rep)
	}
	paths := map[string]string{}
	for _, op := range out.Ops {
		paths[op.OperationID] = strings.Join(op.Path, " ")
	}
	want := map[string]string{
		"listWidgets": "widgets list", "createWidget": "widgets create", "getWidget": "widgets get",
		"updateWidget": "widgets update", "deleteWidget": "widgets delete", "spinWidget": "widgets spin",
		"setWidgetGear": "widgets gears update", "getJob": "jobs get",
	}
	for id, p := range want {
		if paths[id] != p {
			t.Errorf("%s: path %q, want %q", id, paths[id], p)
		}
	}

	list := opByID(t, out, "listWidgets")
	if list.PageLimit != 50 {
		t.Errorf("PageLimit = %d, want the 50 the limit description names", list.PageLimit)
	}
	if !list.List || !list.Paged || list.Short != "List widgets" || list.Columns != "Widget" ||
		len(list.Query) != 1 || list.Query[0].Flag != "colour" || !reflect.DeepEqual(list.Query[0].Enum, []string{"red", "blue"}) {
		t.Errorf("list = %+v", list)
	}
	create := opByID(t, out, "createWidget")
	if !create.HasBody || !create.BodyNested || len(create.Body) != 2 {
		t.Fatalf("create = %+v", create)
	}
	if f := create.Body[1]; f.Name != "ramMib" || f.Flag != "ram-gib" || f.Scale != 1024 || f.Format != "int32" {
		t.Errorf("ramMib = %+v", f)
	}
	hidden, _, err := generate(t, fixtureDoc(), strings.Replace(fixtureOverlay, "    flags:\n", "    flags:\n      name: {hide: true}\n", 1))
	if err != nil {
		t.Fatal(err)
	}
	if b := opByID(t, hidden, "createWidget").Body; len(b) != 1 || b[0].Name != "ramMib" {
		t.Errorf("hidden field still a flag: %+v", b)
	}
	if f := create.Body[0]; f.Name != "name" || !f.Required {
		t.Errorf("name = %+v", f)
	}
	del := opByID(t, out, "deleteWidget")
	if !del.Confirm || del.Async != cmdspec.Location || del.Positional.Name != "id" || del.Positional.Resource != "widget" {
		t.Errorf("delete = %+v", del)
	}
	spin := opByID(t, out, "spinWidget")
	if !spin.Confirm || spin.Async != cmdspec.Location {
		t.Errorf("a protected action does not confirm: %+v", spin)
	}
	if get := opByID(t, out, "getWidget"); get.Confirm || get.Async != cmdspec.NotAsync {
		t.Errorf("get = %+v", get)
	}
	gear := opByID(t, out, "setWidgetGear")
	if gear.Positional.Name != "gearId" || len(gear.Params) != 1 || gear.Params[0].Flag != "widget" || gear.Params[0].Name != "id" {
		t.Errorf("gear = %+v", gear)
	}
	if job := opByID(t, out, "getJob"); !job.Positional.Verbatim {
		t.Errorf("a slashed id is escaped: %+v", job.Positional)
	}
	if !reflect.DeepEqual(out.Scopes, []string{"read:later", "read:w", "write:w"}) {
		t.Errorf("scopes = %v (pending operations' count, protected: ones do not)", out.Scopes)
	}
	wantGroups := []cmdspec.Group{
		{Path: []string{"jobs"}, Short: "Manage jobs", Aliases: []string{"job"}, GroupID: "cli"},
		{Path: []string{"widgets"}, Short: "Manage widgets", Aliases: []string{"widget"}, GroupID: "compute"},
		{Path: []string{"widgets", "gears"}, Short: "Manage gears", Aliases: []string{"gear"}},
	}
	if !reflect.DeepEqual(out.Groups, wantGroups) {
		t.Errorf("groups = %+v\nwant %+v", out.Groups, wantGroups)
	}
	if job := opByID(t, out, "getJob"); job.Positional.Name != "id" {
		t.Errorf("a plural noun lost its positional: %+v", job)
	}

	replaced, _, err := generate(t, fixtureDoc(), fixtureOverlay+"aliases:\n  widgets gears: []\n  jobs: [task, work]\n")
	if err != nil {
		t.Fatal(err)
	}
	if g := replaced.Groups; g[0].Aliases[0] != "task" || len(g[0].Aliases) != 2 || len(g[2].Aliases) != 0 {
		t.Errorf("aliases: did not replace the singular: %+v", g)
	}
}

// An array of strings in the query becomes one repeatable flag; the same array
// in a body stays a --data field.
func TestGenerateStringListQuery(t *testing.T) {
	d := fixtureDoc()
	g := at(d, "paths", "/v1/widgets", "get")
	params, _ := g["parameters"].([]any)
	g["parameters"] = append(params, map[string]any{"name": "tags", "in": "query", "description": "Tags to include.",
		"schema": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}})
	out, _, err := generate(t, d, fixtureOverlay)
	if err != nil {
		t.Fatal(err)
	}
	q := opByID(t, out, "listWidgets").Query
	if len(q) != 2 || q[1].Name != "tags" || q[1].Flag != "tags" || q[1].Type != "stringList" || q[1].Help != "Tags to include." {
		t.Errorf("query = %+v", q)
	}

	d = fixtureDoc()
	props := at(d, "components", "schemas", "CreateWidget", "properties")
	props["tags"] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	out, _, err = generate(t, d, fixtureOverlay)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range opByID(t, out, "createWidget").Body {
		if f.Name == "tags" {
			t.Errorf("a body array became a flag: %+v", f)
		}
	}
}

// short: and long: replace the API's summary and description; an operation without them keeps the API's.
func TestGenerateHelpOverrides(t *testing.T) {
	overlay := strings.Replace(fixtureOverlay, "  createWidget:\n",
		"  createWidget:\n    short: Make a widget\n    long: >-\n      Makes a widget,\n      in ca's words.\n", 1)
	out, _, err := generate(t, fixtureDoc(), overlay)
	if err != nil {
		t.Fatal(err)
	}
	c := opByID(t, out, "createWidget")
	if c.Short != "Make a widget" || c.Long != "Makes a widget, in ca's words." {
		t.Errorf("createWidget help = %q / %q", c.Short, c.Long)
	}
	plain, _, err := generate(t, fixtureDoc(), fixtureOverlay)
	if err != nil {
		t.Fatal(err)
	}
	want := opByID(t, plain, "listWidgets")
	if got := opByID(t, out, "listWidgets"); got.Short != want.Short || got.Long != want.Long {
		t.Errorf("listWidgets help changed: %q / %q", got.Short, got.Long)
	}
}

func TestGenerateRefuses(t *testing.T) {
	for name, c := range map[string]struct {
		doc     func(d map[string]any)
		overlay func(string) string
		want    string
	}{
		"uncovered operation with no column set": {
			overlay: func(o string) string {
				return strings.Replace(o, "  - {operation: getSecret, reason: not for the CLI}\n", "", 1)
			},
			// getSecret now generates; Widget has columns, so it is fine — use a schema without one.
			doc: func(d map[string]any) {
				at(d, "paths", "/v1/secret", "get")["responses"] =
					map[string]any{"200": map[string]any{"content": map[string]any{"application/json": map[string]any{
						"schema": map[string]any{"$ref": "#/components/schemas/CreateWidget"}}}}}
			},
			want: "getSecret: no column set for CreateWidget",
		},
		"get with an inline response schema": {
			overlay: func(o string) string {
				return strings.Replace(o, "  - {operation: getSecret, reason: not for the CLI}\n", "", 1)
			},
			doc: func(d map[string]any) {
				at(d, "paths", "/v1/secret", "get")["responses"] =
					map[string]any{"200": map[string]any{"content": map[string]any{"application/json": map[string]any{
						"schema": map[string]any{"type": "object"}}}}}
			},
			want: "getSecret: the 200 response schema is inline",
		},
		"duplicate command": {
			overlay: func(o string) string {
				return o + "  getJob:\n    path: widgets get\n"
			},
			want: `command "widgets get": both`,
		},
		"leaf that is a group": {
			overlay: func(o string) string { return o + "  getJob:\n    path: widgets gears\n" },
			want:    `command "widgets gears" (getJob) is also a group`,
		},
		"stale operation": {
			overlay: func(o string) string { return o + "  goneOp:\n    path: x y\n" },
			want:    "commands.yaml operations: goneOp is not in openapi.json",
		},
		"stale handwritten": {
			overlay: func(o string) string { return o + "handwritten:\n  goneOp: x y\n" },
			want:    "commands.yaml handwritten: goneOp is not in openapi.json",
		},
		"stale rule": {
			overlay: func(o string) string { return strings.Replace(o, "prefix: /v1/later", "prefix: /v1/gone", 1) },
			want:    "commands.yaml: no path starts with /v1/gone",
		},
		"stale columns": {
			overlay: func(o string) string {
				return strings.Replace(o, "columns:\n  Widget:", "columns:\n  Gone:\n    columns: [{header: X, path: x}]\n  Widget:", 1)
			},
			want: "commands.yaml columns: schema Gone is not in openapi.json",
		},
		"stale param": {
			overlay: func(o string) string { return strings.Replace(o, "params:\n", "params:\n  /v1/gone/{id}: {}\n", 1) },
			want:    "commands.yaml params: no path starts with /v1/gone/{id}",
		},
		"within without a resource": {
			overlay: func(o string) string {
				return strings.Replace(o, "params:\n", "params:\n  /v1/widgets/{id}/gears/{gearId}: {within: widgetId}\n", 1)
			},
			want: "commands.yaml params /v1/widgets/{id}/gears/{gearId}: within needs a resource",
		},
		"within on the first path parameter": {
			overlay: func(o string) string {
				return strings.Replace(o, "{resource: widget}", "{resource: widget, within: ownerId}", 1)
			},
			want: "parameter id is within ownerId but follows no path parameter",
		},
		"param naming no resource": {
			overlay: func(o string) string { return strings.Replace(o, "{resource: widget}", "{resource: nope}", 1) },
			want:    `no resource "nope"`,
		},
		"param naming no converter": {
			overlay: func(o string) string {
				return strings.Replace(o, "{resource: widget}", "{resource: widget, convert: epoc}", 1)
			},
			want: `commands.yaml params /v1/widgets/{id}: no converter "epoc"`,
		},
		"stale noun": {
			overlay: func(o string) string { return strings.Replace(o, "groups:\n", "nouns:\n  gone: g\ngroups:\n", 1) },
			want:    "commands.yaml nouns: no path has a segment gone",
		},
		"path left singular beside its plural group": {
			overlay: func(o string) string {
				return strings.Replace(o, "  jobs: cli\n", "  jobs: cli\n  widget: cli\n", 1) + "  getJob:\n    path: widget get\n"
			},
			want: `"widget" answers to both group widget and group widgets`,
		},
		"alias that is a sibling's name": {
			overlay: func(o string) string { return o + "aliases:\n  widgets: [jobs]\n" },
			want:    `"jobs" answers to both group jobs and group widgets`,
		},
		"alias that is a sibling command's name": {
			overlay: func(o string) string { return o + "aliases:\n  widgets gears: [spin]\n" },
			want:    `"widgets spin" answers to both command widgets spin and group widgets gears`,
		},
		"stale alias": {
			overlay: func(o string) string { return o + "aliases:\n  gone: []\n" },
			want:    `commands.yaml aliases: "gone" is not a group`,
		},
		"top-level group with no section": {
			overlay: func(o string) string { return strings.Replace(o, "  jobs: cli\n", "", 1) },
			want:    `commands.yaml sections: top-level group "jobs" has no section`,
		},
		"section naming no heading": {
			overlay: func(o string) string { return strings.Replace(o, "  jobs: cli\n", "  jobs: clu\n", 1) },
			want:    `commands.yaml sections jobs: no section "clu"`,
		},
		"section for a subgroup": {
			overlay: func(o string) string {
				return strings.Replace(o, "  jobs: cli\n", "  jobs: cli\n  widgets gears: cli\n", 1)
			},
			want: `commands.yaml sections: "widgets gears" is not a top-level group`,
		},
		"stale group": {
			overlay: func(o string) string { return strings.Replace(o, "groups:\n", "groups:\n  gone: Gone\n", 1) },
			want:    `commands.yaml groups: "gone" is not a group`,
		},
		"flag collision with built-in": {
			overlay: func(o string) string { return strings.Replace(o, "{name: ram-gib, scale: 1024}", "{name: timeout}", 1) },
			want:    "createWidget: flag --timeout (body field ramMib) collides with a built-in flag",
		},
		"flag collision between fields": {
			overlay: func(o string) string { return strings.Replace(o, "{name: ram-gib, scale: 1024}", "{name: name}", 1) },
			want:    "createWidget: flag --name is both",
		},
		"scale on a string": {
			overlay: func(o string) string {
				return strings.Replace(o, "ramMib: {name: ram-gib, scale: 1024}", "ramMib: {name: ram-gib}\n      name: {scale: 2}", 1)
			},
			want: "createWidget: scale on name, which is not an integer",
		},
		"flag override for a missing field": {
			overlay: func(o string) string { return strings.Replace(o, "ramMib:", "ramGib:", 1) },
			want:    "createWidget: commands.yaml flags: ramGib is not a scalar query or body field",
		},
		"bad positional": {
			overlay: func(o string) string { return o + "  getWidget:\n    positional: nope\n" },
			want:    `getWidget: positional "nope" is not a path parameter`,
		},
		"bad atLeastOne": {
			overlay: func(o string) string {
				return strings.Replace(o, "      ramMib:", "    atLeastOne: [nope]\n    flags:\n      ramMib:", 1)
			},
			want: "createWidget: atLeastOne: no flag --nope",
		},
		"bad exclusive flag": {
			overlay: func(o string) string {
				return strings.Replace(o, "      ramMib:", "    exclusive: [[name], [nope]]\n    flags:\n      ramMib:", 1)
			},
			want: "createWidget: exclusive: no flag --nope",
		},
		"single exclusive group": {
			overlay: func(o string) string {
				return strings.Replace(o, "      ramMib:", "    exclusive: [[name]]\n    flags:\n      ramMib:", 1)
			},
			want: "createWidget: exclusive needs at least two groups",
		},
		"202 without Location": {
			doc: func(d map[string]any) {
				at(d, "paths", "/v1/widgets/{id}", "delete")["responses"] =
					map[string]any{"202": map[string]any{}}
			},
			want: "deleteWidget: a 202 with no Location or OperationAccepted",
		},
		"wait on an operation with no 202": {
			overlay: func(o string) string { return o + "  getWidget:\n    wait: self\n" },
			want:    "getWidget: wait: set on an operation with no 202",
		},
		"non-scalar query parameter": {
			doc: func(d map[string]any) {
				g := at(d, "paths", "/v1/widgets", "get")
				params, _ := g["parameters"].([]any)
				g["parameters"] = append(params, map[string]any{"name": "ids", "in": "query",
					"schema": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}})
			},
			want: "listWidgets: query parameter ids is not a scalar",
		},
		"enum list query parameter": {
			doc: func(d map[string]any) {
				g := at(d, "paths", "/v1/widgets", "get")
				params, _ := g["parameters"].([]any)
				g["parameters"] = append(params, map[string]any{"name": "ids", "in": "query",
					"schema": map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []any{"a"}}}})
			},
			want: "listWidgets: query parameter ids is not a scalar",
		},
		"exclusion without a reason": {
			overlay: func(o string) string { return strings.Replace(o, ", reason: not for the CLI", "", 1) },
			want:    "getSecret: an exclusion needs a reason",
		},
		"pending without a slice": {
			overlay: func(o string) string { return strings.Replace(o, ", slice: 2", "", 1) },
			want:    "listLater: a pending entry needs its slice",
		},
		"unknown overlay key": {
			overlay: func(o string) string { return o + "bogus: 1\n" },
			want:    `unknown field "bogus"`,
		},
		"column set no command uses": {
			overlay: func(o string) string {
				return strings.Replace(o, "columns:\n  Widget:", "columns:\n  CreateWidget:\n    columns: [{header: X, path: x}]\n  Widget:", 1)
			},
			want: "commands.yaml columns: CreateWidget is used by no command",
		},
		"resource with a bad idPattern": {
			overlay: func(o string) string { return strings.Replace(o, "match: name}", "match: name, idPattern: \"(\"}", 1) },
			want:    "commands.yaml resources widget: idPattern:",
		},
		"resource listing a missing path": {
			overlay: func(o string) string { return strings.Replace(o, "list: /v1/widgets,", "list: /v1/gone,", 1) },
			want:    "commands.yaml resources widget: list /v1/gone is not in openapi.json",
		},
		"resource with no noun": {
			overlay: func(o string) string { return strings.Replace(o, "{noun: widget, ", "{", 1) },
			want:    "commands.yaml resources widget: noun and match are required",
		},
		"resource with no match": {
			overlay: func(o string) string { return strings.Replace(o, ", match: name}", "}", 1) },
			want:    "commands.yaml resources widget: noun and match are required",
		},
		"new operation, uncovered": {
			doc: func(d map[string]any) {
				at(d, "paths", "/v1/widgets/{id}/actions/spin", "post")["operationId"] = "spinWidget"
				at(d, "paths")["/v1/gadgets"] = map[string]any{"get": map[string]any{"operationId": "listGadgets",
					"responses": map[string]any{"200": map[string]any{"content": map[string]any{"application/json": map[string]any{
						"schema": map[string]any{"$ref": "#/components/schemas/CreateWidget"}}}}}}}
			},
			want: "listGadgets: no column set for CreateWidget",
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, o := fixtureDoc(), fixtureOverlay
			if c.doc != nil {
				c.doc(d)
			}
			if c.overlay != nil {
				o = c.overlay(o)
			}
			_, _, err := generate(t, d, o)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v\nwant %q", err, c.want)
			}
		})
	}
}

func TestSourceIsDeterministic(t *testing.T) {
	var first []byte
	for i := 0; i < 5; i++ {
		out, _, err := generate(t, fixtureDoc(), fixtureOverlay)
		if err != nil {
			t.Fatal(err)
		}
		src, err := Source(out, "cli")
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = src
		} else if !bytes.Equal(first, src) {
			t.Fatal("two runs produced different source")
		}
	}
}

// The committed table is what the SDK's document and commands.yaml generate.
func TestCommittedTableIsCurrent(t *testing.T) {
	dir, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/CloudAfrica/cloudafrica-go").Output()
	if err != nil {
		t.Skipf("locating the SDK: %v", err)
	}
	docJSON, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(dir)), "openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	ov, err := os.ReadFile("../cli/commands.yaml")
	if err != nil {
		t.Fatal(err)
	}
	out, rep, err := Generate(docJSON, ov)
	if err != nil {
		t.Fatalf("%v\n(%s)", err, rep)
	}
	src, err := Source(out, "cli")
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile("../cli/commands_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src, committed) {
		t.Error("internal/cli/commands_gen.go is stale: run go generate ./...")
	}
}

func TestKebab(t *testing.T) {
	for in, want := range map[string]string{"ramMib": "ram-mib", "name": "name", "datacenterSlug": "datacenter-slug"} {
		if got := Kebab(in); got != want {
			t.Errorf("Kebab(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScopesCoverHandwrittenAndPendingButNotExcluded(t *testing.T) {
	d := fixtureDoc()
	at(d, "paths", "/v1/widgets/{id}/gears/{gearId}", "put")["security"] =
		[]any{map[string]any{"oauth": []string{"write:gear"}}}
	at(d, "paths", "/v1/secret", "get")["security"] =
		[]any{map[string]any{"oauth": []string{"read:secret"}}}
	out, rep, err := generate(t, d, fixtureOverlay+"handwritten:\n  setWidgetGear: widget gear set\n")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Handwritten != 1 || rep.Generated != 7 {
		t.Errorf("report = %s", rep)
	}
	for _, op := range out.Ops {
		if op.OperationID == "setWidgetGear" {
			t.Error("a hand-written operation was also generated")
		}
	}
	has := func(s string) bool { return slices.Contains(out.Scopes, s) }
	if !has("write:gear") || !has("read:later") || !has("read:w") || has("read:secret") {
		t.Errorf("scopes = %v", out.Scopes)
	}
}

// cliVoice removes each API-only sentence exactly, wherever it sits, and keeps a longer sentence that
// starts the same way.
func TestCliVoiceRemovesExactSentences(t *testing.T) {
	const async = "Asynchronous: returns 202 with the operation to poll at Location."
	const idem = "Must carry an Idempotency-Key header."
	for in, want := range map[string]string{
		async:                               "",
		async + " Refuses with 404.":        "Refuses with 404.",
		"Requires x. " + idem + " " + async: "Requires x.",
		"Requires x. " + idem + " Then y.":  "Requires x. Then y.",
		"Must carry an Idempotency-Key header: a retry replays.":                               "Must carry an Idempotency-Key header: a retry replays.",
		"Asynchronous: returns 202 with the operation to poll at Location, and 409 when busy.": "Asynchronous: returns 202 with the operation to poll at Location, and 409 when busy.",
		"Lists. Cursor-paginated: follow nextCursor via `?cursor=`.":                           "Lists.",
	} {
		if got := cliVoice(in); got != want {
			t.Errorf("cliVoice(%q) = %q, want %q", in, got, want)
		}
	}
}
