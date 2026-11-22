package cli

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/CloudAfrica/cloudafrica-cli/internal/cmdspec"
	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

func canned(h *harness, routes map[string]fakeapi.Canned) {
	h.fake.Canned = routes
}

func TestGeneratedPathParamFlagAndPositional(t *testing.T) {
	h := signedIn(t)
	canned(h, map[string]fakeapi.Canned{"DELETE /v1/vms/vm-1/firewall-rules/r1": {Status: 204}})
	h.mustRun("vm", "firewall-rule", "delete", "r1", "--vm", "web-1", "--yes")
	if len(h.fake.WriteURLs) != 1 || h.fake.WriteURLs[0] != "DELETE /v1/vms/vm-1/firewall-rules/r1" {
		t.Errorf("writes %v", h.fake.WriteURLs)
	}
	if code := h.run("vm", "firewall-rule", "delete", "r1", "--yes"); code != ExitUsage || len(h.fake.Writes) != 1 {
		t.Errorf("without --vm: exit %d, writes %d", code, len(h.fake.Writes))
	}
}

func TestOperationGetKeepsASlashedID(t *testing.T) {
	h := signedIn(t)
	h.mustRun("operation", "get", "op/start-vm-1", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "op/start-vm-1" {
		t.Errorf("stdout = %q", got)
	}
}

func TestOperationTableShowsTheError(t *testing.T) {
	h := signedIn(t)
	canned(h, map[string]fakeapi.Canned{"GET /v1/operations/op-1": {Body: `{"id":"op-1","state":"failed",
		"error":{"message":"disk full"}}`}})
	h.mustRun("operation", "get", "op-1", "-o", "table")
	if out := h.stdout.String(); !strings.Contains(out, "ERROR") || !strings.Contains(out, "disk full") {
		t.Errorf("stdout = %s", out)
	}
}

func TestDataMergesUnderFlags(t *testing.T) {
	h := signedIn(t)
	canned(h, map[string]fakeapi.Canned{"POST /v1/vms/vm-1/firewall-rules": {Status: 201, Body: `{}`}})
	h.mustRun("vm", "firewall-rule", "create", "--vm", "vm-1",
		"-d", `{"protocol":"tcp","source":"0.0.0.0/0","fromPort":22,"future":{"a":1}}`, "--from-port", "2222")
	b := h.fake.Bodies[0]
	if b["protocol"] != "tcp" || b["fromPort"] != float64(2222) || b["future"] == nil || b["source"] != "0.0.0.0/0" {
		t.Errorf("body = %v", b)
	}
	for name, args := range map[string][]string{
		"required field missing": {"--protocol", "tcp"},
		"data not an object":     {"-d", `[1]`, "--protocol", "tcp", "--source", "x"},
		"data not JSON":          {"-d", `{`, "--protocol", "tcp", "--source", "x"},
		"port out of int32":      {"--protocol", "tcp", "--source", "x", "--from-port", "4294967296"},
	} {
		h := signedIn(t)
		if code := h.run(append([]string{"vm", "firewall-rule", "create", "--vm", "vm-1"}, args...)...); code != ExitUsage ||
			len(h.fake.Writes) != 0 {
			t.Errorf("%s: exit %d, writes %d", name, code, len(h.fake.Writes))
		}
	}
}

func TestResizeScalesGiBAndHidesSchemaArtefacts(t *testing.T) {
	h := signedIn(t)
	h.mustRun("vm", "resize", "vm-1", "--ram-gib", "8", "--no-wait")
	if b := h.fake.Bodies[0]; b["ramMib"] != float64(8192) || len(b) != 1 {
		t.Errorf("body = %v", b)
	}
	if code := h.run("vm", "resize", "vm-1", "--empty"); code != ExitUsage {
		t.Errorf("--empty is a flag: exit %d", code)
	}
}

func TestEpochPositionalTakesRFC3339OrSeconds(t *testing.T) {
	at := time.Date(2026, 11, 22, 8, 0, 0, 0, time.UTC)
	for _, given := range []string{at.Format(time.RFC3339), fmt.Sprint(at.Unix())} {
		h := signedIn(t)
		want := fmt.Sprintf("DELETE /v1/vms/vm-1/backups/%d", at.Unix())
		canned(h, map[string]fakeapi.Canned{want: {Status: 204}})
		h.mustRun("vm", "backup", "delete", given, "--vm", "vm-1", "--yes")
		if len(h.fake.WriteURLs) != 1 || h.fake.WriteURLs[0] != want {
			t.Errorf("%s: writes %v", given, h.fake.WriteURLs)
		}
	}
	h := signedIn(t)
	if code := h.run("vm", "backup", "delete", "yesterday", "--vm", "vm-1", "--yes"); code != ExitUsage || len(h.fake.Writes) != 0 {
		t.Errorf("bad time: exit %d", code)
	}
}

// No slice-1 field has an enum, so the check is driven directly.
func TestEnumIsChecked(t *testing.T) {
	field := cmdspec.Field{Name: "direction", Flag: "direction", Type: "string", Enum: []string{"in", "out"}}
	for given, ok := range map[string]bool{"in": true, "sideways": false} {
		fs := pflag.NewFlagSet("t", pflag.ContinueOnError)
		fs.String("direction", "", "")
		_ = fs.Set("direction", given)
		v, set, err := fieldValue(fs, field)
		if !set || (err == nil) != ok || (ok && v != given) || (!ok && !strings.Contains(err.Error(), "use one of in, out")) {
			t.Errorf("%s: %v %v %v", given, v, set, err)
		}
	}
}

func TestCancelWaitsUntilCancelled(t *testing.T) {
	h := signedIn(t)
	canned(h, map[string]fakeapi.Canned{"POST /v1/operations/op-9/cancel": {Status: 202}})
	h.fake.OpStates["op-9"] = []string{"running", "cancelled"}
	h.mustRun("operation", "cancel", "op-9", "-o", "json")
	if !strings.Contains(h.stdout.String(), `"cancelled"`) {
		t.Errorf("stdout = %s", h.stdout)
	}
	h.fake.OpStates["op-9"] = []string{"failed"}
	if code := h.run("operation", "cancel", "op-9"); code != ExitAPI {
		t.Errorf("a failed operation is not a cancel: exit %d", code)
	}
}

func TestWaitShowsProgressOnlyOnATerminal(t *testing.T) {
	for _, tty := range []bool{true, false} {
		h := signedIn(t)
		h.tty = tty
		h.fake.OpStates["op/start-vm-1"] = []string{"running", "running", "complete"}
		h.mustRun("vm", "start", "vm-1", "-o", "json")
		got := strings.Count(h.stderr.String(), "vm.action: running 40% copying")
		if (tty && got != 1) || (!tty && got != 0) {
			t.Errorf("tty=%v: progress lines %d, stderr %q", tty, got, h.stderr)
		}
	}
}

func TestNoContentSaysDoneOnlyOnATerminal(t *testing.T) {
	for _, tty := range []bool{true, false} {
		h := signedIn(t)
		h.tty = tty
		canned(h, map[string]fakeapi.Canned{"DELETE /v1/vms/vm-1/ssh-keys/k1": {Status: 204}})
		h.mustRun("vm", "ssh-key", "detach", "laptop", "--vm", "web-1", "--yes")
		if strings.Contains(h.stderr.String(), "Done.") != tty || h.stdout.Len() != 0 {
			t.Errorf("tty=%v: stdout %q stderr %q", tty, h.stdout, h.stderr)
		}
	}
}

func TestConfirmNamesTheResolvedTarget(t *testing.T) {
	h := signedIn(t)
	h.tty = true
	h.stdin = "n\n"
	if code := h.run("vm", "delete", "web-1"); code == ExitOK || h.fake.DeleteCalls != 0 {
		t.Fatalf("declined: exit %d", code)
	}
	if !strings.Contains(h.stderr.String(), "Delete VM web-1 (vm-1)?") {
		t.Errorf("prompt: %s", h.stderr)
	}
	h.stdin = "n\n"
	if code := h.run("vm", "firewall-rule", "delete", "r1", "--vm", "web-1"); code == ExitOK || len(h.fake.Writes) != 0 {
		t.Fatalf("declined: exit %d", code)
	}
	if !strings.Contains(h.stderr.String(), "Delete firewall rule r1 of VM web-1 (vm-1)? [y/N]") {
		t.Errorf("sub-resource prompt: %s", h.stderr)
	}
}

func TestNameResolution(t *testing.T) {
	vms := []map[string]any{}
	for i := 1; i <= 6; i++ {
		vms = append(vms, map[string]any{"id": fmt.Sprintf("vm-%d", i), "name": fmt.Sprintf("n%d", i)})
	}
	vms = append(vms, map[string]any{"id": "vm-7", "name": "twin"}, map[string]any{"id": "vm-8", "name": "twin"})

	h := signedIn(t)
	h.fake.VMs = vms
	h.mustRun("vm", "start", "n5", "--no-wait", "-q")
	if h.fake.WriteURLs[0] != "POST /v1/vms/vm-5/actions/start" {
		t.Errorf("found across pages: %v", h.fake.WriteURLs)
	}
	if code := h.run("vm", "start", "nobody", "--no-wait"); code != ExitUsage || !strings.Contains(h.stderr.String(), `"nobody"`) {
		t.Errorf("none: exit %d, %s", code, h.stderr)
	}
	if code := h.run("vm", "start", "twin", "--no-wait"); code != ExitUsage ||
		!strings.Contains(h.stderr.String(), "vm-7") || !strings.Contains(h.stderr.String(), "vm-8") {
		t.Errorf("ambiguous: exit %d, %s", code, h.stderr)
	}
	if len(h.fake.Writes) != 1 {
		t.Errorf("a failed lookup wrote: %v", h.fake.WriteURLs)
	}
	// An id the pattern recognises is not looked up.
	before := len(h.fake.ElevationHeaders)
	h.mustRun("vm", "start", "00000000-0000-0000-0000-00000000000a", "--no-wait", "-q")
	if n := len(h.fake.ElevationHeaders) - before; n != 1 {
		t.Errorf("a UUID was looked up: %d requests", n)
	}
}

func TestPlacementGroupMemberAddResolvesBoth(t *testing.T) {
	h := signedIn(t)
	canned(h, map[string]fakeapi.Canned{"PUT /v1/placement-groups/pg-1/members/vm-1": {Body: `{"id":"pg-1","name":"spread"}`}})
	h.mustRun("placement-group", "member", "add", "web-1", "--placement-group", "spread", "-q")
	if h.fake.WriteURLs[0] != "PUT /v1/placement-groups/pg-1/members/vm-1" || strings.TrimSpace(h.stdout.String()) != "pg-1" {
		t.Errorf("writes %v, stdout %q", h.fake.WriteURLs, h.stdout)
	}
}

func TestListLimitPrintsOnePageVerbatim(t *testing.T) {
	h := signedIn(t)
	h.fake.VMs = []map[string]any{{"id": "vm-1"}, {"id": "vm-2"}, {"id": "vm-3"}}
	h.mustRun("vm", "list", "--limit", "2", "-o", "json")
	var page map[string]any
	if err := json.Unmarshal(h.stdout.Bytes(), &page); err != nil || page["nextCursor"] != "c2" {
		t.Errorf("page = %s (%v)", h.stdout, err)
	}
	h.mustRun("vm", "list", "--limit", "2", "-o", "table")
	if !strings.Contains(h.stderr.String(), "More rows: --cursor c2") {
		t.Errorf("stderr = %s", h.stderr)
	}
	h.mustRun("vm", "list", "--limit", "2", "--cursor", "c2", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "vm-3" {
		t.Errorf("second page = %q", got)
	}
	if code := h.run("vm", "list", "--cursor", "c2"); code != ExitUsage {
		t.Errorf("--cursor without --limit: exit %d", code)
	}
	for _, n := range []string{"0", "-1"} {
		if code := h.run("vm", "list", "--limit", n); code != ExitUsage {
			t.Errorf("--limit %s: exit %d", n, code)
		}
	}
	h.mustRun("vm", "list", "-o", "json")
	var all []map[string]any
	if err := json.Unmarshal(h.stdout.Bytes(), &all); err != nil || len(all) != 3 {
		t.Errorf("all = %s", h.stdout)
	}
}

func TestColumnsFormatAndJSONIsVerbatim(t *testing.T) {
	h := signedIn(t)
	canned(h, map[string]fakeapi.Canned{"GET /v1/vms/vm-1/backups": {Body: `{"data":[
		{"startedAt":"2026-11-22T08:00:00Z","sizeBytes":3221225472,"protected":true,"origin":"manual","future":1}]}`}})
	h.mustRun("vm", "backup", "list", "--vm", "vm-1", "-o", "table")
	out := h.stdout.String()
	for _, want := range []string{"STARTED", "SIZE", "PROTECTED", "3.0 GiB", "yes", "manual", "2026-11-22"} {
		if !strings.Contains(out, want) {
			t.Errorf("table lacks %q:\n%s", want, out)
		}
	}
	h.mustRun("vm", "backup", "list", "--vm", "vm-1", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "2026-11-22T08:00:00Z" {
		t.Errorf("-q = %q", got)
	}
	h.mustRun("vm", "backup", "list", "--vm", "vm-1", "-o", "json")
	if !strings.Contains(h.stdout.String(), `"future": 1`) && !strings.Contains(h.stdout.String(), `"future":1`) {
		t.Errorf("json dropped an unknown field: %s", h.stdout)
	}
}

func TestIDsWorkWithoutTheScopeToListThem(t *testing.T) {
	h := newHarness(t)
	h.seed("default", &fakeapi.Grant{Scopes: []string{"read:compute", "write:compute"}}, time.Now().Add(time.Hour))
	h.mustRun(append(slices.Clone(createArgs), "--ssh-key", "k1", "-q")...)
	if b := h.fake.Bodies[0]; b["vpcResourceId"] != "vpc-1" || fmt.Sprint(b["sshKeyIds"]) != "[k1]" {
		t.Errorf("body = %v", b)
	}
}

func TestDataMustBeOneObject(t *testing.T) {
	for name, data := range map[string]string{
		"null":             `null`,
		"trailing object":  `{"protocol":"tcp","source":"x"} {"fromPort":99}`,
		"trailing garbage": `{"protocol":"tcp","source":"x"} x`,
	} {
		h := signedIn(t)
		code := h.run("vm", "firewall-rule", "create", "--vm", "vm-1", "-d", data, "--protocol", "tcp", "--source", "x")
		if code != ExitUsage || len(h.fake.Writes) != 0 {
			t.Errorf("%s: exit %d, writes %d, stderr %q", name, code, len(h.fake.Writes), h.stderr)
		}
	}
}

func TestNonJSONSuccessIsAnError(t *testing.T) {
	h := signedIn(t)
	canned(h, map[string]fakeapi.Canned{"GET /v1/vms/vm-1": {Status: 200, Body: `<html>captive portal</html>`}})
	if code := h.run("vm", "get", "vm-1", "-o", "table"); code == ExitOK || h.stdout.Len() != 0 ||
		!strings.Contains(h.stderr.String(), "not JSON") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, h.stdout, h.stderr)
	}
}

func TestMalformedListPagesAreErrors(t *testing.T) {
	for name, body := range map[string]string{
		"rows under another key":    `{"items":[{"id":"vm-1"}]}`,
		"nextCursor not a string":   `{"data":[{"id":"vm-1"}],"nextCursor":7}`,
		"server repeats its cursor": `{"data":[{"id":"vm-1"}],"nextCursor":"x"}`,
	} {
		h := signedIn(t)
		canned(h, map[string]fakeapi.Canned{"GET /v1/vms": {Status: 200, Body: body}})
		done := make(chan int, 1)
		go func() { done <- h.run("vm", "list", "-o", "json") }()
		select {
		case code := <-done:
			if code == ExitOK {
				t.Errorf("%s: exit 0, stdout %q", name, h.stdout)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: vm list did not finish", name)
		}
	}
	h := signedIn(t)
	canned(h, map[string]fakeapi.Canned{"GET /v1/vms": {Status: 200, Body: `{"data":[],"nextCursor":null}`}})
	h.mustRun("vm", "list", "-o", "json")
	if got := strings.TrimSpace(h.stdout.String()); got != "[]" {
		t.Errorf("empty list printed %q", got)
	}
}

func TestUnexpectedOperationStateEndsTheWait(t *testing.T) {
	h := signedIn(t)
	h.fake.OpStates["op/start-vm-1"] = []string{"running", "expired"}
	if code := h.run("vm", "start", "vm-1", "--timeout", "1m"); code != ExitAPI ||
		!strings.Contains(h.stderr.String(), `unexpected state "expired"`) {
		t.Errorf("exit %d, stderr %q", code, h.stderr)
	}
	h = signedIn(t)
	h.fake.OpStates["op/start-vm-1"] = []string{"failed"}
	if code := h.run("vm", "start", "vm-1"); code != ExitAPI || !strings.Contains(h.stderr.String(), "fault") {
		t.Errorf("a failure names the error's kind: exit %d, stderr %q", code, h.stderr)
	}
}

func TestBooleanFlags(t *testing.T) {
	const url = "PUT /v1/vms/vm-1/backups/1700000000/protection"
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--protected=false"}, "map[protected:false]"},
		{[]string{"--protected"}, "map[protected:true]"},
		{nil, "map[]"},
	} {
		h := signedIn(t)
		canned(h, map[string]fakeapi.Canned{url: {Status: 204}})
		h.mustRun(append([]string{"vm", "backup", "set-protection", "1700000000", "--vm", "vm-1", "--yes"}, c.args...)...)
		if got := fmt.Sprint(h.fake.Bodies[0]); got != c.want {
			t.Errorf("%v: body %s, want %s", c.args, got, c.want)
		}
	}
	h := signedIn(t)
	if code := h.run("vm", "backup", "restore-to-new-vm", "1700000000", "--vm", "vm-1", "--name", "n",
		"--datacenter-slug", "jb1", "--yes"); code != ExitUsage || len(h.fake.Writes) != 0 {
		t.Errorf("without --terms-accepted: exit %d, writes %d", code, len(h.fake.Writes))
	}
}

func TestAFailedLookupFallsBackToTheID(t *testing.T) {
	for name, c := range map[string]fakeapi.Canned{
		"unnamed scope refusal": {Status: 403, Body: `{"type":"https://api.cloudafrica.net/problems/insufficient-scope","title":"Insufficient scope","status":403}`},
		"other refusal":         {Status: 403, Body: `{"type":"https://api.cloudafrica.net/problems/forbidden","title":"Forbidden","status":403}`},
	} {
		h := signedIn(t)
		canned(h, map[string]fakeapi.Canned{"GET /v1/vpcs": c})
		h.mustRun(append(slices.Clone(createArgs), "--ssh-key", "k1", "-q")...)
		if b := h.fake.Bodies[0]; b["vpcResourceId"] != "vpc-1" || !strings.Contains(h.stderr.String(), `using it as an id`) {
			t.Errorf("%s: body %v, stderr %q", name, b, h.stderr)
		}
	}
	h := signedIn(t)
	canned(h, map[string]fakeapi.Canned{"GET /v1/vpcs": {Status: 401, Body: `{"type":"about:blank","title":"Unauthorized","status":401}`}})
	if code := h.run(append(slices.Clone(createArgs), "--ssh-key", "k1", "-q")...); code == ExitOK || len(h.fake.Writes) != 0 {
		t.Errorf("a 401 on the lookup: exit %d, writes %d", code, len(h.fake.Writes))
	}
}

func TestResizeTakesDisksFromData(t *testing.T) {
	h := signedIn(t)
	h.mustRun("vm", "resize", "vm-1", "-d", `{"disks":[{"id":"d1","sizeGib":50}]}`, "--no-wait")
	if got := fmt.Sprint(h.fake.Bodies[0]); got != "map[disks:[map[id:d1 sizeGib:50]]]" {
		t.Errorf("body %s", got)
	}
	h = signedIn(t)
	h.mustRun("vm", "resize", "vm-1", "-d", `{"cpus":4}`, "--no-wait")
	if code := h.run("vm", "resize", "vm-1", "-d", `{}`); code != ExitUsage {
		t.Errorf("an empty --data: exit %d", code)
	}
}

// A VIP's firewall rules come back unpaged under "rules": the table and -q
// read that field, JSON is the body verbatim, and --vpc and --vip resolve.
func TestVipFirewallRuleListReadsRules(t *testing.T) {
	h := signedIn(t)
	h.fake.VPCs = []map[string]any{{"id": "vpc-1", "name": "web"}}
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/vips": {Body: `{"data":[{"id":"vip-1","name":"lb1","vpcId":"vpc-1"}]}`},
		"GET /v1/vpcs/vpc-1/vips/vip-1/firewall-rules": {Body: `{"rules":[
			{"id":"r1","direction":"ingress","action":"accept","protocol":"tcp","source":"0.0.0.0/0",
			"fromPort":80,"toPort":443,"status":"applied","description":"web"}],
			"writable":false,"readOnlyReason":"managed"}`},
	})
	h.mustRun("vpc", "vip", "firewall-rule", "list", "--vip", "lb1", "--vpc", "web", "-o", "table")
	out := h.stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "DIRECTION") ||
		strings.Join(strings.Fields(lines[1]), " ") != "r1 INGRESS ACCEPT TCP 0.0.0.0/0 80 443 applied web" {
		t.Errorf("table:\n%s", out)
	}
	h.mustRun("vpc", "vip", "firewall-rule", "list", "--vip", "lb1", "--vpc", "web", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "r1" {
		t.Errorf("-q = %q", got)
	}
	h.mustRun("vpc", "vip", "firewall-rule", "list", "--vip", "lb1", "--vpc", "web", "-o", "json")
	if !strings.Contains(h.stdout.String(), `"readOnlyReason": "managed"`) {
		t.Errorf("json is not the body: %s", h.stdout)
	}
}

// A VIP delete resolves the VPC flag and the VIP positional before it writes.
func TestVpcVipDeleteResolvesBothNames(t *testing.T) {
	h := newHarness(t)
	g := protected()
	g.Elevated = true
	h.seed("default", g, time.Now().Add(time.Hour))
	h.fake.VPCs = []map[string]any{{"id": "vpc-1", "name": "web"}}
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/vips":                     {Body: `{"data":[{"id":"vip-1","name":"lb1","vpcId":"vpc-1"}]}`},
		"DELETE /v1/vpcs/vpc-1/vips/vip-1": {Status: 202, Body: `{"operationId":"op/vip-del"}`},
	})
	h.mustRun("vpc", "vip", "delete", "lb1", "--vpc", "web", "--yes", "--no-wait")
	if len(h.fake.WriteURLs) != 1 || h.fake.WriteURLs[0] != "DELETE /v1/vpcs/vpc-1/vips/vip-1" {
		t.Errorf("writes %v", h.fake.WriteURLs)
	}
}

// The networking list columns: ip-address IDs come from the address, servedBy
// is a count, primary reads yes/no, and a null leaves the cell blank.
func TestNetworkingListColumns(t *testing.T) {
	h := signedIn(t)
	h.fake.VPCs = []map[string]any{{"id": "vpc-1", "name": "web"}}
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/ip-addresses": {Body: `{"data":[
			{"address":"203.0.113.7","kind":"public","ownerName":"web-01","ownerKind":"vm","primary":true},
			{"address":"10.0.0.5","kind":"private","ownerName":"lb1","ownerKind":"vip","vpcName":"web","primary":false},
			{"address":"203.0.113.9","kind":"public","primary":null}]}`},
		"GET /v1/vips": {Body: `{"data":[
			{"id":"vip-1","name":"lb1","vpcName":"web","publicAddress":"203.0.113.20","privateAddress":"10.0.0.20",
			"servedBy":[{"vmId":"a"},{"vmId":"b"}]},
			{"id":"vip-2","name":"lb2","vpcName":"web","servedBy":null}]}`},
		"GET /v1/vpcs/vpc-1/reserved-ips": {Body: `{"data":[
			{"id":"rip-1","address":"203.0.113.7","vmId":"vm-1"},{"id":"rip-2","address":"203.0.113.8","vmId":null}]}`},
	})
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"ip-address", "list"}, "203.0.113.7 10.0.0.5 203.0.113.9"},
		{[]string{"vip", "list"}, "vip-1 vip-2"},
		{[]string{"vpc", "reserved-ip", "list", "--vpc", "web"}, "rip-1 rip-2"},
	} {
		h.mustRun(append(c.args, "-q")...)
		if got := strings.Join(strings.Fields(h.stdout.String()), " "); got != c.want {
			t.Errorf("%v -q = %q, want %q", c.args, got, c.want)
		}
	}
	rows := func(args ...string) [][]string {
		h.mustRun(append(args, "-o", "table")...)
		var out [][]string
		for _, l := range strings.Split(strings.TrimSpace(h.stdout.String()), "\n")[1:] {
			out = append(out, strings.Fields(l))
		}
		return out
	}
	ips := rows("ip-address", "list")
	if got := fmt.Sprint(ips); got != "[[203.0.113.7 public web-01 vm yes] [10.0.0.5 private lb1 vip web no] [203.0.113.9 public]]" {
		t.Errorf("ip-address rows %s", got)
	}
	vips := rows("vip", "list")
	if got := fmt.Sprint(vips); got != "[[vip-1 lb1 web 203.0.113.20 10.0.0.20 2] [vip-2 lb2 web]]" {
		t.Errorf("vip rows %s", got)
	}
	reserved := rows("vpc", "reserved-ip", "list", "--vpc", "web")
	if got := fmt.Sprint(reserved); got != "[[rip-1 203.0.113.7 vm-1] [rip-2 203.0.113.8]]" {
		t.Errorf("reserved-ip rows %s", got)
	}
}

// An empty "rules" is an empty table; a body without "rules" is an API error,
// not an empty list.
func TestVipFirewallRuleListEmptyAndMissingRules(t *testing.T) {
	h := signedIn(t)
	h.fake.VPCs = []map[string]any{{"id": "vpc-1", "name": "web"}}
	routes := map[string]fakeapi.Canned{
		"GET /v1/vips": {Body: `{"data":[{"id":"vip-1","name":"lb1","vpcId":"vpc-1"}]}`},
		"GET /v1/vpcs/vpc-1/vips/vip-1/firewall-rules": {Body: `{"rules":[],"writable":true}`},
	}
	canned(h, routes)
	args := []string{"vpc", "vip", "firewall-rule", "list", "--vip", "lb1", "--vpc", "web"}
	h.mustRun(append(args, "-q")...)
	if got := h.stdout.String(); got != "" {
		t.Errorf("-q on no rules = %q", got)
	}
	h.mustRun(append(args, "-o", "table")...)
	if lines := strings.Split(strings.TrimSpace(h.stdout.String()), "\n"); len(lines) != 1 || !strings.Contains(lines[0], "DIRECTION") {
		t.Errorf("table on no rules:\n%s", h.stdout)
	}
	routes["GET /v1/vpcs/vpc-1/vips/vip-1/firewall-rules"] = fakeapi.Canned{Body: `{"writable":true}`}
	if code := h.run(append(args, "-q")...); code != ExitAPI {
		t.Errorf("missing rules: exit %d", code)
	}
	if s := h.stderr.String(); !strings.Contains(s, `no "rules" field`) {
		t.Errorf("stderr = %s", s)
	}
}

// A VIP resolves within its --vpc: a name another VPC shares picks this VPC's,
// a name or id only another VPC holds is refused, and refusals send nothing.
func TestVpcVipResolvesWithinItsVpc(t *testing.T) {
	h := newHarness(t)
	g := protected()
	g.Elevated = true
	h.seed("default", g, time.Now().Add(time.Hour))
	h.fake.VPCs = []map[string]any{{"id": "vpc-1", "name": "web"}, {"id": "vpc-2", "name": "db"}}
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/vips": {Body: `{"data":[
			{"id":"vip-1","name":"lb1","vpcId":"vpc-1"},{"id":"vip-2","name":"lb1","vpcId":"vpc-2"},
			{"id":"vip-3","name":"only-db","vpcId":"vpc-2"},
			{"id":"vip-4","name":"twin","vpcId":"vpc-1"},{"id":"vip-5","name":"twin","vpcId":"vpc-1"}]}`},
		"DELETE /v1/vpcs/vpc-1/vips/vip-1": {Status: 202, Body: `{"operationId":"op/vip-del"}`},
	})
	h.mustRun("vpc", "vip", "delete", "lb1", "--vpc", "web", "--yes", "--no-wait")
	if len(h.fake.WriteURLs) != 1 || h.fake.WriteURLs[0] != "DELETE /v1/vpcs/vpc-1/vips/vip-1" {
		t.Fatalf("writes %v", h.fake.WriteURLs)
	}
	for _, c := range []struct{ vip, want string }{
		{"only-db", `no VIP with id or name "only-db" in --vpc web`},
		{"vip-3", "VIP vip-3 is not in --vpc web"},
		{"twin", `2 VIPs in --vpc web have name "twin"`},
		{"nope", `no VIP with id or name "nope" in --vpc web`},
	} {
		h.stderr.Reset()
		if code := h.run("vpc", "vip", "delete", c.vip, "--vpc", "web", "--yes", "--no-wait"); code != ExitUsage {
			t.Errorf("%s: exit %d", c.vip, code)
		}
		if s := h.stderr.String(); !strings.Contains(s, c.want) {
			t.Errorf("%s: stderr = %s", c.vip, s)
		}
	}
	if code := h.run("vpc", "vip", "firewall-rule", "list", "--vip", "only-db", "--vpc", "web"); code != ExitUsage {
		t.Errorf("firewall-rule list on another VPC's VIP: exit %d", code)
	}
	if len(h.fake.WriteURLs) != 1 {
		t.Errorf("a refused VIP was written: %v", h.fake.WriteURLs)
	}
	// Unable to list VIPs, a VPC-scoped command cannot check the VIP is in the
	// VPC, so it stops rather than sending the value as an id.
	h.fake.Canned["GET /v1/vips"] = fakeapi.Canned{Status: 500, Body: `{"title":"down"}`}
	if code := h.run("vpc", "vip", "delete", "vip-3", "--vpc", "web", "--yes", "--no-wait"); code == ExitOK {
		t.Errorf("an unlistable VIP was used as an id")
	}
	if len(h.fake.WriteURLs) != 1 {
		t.Errorf("writes %v", h.fake.WriteURLs)
	}
}
