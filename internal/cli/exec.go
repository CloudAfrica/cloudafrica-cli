package cli

//go:generate go run ../gen/cagen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/cmdspec"
	"github.com/CloudAfrica/cloudafrica-cli/internal/output"
)

// conflictHints add a remedy to a 409 from one operation; %[1]s is the positional value.
var conflictHints = map[string]string{
	"deleteVm": "If the VM is running, stop it with `ca vm shutdown %[1]s` (or `ca vm force-stop %[1]s`), " +
		"then delete it; otherwise the message above gives the reason.",
}

// addGenerated adds the generated commands to root, inside any hand-written
// group of the same name.
func (a *App) addGenerated(root *cobra.Command) {
	for _, g := range genGroups {
		parent := ensureGroup(root, g.Path[:len(g.Path)-1])
		if c := child(parent, g.Path[len(g.Path)-1]); c != nil {
			c.Short = g.Short
			continue
		}
		parent.AddCommand(group(g.Path[len(g.Path)-1], g.Short))
	}
	for _, op := range genOps {
		parent := ensureGroup(root, op.Path[:len(op.Path)-1])
		if child(parent, op.Path[len(op.Path)-1]) != nil {
			panic(fmt.Sprintf("generated command %q clashes with a hand-written one", strings.Join(op.Path, " ")))
		}
		parent.AddCommand(a.genCommand(op))
	}
}

func child(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func ensureGroup(root *cobra.Command, path []string) *cobra.Command {
	cur := root
	for _, name := range path {
		next := child(cur, name)
		if next == nil {
			next = group(name, "")
			cur.AddCommand(next)
		}
		cur = next
	}
	return cur
}

// argName is how a positional shows in usage: VM, SSH_KEY, OPERATION_ID, RULE_ID.
func argName(op cmdspec.Op) string {
	p := op.Positional
	if p.Resource != "" {
		return strings.ToUpper(strings.ReplaceAll(p.Resource, "-", "_"))
	}
	if p.Name == "id" {
		return strings.ToUpper(strings.ReplaceAll(op.Path[len(op.Path)-2], "-", "_")) + "_ID"
	}
	return strings.ToUpper(strings.ReplaceAll(kebab(p.Name), "-", "_"))
}

func kebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// genRun holds one generated command's executor flags.
type genRun struct {
	yes     bool
	noWait  bool
	timeout time.Duration
	data    string
	limit   int
	cursor  string
}

func (a *App) genCommand(op cmdspec.Op) *cobra.Command {
	var r genRun
	use := op.Path[len(op.Path)-1]
	nargs := 0
	if op.Positional.Name != "" {
		use += " " + argName(op)
		nargs = 1
	}
	cmd := &cobra.Command{
		Use:     use,
		Short:   op.Short,
		Long:    longHelp(op),
		Example: indent(op.Example),
		Args:    exactArgs(nargs, strings.TrimPrefix(use, op.Path[len(op.Path)-1]+" ")),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runGenerated(cmd, op, r, args)
		},
	}
	f := cmd.Flags()
	for _, p := range op.Params {
		f.String(p.Flag, "", paramHelp(p))
	}
	for _, fl := range slices.Concat(op.Query, op.Body) {
		help := fl.Help
		if len(fl.Enum) > 0 {
			help = strings.TrimSpace(help + " (one of: " + strings.Join(fl.Enum, ", ") + ")")
		}
		if fl.Required {
			help = strings.TrimSpace(help + " (required)")
		}
		switch fl.Type {
		case "integer":
			f.Int64(fl.Flag, 0, help)
		case "number":
			f.Float64(fl.Flag, 0, help)
		case "boolean":
			f.Bool(fl.Flag, false, help)
		default:
			f.String(fl.Flag, "", help)
		}
	}
	if op.HasBody {
		f.StringVarP(&r.data, "data", "d", "", "the JSON request body: a literal, @file or @- (stdin); flags override its fields")
	}
	if op.Confirm {
		f.BoolVarP(&r.yes, "yes", "y", false, "do not ask for confirmation")
	}
	if op.Async != cmdspec.NotAsync {
		f.BoolVar(&r.noWait, "no-wait", false, "print the operation and return without waiting for it")
		f.DurationVar(&r.timeout, "timeout", defaultWaitTimeout, "how long to wait for the operation")
	}
	if op.List && op.Paged {
		f.IntVar(&r.limit, "limit", 0, "fetch one page of at most this many rows and print it as returned")
		f.StringVar(&r.cursor, "cursor", "", "with --limit, the page to fetch (the previous page's nextCursor)")
	}
	return cmd
}

func longHelp(op cmdspec.Op) string {
	long := op.Long
	if long == "" {
		long = op.Short
	}
	if op.BodyNested {
		long += "\n\nSome fields of the request body have no flag; set them with --data."
	}
	return long
}

func indent(s string) string {
	if s == "" {
		return ""
	}
	return "  " + strings.ReplaceAll(strings.TrimSpace(s), "\n", "\n  ")
}

func paramHelp(p cmdspec.Param) string {
	h := p.Help
	if h == "" {
		h = "the " + p.Flag
		if p.Resource != "" {
			h += " id or name"
		} else {
			h += " id"
		}
	}
	return h + " (required)"
}

// target is a path parameter's given value and the id it resolved to.
type target struct {
	param  cmdspec.Param
	given  string
	id     string
	byName bool
}

func (a *App) runGenerated(cmd *cobra.Command, op cmdspec.Op, r genRun, args []string) error {
	ctx := cmd.Context()
	f := cmd.Flags()
	pr, err := a.printer()
	if err != nil {
		return err
	}

	// 1. Validate every input before any request.
	var targets []target
	if op.Positional.Name != "" {
		targets = append(targets, target{param: op.Positional, given: args[0]})
	}
	for _, p := range op.Params {
		v, _ := f.GetString(p.Flag)
		if strings.TrimSpace(v) == "" {
			return usageErr(fmt.Errorf("--%s is required", p.Flag))
		}
		targets = append(targets, target{param: p, given: v})
	}
	for i := range targets {
		v, err := convertParam(targets[i].param, targets[i].given)
		if err != nil {
			return usageErr(err)
		}
		targets[i].id = v
	}
	query := url.Values{}
	for _, fl := range op.Query {
		v, set, err := fieldValue(f, fl)
		if err != nil {
			return usageErr(err)
		}
		if !set {
			if fl.Required {
				return usageErr(fmt.Errorf("--%s is required", fl.Flag))
			}
			continue
		}
		query.Set(fl.Name, fmt.Sprint(v))
	}
	var body map[string]any
	if op.HasBody {
		if body, err = a.bodyFrom(op, f, r.data); err != nil {
			return usageErr(err)
		}
	}
	if len(op.AtLeastOne) > 0 && !atLeastOneGiven(op, f, body) {
		return usageErr(fmt.Errorf("give %s", orList(op.AtLeastOne)))
	}
	if op.List && r.cursor != "" && r.limit == 0 {
		return usageErr(errors.New("--cursor needs --limit"))
	}
	if r.limit < 0 || r.limit == 0 && f.Changed("limit") {
		return usageErr(errors.New("--limit must be at least 1"))
	}
	if op.Confirm && !r.yes && !a.TTY {
		return usageErr(fmt.Errorf("refusing to %s without --yes when not on a terminal", strings.Join(op.Path, " ")))
	}

	// 2. Resolve names (reads only), then confirm once, outside any retried closure.
	if slices.ContainsFunc(targets, func(t target) bool { return t.param.Resource != "" }) {
		err := a.withSession(ctx, func(s *session) error {
			// A parameter within another resolves after it, so it can be checked against its id.
			for _, within := range []bool{false, true} {
				for i := range targets {
					p := targets[i].param
					if p.Resource == "" || (p.Within != "") != within {
						continue
					}
					var parent *target
					if within {
						parent = &targets[slices.IndexFunc(targets, func(t target) bool { return t.param.Name == p.WithinParam })]
					}
					id, byName, err := a.resolveParam(ctx, s, p, targets[i].given, parent)
					if err != nil {
						return err
					}
					targets[i].id, targets[i].byName = id, byName
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if op.Confirm && !r.yes {
		if !a.confirm(confirmQuestion(op, targets)) {
			return errors.New("cancelled")
		}
	}

	// 3. Send.
	path := buildPath(op.URL, targets)
	if op.List {
		return a.withSession(ctx, func(s *session) error { return a.runList(ctx, s, pr, op, path, query, r) })
	}
	var payload []byte
	if op.HasBody && (len(body) > 0 || op.Method != http.MethodDelete) {
		if body == nil {
			body = map[string]any{}
		}
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	var (
		status int
		resp   []byte
		loc    string
	)
	sendCtx := ctx
	if unrepeatable[op.OperationID] != "" {
		sendCtx = singleShot(ctx)
	}
	err = a.withSession(ctx, func(s *session) error {
		var err error
		status, resp, loc, err = send(sendCtx, s, op.Method, path, query, payload)
		if err != nil && cloudafrica.IsSlug(err, "conflict") && conflictHints[op.OperationID] != "" && len(targets) > 0 {
			return &hintedError{err: err, hint: fmt.Sprintf(conflictHints[op.OperationID], targets[0].given)}
		}
		if errors.Is(err, errNotRetried) && len(targets) > 0 {
			return &hintedError{err: errNotRetried, hint: fmt.Sprintf(unrepeatable[op.OperationID], targets[0].given)}
		}
		return err
	})
	if err != nil {
		return err
	}

	// 4. Wait, or print.
	if status == http.StatusAccepted && op.Async != cmdspec.NotAsync {
		opID := ""
		if op.Async == cmdspec.Self {
			opID = targets[0].id
		} else {
			opID = acceptedID(resp, loc)
		}
		if opID == "" {
			return errors.New("the API accepted the request but returned no operation id")
		}
		return a.withSession(ctx, func(s *session) error {
			return a.accepted(ctx, s, opID, waitFlags{noWait: r.noWait, timeout: r.timeout, cancelled: op.Async == cmdspec.Self})
		})
	}
	if len(bytes.TrimSpace(resp)) == 0 {
		if status != http.StatusNoContent && op.Columns != "" {
			return fmt.Errorf("HTTP %d with an empty body", status)
		}
		if a.TTY && !a.quiet {
			fmt.Fprintln(a.Stderr, "Done.")
		}
		return nil
	}
	if !json.Valid(resp) {
		return fmt.Errorf("HTTP %d: the response is not JSON", status)
	}
	if err := printRaw(pr, op.Columns, resp); err != nil {
		return err
	}
	name, _ := body["name"].(string)
	if len(targets) > 0 {
		name = targets[0].given
	}
	return missingSecret(op.OperationID, name, resp)
}

func orList(flags []string) string {
	q := make([]string, len(flags))
	for i, f := range flags {
		q[i] = "--" + f
	}
	if len(q) == 2 {
		return q[0] + ", " + q[1] + " or both"
	}
	return "at least one of " + strings.Join(q, ", ")
}

// convertParam applies a path parameter's converter.
func convertParam(p cmdspec.Param, v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s is empty", p.Name)
	}
	switch p.Convert {
	case "":
		return v, nil
	case "epoch":
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return strconv.FormatInt(n, 10), nil
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return "", fmt.Errorf("%q is not a time: give RFC 3339 (2026-01-02T15:04:05Z) or epoch seconds", v)
		}
		return strconv.FormatInt(t.Unix(), 10), nil
	}
	return "", fmt.Errorf("internal: unknown converter %q", p.Convert)
}

// fieldValue reads a flag as its JSON value, scaled; set is false when the flag was not given.
func fieldValue(f *pflag.FlagSet, fl cmdspec.Field) (v any, set bool, err error) {
	if !f.Changed(fl.Flag) {
		return nil, false, nil
	}
	switch fl.Type {
	case "integer":
		n, _ := f.GetInt64(fl.Flag)
		limit := int64(math.MaxInt64)
		if fl.Format == "int32" {
			limit = math.MaxInt32
		}
		if fl.Scale != 0 {
			if n <= 0 || n > limit/fl.Scale {
				return nil, true, fmt.Errorf("--%s %d is out of range", fl.Flag, n)
			}
			n *= fl.Scale
		}
		if n > limit || n < -limit-1 {
			return nil, true, fmt.Errorf("--%s %d is out of range", fl.Flag, n)
		}
		return n, true, nil
	case "number":
		n, _ := f.GetFloat64(fl.Flag)
		return n, true, nil
	case "boolean":
		b, _ := f.GetBool(fl.Flag)
		return b, true, nil
	}
	s, _ := f.GetString(fl.Flag)
	if len(fl.Enum) > 0 && !slices.Contains(fl.Enum, s) {
		return nil, true, fmt.Errorf("--%s %q: use one of %s", fl.Flag, s, strings.Join(fl.Enum, ", "))
	}
	return s, true, nil
}

// bodyFrom merges --data with the body flags; flags win.
func (a *App) bodyFrom(op cmdspec.Op, f *pflag.FlagSet, data string) (map[string]any, error) {
	body := map[string]any{}
	raw, err := a.readData(data)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&body); err != nil {
			return nil, fmt.Errorf("--data is not a JSON object: %w", err)
		}
		if body == nil {
			return nil, errors.New("--data is not a JSON object")
		}
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			return nil, errors.New("--data has content after its JSON object")
		}
	}
	for _, fl := range op.Body {
		v, set, err := fieldValue(f, fl)
		if err != nil {
			return nil, err
		}
		if set {
			body[fl.Name] = v
			continue
		}
		if _, ok := body[fl.Name]; !ok && fl.Required {
			return nil, fmt.Errorf("--%s is required", fl.Flag)
		}
	}
	return body, nil
}

func confirmQuestion(op cmdspec.Op, targets []target) string {
	verb := op.Path[len(op.Path)-1]
	verb = strings.ToUpper(verb[:1]) + strings.ReplaceAll(verb[1:], "-", " ")
	noun := strings.ReplaceAll(op.Path[len(op.Path)-2], "-", " ")
	if r, ok := genResources[op.Positional.Resource]; ok && op.Positional.Name != "" {
		noun = r.Noun
	}
	owner := func(t target) string {
		if r, ok := genResources[t.param.Resource]; ok {
			return r.Noun + " " + describe(t)
		}
		return t.param.Flag + " " + describe(t)
	}
	if op.Positional.Name != "" {
		if len(targets) > 1 {
			return fmt.Sprintf("%s %s %s of %s?", verb, noun, describe(targets[0]), owner(targets[1]))
		}
		return fmt.Sprintf("%s %s %s?", verb, noun, describe(targets[0]))
	}
	if len(targets) > 0 {
		return fmt.Sprintf("%s the %s of %s?", verb, noun, owner(targets[0]))
	}
	return fmt.Sprintf("%s %s?", verb, noun)
}

func describe(t target) string {
	if t.byName {
		return fmt.Sprintf("%s (%s)", t.given, t.id)
	}
	return t.id
}

// buildPath fills the URL template; a verbatim parameter keeps its slashes.
func buildPath(tmpl string, targets []target) string {
	for _, t := range targets {
		v := t.id
		if t.param.Verbatim {
			v = strings.TrimLeft(v, "/")
		} else {
			v = url.PathEscape(v)
		}
		tmpl = strings.Replace(tmpl, "{"+t.param.Name+"}", v, 1)
	}
	return tmpl
}

// send makes one API call and returns its status, body and Location.
func send(ctx context.Context, s *session, method, path string, query url.Values, payload []byte) (int, []byte, string, error) {
	u := s.endpoint + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var rd io.Reader
	if payload != nil {
		rd = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return 0, nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := rawDo(s, req)
	if err != nil {
		return 0, nil, "", err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, "", err
	}
	if err := cloudafrica.CheckResponse(resp, out); err != nil {
		return resp.StatusCode, out, "", err
	}
	return resp.StatusCode, out, resp.Header.Get("Location"), nil
}

// acceptedID is a 202's operation id: the body's operationId, else the Location's.
func acceptedID(body []byte, loc string) string {
	var acc struct {
		OperationID string `json:"operationId"`
	}
	if json.Unmarshal(body, &acc) == nil && acc.OperationID != "" {
		return acc.OperationID
	}
	id, _ := operationIDFromLocation(loc)
	return id
}

// page is one CursorPage body.
type page struct {
	rows []json.RawMessage
	next string
}

func decodePage(body []byte, rowsField string) (page, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return page{}, fmt.Errorf("decoding the list: %w", err)
	}
	var p page
	raw, ok := m[rowsField]
	if !ok {
		return page{}, fmt.Errorf("the list has no %q field", rowsField)
	}
	if err := json.Unmarshal(raw, &p.rows); err != nil {
		return page{}, fmt.Errorf("decoding the list's %s: %w", rowsField, err)
	}
	if raw, ok := m["nextCursor"]; ok {
		if err := json.Unmarshal(raw, &p.next); err != nil {
			return page{}, fmt.Errorf("the list's nextCursor is not a string: %w", err)
		}
	}
	return p, nil
}

func rowsField(columns string) string {
	if cs, ok := genColumns[columns]; ok && cs.Rows != "" {
		return cs.Rows
	}
	return "data"
}

// listPages walks every page of a list path.
func listPages(ctx context.Context, s *session, path string, query url.Values, paged bool, rows string) ([]json.RawMessage, error) {
	out := []json.RawMessage{}
	seen := map[string]bool{}
	q := url.Values{}
	for k, v := range query {
		q[k] = v
	}
	if paged {
		q.Set("limit", strconv.Itoa(pageLimit))
	}
	for {
		_, body, _, err := send(ctx, s, http.MethodGet, path, q, nil)
		if err != nil {
			return nil, err
		}
		p, err := decodePage(body, rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p.rows...)
		if !paged || p.next == "" {
			return out, nil
		}
		if seen[p.next] {
			return nil, fmt.Errorf("the server repeated the page cursor %q", p.next)
		}
		seen[p.next] = true
		q.Set("cursor", p.next)
	}
}

func (a *App) runList(ctx context.Context, s *session, pr output.Printer, op cmdspec.Op, path string, query url.Values, r genRun) error {
	rows := rowsField(op.Columns)
	if r.limit > 0 {
		q := url.Values{}
		for k, v := range query {
			q[k] = v
		}
		q.Set("limit", strconv.Itoa(r.limit))
		if r.cursor != "" {
			q.Set("cursor", r.cursor)
		}
		_, body, _, err := send(ctx, s, http.MethodGet, path, q, nil)
		if err != nil {
			return err
		}
		p, err := decodePage(body, rows)
		if err != nil {
			return err
		}
		res := tableOf(op.Columns, p.rows)
		res.Value = json.RawMessage(body)
		if err := pr.Print(res); err != nil {
			return err
		}
		if p.next != "" && pr.Format == output.Table && !pr.Quiet {
			fmt.Fprintf(a.Stderr, "More rows: --cursor %s\n", p.next)
		}
		return nil
	}
	all, err := listPages(ctx, s, path, query, op.Paged, rows)
	if err != nil {
		return err
	}
	res := tableOf(op.Columns, all)
	res.Value = all
	return pr.Print(res)
}

// printRaw prints one response body: verbatim for json/yaml, through the column set for a table.
func printRaw(pr output.Printer, columns string, body []byte) error {
	cs, ok := genColumns[columns]
	if !ok {
		res := output.Result{Value: json.RawMessage(body)}
		if id := field(decode(body), "id"); id != "" {
			res.IDs = []string{id}
		}
		if pr.Format == output.Table && !pr.Quiet {
			pr.Format = output.JSON
		}
		return pr.Print(res)
	}
	if cs.Rows != "" {
		p, err := decodePage(body, cs.Rows)
		if err != nil {
			return err
		}
		res := tableOf(columns, p.rows)
		res.Value = json.RawMessage(body)
		return pr.Print(res)
	}
	res := tableOf(columns, []json.RawMessage{body})
	res.Value = json.RawMessage(body)
	return pr.Print(res)
}

func decode(raw []byte) any {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	_ = dec.Decode(&v)
	return v
}

// tableOf renders rows through a column set.
func tableOf(columns string, rows []json.RawMessage) output.Result {
	cs := genColumns[columns]
	idField := cs.ID
	if idField == "" {
		idField = "id"
	}
	res := output.Result{}
	for _, c := range cs.Columns {
		res.Headers = append(res.Headers, c.Header)
	}
	for _, raw := range rows {
		v := decode(raw)
		row := make([]string, len(cs.Columns))
		for i, c := range cs.Columns {
			row[i] = format(lookup(v, c.Path), c.Format)
		}
		res.Rows = append(res.Rows, row)
		res.IDs = append(res.IDs, field(v, idField))
	}
	return res
}

func lookup(v any, path string) any {
	for _, k := range strings.Split(path, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

func field(v any, path string) string {
	return format(lookup(v, path), "")
}

func format(v any, how string) string {
	if v == nil {
		return ""
	}
	switch how {
	case "bytes":
		if n, ok := v.(json.Number); ok {
			if i, err := n.Int64(); err == nil {
				return humanBytes(i)
			}
		}
	case "time":
		if s, ok := v.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t.Local().Format("2006-01-02 15:04 MST")
			}
		}
	case "upper":
		if s, ok := v.(string); ok {
			return strings.ToUpper(s)
		}
	case "bool":
		if b, ok := v.(bool); ok {
			if b {
				return "yes"
			}
			return "no"
		}
	case "list":
		if l, ok := v.([]any); ok {
			parts := make([]string, len(l))
			for i, e := range l {
				parts[i] = format(e, "")
			}
			return strings.Join(parts, ", ")
		}
	case "count":
		if l, ok := v.([]any); ok {
			return strconv.Itoa(len(l))
		}
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit && exp < 5; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// resolveName turns a resource's id or name into its id. byName reports a name match.
func (a *App) resolveName(ctx context.Context, s *session, resource, value string) (id string, byName bool, err error) {
	return a.resolveParam(ctx, s, cmdspec.Param{Resource: resource}, value, nil)
}

// resolveParam is resolveName for a path parameter. With a parent (p.Within set),
// only a row whose p.Within field holds the parent's id is accepted: an id
// elsewhere is refused, and a name matches only there.
func (a *App) resolveParam(ctx context.Context, s *session, p cmdspec.Param, value string, parent *target) (id string, byName bool, err error) {
	r, ok := genResources[p.Resource]
	if !ok {
		return value, false, nil
	}
	if parent == nil && r.IDPattern != "" && regexp.MustCompile(r.IDPattern).MatchString(value) {
		return value, false, nil
	}
	idField := r.ID
	if idField == "" {
		idField = "id"
	}
	rows, err := listPages(ctx, s, r.List, nil, true, "data")
	if err != nil {
		var e *cloudafrica.APIError
		if parent != nil || ctx.Err() != nil || errors.As(err, &e) && e.Status == http.StatusUnauthorized {
			return "", false, err
		}
		// The value may well be an id, and the API answers for a wrong one.
		fmt.Fprintf(a.Stderr, "Cannot list %ss to look up %q (%v); using it as an id.\n", r.Noun, value, err)
		return value, false, nil
	}
	inParent := func(v any) bool { return parent == nil || field(v, p.Within) == parent.id }
	where := ""
	if parent != nil {
		where = fmt.Sprintf(" in --%s %s", parent.param.Flag, parent.given)
	}
	var matches []any
	for _, raw := range rows {
		v := decode(raw)
		if field(v, idField) == value {
			if !inParent(v) {
				return "", false, usageErr(fmt.Errorf("%s %s is not%s", r.Noun, value, where))
			}
			return value, false, nil
		}
		if field(v, r.Match) == value && inParent(v) {
			matches = append(matches, v)
		}
	}
	switch len(matches) {
	case 0:
		return "", false, usageErr(fmt.Errorf("no %s with id or %s %q%s", r.Noun, r.Match, value, where))
	case 1:
		return field(matches[0], idField), true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d %ss%s have %s %q; give the id instead:", len(matches), r.Noun, where, r.Match, value)
	for _, m := range matches {
		fmt.Fprintf(&b, "\n  %s  %s", field(m, idField), field(m, r.Match))
	}
	return "", false, usageErr(errors.New(b.String()))
}

// atLeastOneGiven reports whether a flag in op.AtLeastOne was given, or --data
// set its field or a field that has no flag.
func atLeastOneGiven(op cmdspec.Op, f *pflag.FlagSet, body map[string]any) bool {
	if slices.ContainsFunc(op.AtLeastOne, f.Changed) {
		return true
	}
	flagged := map[string]bool{}
	for _, fl := range op.Body {
		flagged[fl.Name] = true
		if _, ok := body[fl.Name]; ok && slices.Contains(op.AtLeastOne, fl.Flag) {
			return true
		}
	}
	for k := range body {
		if !flagged[k] {
			return true
		}
	}
	return false
}
