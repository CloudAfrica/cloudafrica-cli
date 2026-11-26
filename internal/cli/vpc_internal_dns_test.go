package cli

import (
	"strings"
	"testing"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

const internalDnsHead = "STATE RESOLVERS GUESTS USE RESOLVER DRAINING UNTIL REASON"

const internalDnsBody = `{"state":"winding_down","resolverAddresses":["10.0.0.253","10.0.0.252"],"guestsUseResolver":true,
	"drainingUntil":"2026-11-26T10:00:00Z","reason":"resolver_unreachable","reasonDetail":"resolver unreachable",
	"records":[{"name":"web-01.internal","type":"A","ttl":60,"values":["10.0.0.5"],"zone":"internal"}],
	"clashes":[],"skipped":[]}`

const internalDnsDegraded = `{"state":"active","resolverAddresses":["10.0.0.253"],"guestsUseResolver":true,
	"records":[],"clashes":[{"name":"db"},{"name":"cache"}],"skipped":[{"name":"vm-9","reason":"invalid_name"}]}`

func internalDnsHarness(t *testing.T, routes map[string]fakeapi.Canned) *harness {
	t.Helper()
	h := s3Harness(t)
	h.fake.VPCs = []map[string]any{{"id": "vpc-1", "name": "web"}}
	canned(h, routes)
	return h
}

// get prints every column of the state, set sends the enabled flag as given (false as well as
// true) and prints the answer, and --vpc resolves a name before either calls the API.
func TestVpcInternalDnsGetAndSet(t *testing.T) {
	h := internalDnsHarness(t, map[string]fakeapi.Canned{
		"GET /v1/vpcs/vpc-1/internal-dns": {Body: internalDnsBody},
		"PUT /v1/vpcs/vpc-1/internal-dns": {Body: internalDnsBody},
	})
	want := row("winding_down", "10.0.0.253,", "10.0.0.252", "yes", stamp("2026-11-26T10:00:00Z"), "resolver", "unreachable")
	head, rows := table(h, "vpcs", "internal-dns", "get", "--vpc", "web")
	if head != internalDnsHead || rows != want {
		t.Errorf("get %q %s want %s", head, rows, want)
	}
	if h.stderr.Len() != 0 {
		t.Errorf("a zone with no clashes and nothing left out wrote to stderr: %s", h.stderr)
	}
	h.mustRun("vpcs", "internal-dns", "get", "--vpc", "web", "-o", "json")
	contains(t, h.stdout.String(), `"name": "web-01.internal"`, `"reason": "resolver_unreachable"`)

	head, rows = table(h, "vpcs", "internal-dns", "set", "--vpc", "web", "--enabled=false")
	if head != internalDnsHead || rows != want {
		t.Errorf("set %q %s", head, rows)
	}
	canned(h, map[string]fakeapi.Canned{"PUT /v1/vpcs/vpc-1/internal-dns": {Body: `{"state":"setting_up"}`}})
	head, rows = table(h, "vpcs", "internal-dns", "set", "--vpc", "web", "--enabled=true")
	if head != internalDnsHead || rows != row("setting_up") {
		t.Errorf("set on %q %s", head, rows)
	}
	if len(h.fake.WriteURLs) != 2 || h.fake.WriteURLs[0] != "PUT /v1/vpcs/vpc-1/internal-dns" {
		t.Fatalf("writes %v", h.fake.WriteURLs)
	}
	if b := h.fake.Bodies; len(b) != 2 || len(b[0]) != 1 || b[0]["enabled"] != false || len(b[1]) != 1 || b[1]["enabled"] != true {
		t.Errorf("bodies = %v", b)
	}
}

// A zone that leaves names out can read active in the table, so the table owes a stderr line
// naming both counts; -o json and -q stay quiet, and the same line follows a set.
func TestVpcInternalDnsNoteNamesClashesAndSkips(t *testing.T) {
	h := internalDnsHarness(t, map[string]fakeapi.Canned{
		"GET /v1/vpcs/vpc-1/internal-dns": {Body: internalDnsDegraded},
		"PUT /v1/vpcs/vpc-1/internal-dns": {Body: internalDnsDegraded},
	})
	const note = "names claimed more than once: 2; VMs and virtual IPs left out of the zone: 1. -o json lists them."
	for _, args := range [][]string{
		{"vpcs", "internal-dns", "get", "--vpc", "web"},
		{"vpcs", "internal-dns", "set", "--vpc", "web", "--enabled=true"},
	} {
		h.mustRun(append(args, "-o", "table")...)
		if got := strings.TrimSpace(h.stderr.String()); got != note {
			t.Errorf("%v stderr = %q", args, got)
		}
	}
	for _, flag := range []string{"json", "yaml"} {
		h.mustRun("vpcs", "internal-dns", "get", "--vpc", "web", "-o", flag)
		if h.stderr.Len() != 0 {
			t.Errorf("-o %s wrote to stderr: %s", flag, h.stderr)
		}
	}
	h.mustRun("vpcs", "internal-dns", "get", "--vpc", "web", "-q")
	if h.stderr.Len() != 0 {
		t.Errorf("-q wrote to stderr: %s", h.stderr)
	}
	canned(h, map[string]fakeapi.Canned{"GET /v1/vpcs/vpc-1/internal-dns": {Body: `{"state":"active","clashes":[{"name":"db"}],"skipped":[]}`}})
	h.mustRun("vpcs", "internal-dns", "get", "--vpc", "web", "-o", "table")
	if got := strings.TrimSpace(h.stderr.String()); got != "names claimed more than once: 1. -o json lists them." {
		t.Errorf("clashes only: stderr = %q", got)
	}
}

// set without --enabled is a usage error that sends nothing, and a 409 refusal exits 1 with its
// sentence, after the one write it was sent in.
func TestVpcInternalDnsSetRefusals(t *testing.T) {
	h := internalDnsHarness(t, map[string]fakeapi.Canned{
		"PUT /v1/vpcs/vpc-1/internal-dns": {Status: 409, Body: `{"type":"about:blank","title":"Conflict","status":409,"detail":"two VMs claim the name db"}`},
	})
	if code := h.run("vpcs", "internal-dns", "set", "--vpc", "web"); code != ExitUsage || len(h.fake.WriteURLs) != 0 ||
		!strings.Contains(h.stderr.String(), "--enabled is required") {
		t.Errorf("no --enabled: exit %d, writes %v, stderr %s", code, h.fake.WriteURLs, h.stderr)
	}
	if code := h.run("vpcs", "internal-dns", "set", "--vpc", "web", "--enabled=true"); code != 1 || len(h.fake.WriteURLs) != 1 {
		t.Errorf("409: exit %d, writes %v", code, h.fake.WriteURLs)
	}
	if got := h.stderr.String(); !strings.Contains(got, "two VMs claim the name db") {
		t.Errorf("stderr lacks the refusal sentence: %s", got)
	}
}
