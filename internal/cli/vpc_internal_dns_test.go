package cli

import (
	"strings"
	"testing"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

const internalDnsBody = `{"state":"winding_down","resolverAddresses":["10.0.0.253","10.0.0.252"],"guestsUseResolver":true,
	"drainingUntil":"2026-11-26T10:00:00Z","reason":"resolver_unreachable","reasonDetail":"resolver unreachable",
	"records":[{"name":"web-01.internal","type":"A","ttl":60,"values":["10.0.0.5"],"zone":"internal"}],
	"clashes":[],"skipped":[]}`

// get prints every column of the state, set sends only the enabled flag and prints the answer
// it gets, and --vpc resolves a name before either calls the API.
func TestVpcInternalDnsGetAndSet(t *testing.T) {
	h := s3Harness(t)
	h.fake.VPCs = []map[string]any{{"id": "vpc-1", "name": "web"}}
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/vpcs/vpc-1/internal-dns": {Body: internalDnsBody},
		"PUT /v1/vpcs/vpc-1/internal-dns": {Body: internalDnsBody},
	})
	want := row("winding_down", "10.0.0.253,", "10.0.0.252", "yes", stamp("2026-11-26T10:00:00Z"), "resolver", "unreachable")
	head, rows := table(h, "vpcs", "internal-dns", "get", "--vpc", "web")
	if head != "STATE RESOLVERS GUESTS USE RESOLVER DRAINING UNTIL REASON" || rows != want {
		t.Errorf("get %q %s want %s", head, rows, want)
	}
	h.mustRun("vpcs", "internal-dns", "get", "--vpc", "web", "-o", "json")
	contains(t, h.stdout.String(), `"name": "web-01.internal"`, `"reason": "resolver_unreachable"`)

	head, rows = table(h, "vpcs", "internal-dns", "set", "--vpc", "web", "--enabled=false")
	if head != "STATE RESOLVERS GUESTS USE RESOLVER DRAINING UNTIL REASON" || rows != want {
		t.Errorf("set %q %s", head, rows)
	}
	if len(h.fake.WriteURLs) != 1 || h.fake.WriteURLs[0] != "PUT /v1/vpcs/vpc-1/internal-dns" {
		t.Fatalf("writes %v", h.fake.WriteURLs)
	}
	if b := h.fake.Bodies[0]; len(b) != 1 || b["enabled"] != false {
		t.Errorf("body = %v", b)
	}
}

// set without --enabled sends nothing, and a 409 refusal reaches the user with its sentence.
func TestVpcInternalDnsSetRefusals(t *testing.T) {
	h := s3Harness(t)
	h.fake.VPCs = []map[string]any{{"id": "vpc-1", "name": "web"}}
	canned(h, map[string]fakeapi.Canned{
		"PUT /v1/vpcs/vpc-1/internal-dns": {Status: 409, Body: `{"type":"about:blank","title":"Conflict","status":409,"detail":"two VMs claim the name db"}`},
	})
	if code := h.run("vpcs", "internal-dns", "set", "--vpc", "web"); code == 0 || len(h.fake.WriteURLs) != 0 {
		t.Errorf("no --enabled: exit %d, writes %v", code, h.fake.WriteURLs)
	}
	if code := h.run("vpcs", "internal-dns", "set", "--vpc", "web", "--enabled=true"); code == 0 {
		t.Errorf("a 409 exited 0")
	}
	if got := h.stderr.String(); !strings.Contains(got, "two VMs claim the name db") {
		t.Errorf("stderr lacks the refusal sentence: %s", got)
	}
}
