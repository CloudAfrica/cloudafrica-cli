package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

func TestElevatePollsUntilPaired(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.PairOnPoll = 3
	var opened []string
	h.browser = func(u string) error { opened = append(opened, u); return nil }
	h.mustRun("auth", "elevate")
	if h.fake.ElevationCalls != 3 {
		t.Errorf("elevation calls = %d, want 3", h.fake.ElevationCalls)
	}
	if len(opened) != 1 || opened[0] != h.fake.URL+"/pair?h=abc" {
		t.Errorf("opened = %v, want the pair page once", opened)
	}
	if s := h.stderr.String(); !strings.Contains(s, "ABCD-1234") || !strings.Contains(s, "Paired") {
		t.Errorf("stderr = %s", s)
	}
	if h.stdout.Len() != 0 {
		t.Errorf("stdout = %s", h.stdout)
	}
	if got := h.fake.ElevationBody[0]["label"]; got != elevateLabel {
		t.Errorf("label = %v", got)
	}
}

func TestElevateAlreadyPairedOpensNothing(t *testing.T) {
	h := newHarness(t)
	g := protected()
	g.Elevated = true
	h.seed("default", g, time.Now().Add(time.Hour))
	h.browser = func(u string) error { t.Errorf("opened %s", u); return nil }
	h.mustRun("auth", "elevate")
	if h.fake.ElevationCalls != 1 || !strings.Contains(h.stderr.String(), "already allowed") {
		t.Errorf("calls %d, stderr %s", h.fake.ElevationCalls, h.stderr)
	}
}

func TestElevateTimesOutWithExit3(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.elevateTimeout = 100 * time.Millisecond
	h.browser = func(string) error { return nil }
	if code := h.run("auth", "elevate"); code != ExitTimeout || !strings.Contains(h.stderr.String(), "not confirmed") {
		t.Errorf("exit %d, stderr %s", code, h.stderr)
	}
	if h.fake.ElevationCalls < 2 {
		t.Errorf("elevation calls = %d: did not poll", h.fake.ElevationCalls)
	}
}

const testSecret = "cae_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// An API token pairs a new secret, printed alone on stdout even when -o
// would make it JSON, and later commands carry it on every request.
func TestElevateEnvTokenPairsANewSecretAndPrintsItAlone(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	h.fake.PairOnPoll = 2
	var opened []string
	h.browser = func(u string) error { opened = append(opened, u); return nil }
	h.mustRun("auth", "elevate", "-o", "json")
	secret := strings.TrimSuffix(h.stdout.String(), "\n")
	if !wellFormedElevation(secret) || strings.Count(h.stdout.String(), "\n") != 1 || len(secret) != 4+43 {
		t.Fatalf("stdout = %q", h.stdout)
	}
	if len(opened) != 1 || !strings.Contains(h.stderr.String(), "ABCD-1234") ||
		!strings.Contains(h.stderr.String(), "Paired: this API token") {
		t.Errorf("opened %v, stderr %s", opened, h.stderr)
	}
	if got := h.fake.ElevationBody[0]["label"]; got != elevateTokenLabel {
		t.Errorf("label = %v", got)
	}
	for _, line := range h.fake.ElevationHeaders {
		if !strings.HasSuffix(line, " "+secret) {
			t.Errorf("elevation request without the secret: %q", line)
		}
	}

	h.env["CLOUDAFRICA_ELEVATION"] = secret
	h.fake.ElevationHeaders = nil
	h.mustRun("vm", "list")
	h.mustRun("vm", "delete", "vm-1", "--yes")
	if len(h.fake.ElevationHeaders) < 2 {
		t.Fatalf("requests = %v", h.fake.ElevationHeaders)
	}
	for _, line := range h.fake.ElevationHeaders {
		if !strings.HasSuffix(line, " "+secret) {
			t.Errorf("request without the secret: %q", line)
		}
	}
}

// A set secret that lapsed is paired again rather than replaced, so a
// script's value survives a re-elevate after the window.
func TestElevateEnvTokenRepairsTheSetSecret(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	h.env["CLOUDAFRICA_ELEVATION"] = testSecret
	h.fake.PairOnPoll = 3
	var opened []string
	h.browser = func(u string) error { opened = append(opened, u); return nil }
	h.mustRun("auth", "elevate")
	if h.stdout.String() != testSecret+"\n" || len(opened) != 1 ||
		!strings.Contains(h.stderr.String(), "Paired: this API token") {
		t.Errorf("stdout %q, opened %v, stderr %s", h.stdout, opened, h.stderr)
	}
	for _, line := range h.fake.ElevationHeaders {
		if !strings.HasSuffix(line, " "+testSecret) {
			t.Errorf("request without the set secret: %q", line)
		}
	}
	h.mustRun("vm", "delete", "vm-1", "--yes")
}

// The secret is the command's whole output: failing to write it fails.
func TestElevateEnvTokenFailsWhenTheSecretCannotBeWritten(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	h.fake.PairOnPoll = 1
	h.failStdout = true
	if code := h.run("auth", "elevate"); code == 0 || !strings.Contains(h.stderr.String(), "writing the elevation secret") {
		t.Errorf("exit %d, stderr %s", code, h.stderr)
	}
}

// A token that is already paired by its secret prints it and opens nothing.
func TestElevateEnvTokenAlreadyPaired(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	h.env["CLOUDAFRICA_ELEVATION"] = testSecret
	h.fake.PairToken("cak_protected", testSecret)
	h.browser = func(u string) error { t.Errorf("opened %s", u); return nil }
	h.mustRun("auth", "elevate")
	if h.stdout.String() != testSecret+"\n" || h.fake.ElevationCalls != 1 {
		t.Errorf("stdout %q, calls %d", h.stdout, h.fake.ElevationCalls)
	}
}

// A timeout prints nothing on stdout, so the documented assignment fails.
func TestElevateEnvTokenTimeoutPrintsNoSecret(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	h.elevateTimeout = 100 * time.Millisecond
	h.browser = func(string) error { return nil }
	if code := h.run("auth", "elevate"); code != ExitTimeout || h.stdout.Len() != 0 {
		t.Errorf("exit %d, stdout %q", code, h.stdout)
	}
}

func TestElevationEnvRefusedBeforeAnyRequest(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"malformed":     {"CLOUDAFRICA_TOKEN": "cak_protected", "CLOUDAFRICA_ELEVATION": "cae_short"},
		"no prefix":     {"CLOUDAFRICA_TOKEN": "cak_protected", "CLOUDAFRICA_ELEVATION": strings.TrimPrefix(testSecret, "cae_") + "AAAA"},
		"bad character": {"CLOUDAFRICA_TOKEN": "cak_protected", "CLOUDAFRICA_ELEVATION": testSecret + "="},
		"42 random":     {"CLOUDAFRICA_TOKEN": "cak_protected", "CLOUDAFRICA_ELEVATION": testSecret[:len(testSecret)-1]},
		"without token": {"CLOUDAFRICA_ELEVATION": testSecret},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			h.seed("default", protected(), time.Now().Add(time.Hour))
			for k, v := range env {
				h.env[k] = v
			}
			for _, args := range [][]string{{"vm", "list"}, {"auth", "elevate"}} {
				if code := h.run(args...); code != ExitUsage {
					t.Errorf("%v: exit %d, stderr %s", args, code, h.stderr)
				}
			}
			if n := len(h.fake.ElevationHeaders); n != 0 {
				t.Errorf("%d requests sent", n)
			}
			if name == "without token" && !strings.Contains(h.stderr.String(), "unset CLOUDAFRICA_ELEVATION") {
				t.Errorf("stderr %s", h.stderr)
			}
			// The commands that diagnose it still run.
			h.mustRun("auth", "status")
			h.mustRun("version")
		})
	}
}

// A blank value is unset, as a blank CLOUDAFRICA_TOKEN is.
func TestElevationEnvBlankIsUnset(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.env["CLOUDAFRICA_ELEVATION"] = "  "
	h.mustRun("vm", "list")
}

func TestAuthStatusShowsWhetherElevationIsSet(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	h.mustRun("auth", "status", "-o", "json")
	var v map[string]any
	if err := json.Unmarshal(h.stdout.Bytes(), &v); err != nil || v["elevation"] != false {
		t.Errorf("status %s (%v)", h.stdout, err)
	}
	h.env["CLOUDAFRICA_ELEVATION"] = testSecret
	h.mustRun("auth", "status", "-o", "json")
	if err := json.Unmarshal(h.stdout.Bytes(), &v); err != nil || v["elevation"] != true ||
		strings.Contains(h.stdout.String(), testSecret) {
		t.Errorf("status %s (%v)", h.stdout, err)
	}
	h.mustRun("auth", "status", "-o", "table")
	if s := h.stdout.String(); !strings.Contains(s, "CLOUDAFRICA_ELEVATION set") || strings.Contains(s, testSecret) {
		t.Errorf("table %s", s)
	}
}

func TestElevateRefusesAPlainHTTPPage(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.PairURL = "http://portal.example/pair?h=abc"
	h.browser = func(u string) error { t.Errorf("opened %s", u); return nil }
	if code := h.run("auth", "elevate"); code != ExitAPI || !strings.Contains(h.stderr.String(), "plain http") {
		t.Errorf("exit %d, stderr %s", code, h.stderr)
	}
}

// Pairing a sign-in without protected:compute would not allow protected
// actions, so elevate refuses before asking.
func TestElevateWithoutProtectedScopeOffersLogin(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	if code := h.run("auth", "elevate"); code != ExitStepUp || h.fake.ElevationCalls != 0 {
		t.Fatalf("exit %d, elevation calls %d", code, h.fake.ElevationCalls)
	}
	if s := h.stderr.String(); !strings.Contains(s, "not granted any protected: scope") || !strings.Contains(s, "`ca auth login`") {
		t.Errorf("stderr %s", s)
	}
}

func TestElevateRefusesAPageWithoutACode(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.NoPairCode = true
	h.browser = func(u string) error { t.Errorf("opened %s", u); return nil }
	if code := h.run("auth", "elevate"); code != ExitAPI || !strings.Contains(h.stderr.String(), "no confirmation code") {
		t.Errorf("exit %d, stderr %s", code, h.stderr)
	}
}

func TestElevateNotSignedInExits4(t *testing.T) {
	h := newHarness(t)
	if code := h.run("auth", "elevate"); code != ExitAuth {
		t.Errorf("exit %d", code)
	}
}

func TestRequiredScope(t *testing.T) {
	for _, c := range []struct {
		e    cloudafrica.APIError
		want string
	}{
		{cloudafrica.APIError{Detail: "This token is missing the required scope: protected:compute"}, "protected:compute"},
		{cloudafrica.APIError{Detail: "This token is missing the required scope: read:network."}, "read:network"},
		{cloudafrica.APIError{Scope: "write:compute", Detail: "This token is missing the required scope: read:x"}, "write:compute"},
		{cloudafrica.APIError{Detail: "no scope here"}, ""},
		{cloudafrica.APIError{}, ""},
	} {
		if got := requiredScope(&c.e); got != c.want {
			t.Errorf("requiredScope(%+v) = %q, want %q", c.e, got, c.want)
		}
	}
}

func TestDeleteRunningVMHintsStopFirst(t *testing.T) {
	h := newHarness(t)
	g := protected()
	g.Elevated = true
	h.seed("default", g, time.Now().Add(time.Hour))
	h.fake.VMs = []map[string]any{{"id": "vm-1", "status": "running"}}
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitAPI {
		t.Fatalf("exit %d", code)
	}
	if s := h.stderr.String(); !strings.Contains(s, "conflict") || !strings.Contains(s, "ca vms shutdown vm-1`") {
		t.Errorf("stderr = %s", s)
	}
	h.fake.VMs[0]["status"] = "stopped"
	h.mustRun("vm", "delete", "vm-1", "--yes")
}

func TestListSSHKeysVPCsAndDatacenters(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.SSHKeys = []map[string]any{{"id": "k1", "name": "a"}, {"id": "k2"}, {"id": "k3"}}
	h.fake.VPCs = []map[string]any{{"id": "v1"}, {"id": "v2"}, {"id": "v3"}}
	h.fake.Datacenters = []map[string]any{{"slug": "jhb1", "label": "Johannesburg"}, {"slug": "cpt1"}}
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"ssh-key", "list"}, "k1 k2 k3"},
		{[]string{"vpc", "list"}, "v1 v2 v3"},
		{[]string{"datacenter", "list"}, "jhb1 cpt1"},
	} {
		h.mustRun(append(c.args, "-q")...)
		if got := strings.Join(strings.Fields(h.stdout.String()), " "); got != c.want {
			t.Errorf("%v -q = %q, want %q", c.args, got, c.want)
		}
	}
	h.mustRun("datacenter", "list", "-o", "json")
	var dcs []map[string]any
	if err := json.Unmarshal(h.stdout.Bytes(), &dcs); err != nil || len(dcs) != 2 || dcs[0]["label"] != "Johannesburg" {
		t.Errorf("json = %s (%v)", h.stdout, err)
	}
	h.fake.VPCs = nil
	h.mustRun("vpc", "list", "-o", "json")
	if got := strings.TrimSpace(h.stdout.String()); got != "[]" {
		t.Errorf("empty list = %s", got)
	}
}

// A profile signed in before read:network was a default is told to sign in again.
func TestListWithoutItsScopeHintsLogin(t *testing.T) {
	h := newHarness(t)
	h.seed("default", &fakeapi.Grant{Scopes: []string{"read:compute", "write:compute"}}, time.Now().Add(time.Hour))
	if code := h.run("vpc", "list"); code != ExitAPI {
		t.Fatalf("exit %d", code)
	}
	if s := h.stderr.String(); !strings.Contains(s, "not granted read:network") || !strings.Contains(s, "`ca auth login`") {
		t.Errorf("stderr = %s", s)
	}
}

func TestListAllRefusesARepeatedCursor(t *testing.T) {
	next := "c1"
	calls := 0
	_, err := listAll(func(*string) (*[]int, *string, error) {
		calls++
		return &[]int{calls}, &next, nil
	})
	if err == nil || calls != 2 {
		t.Errorf("err %v after %d calls", err, calls)
	}
	cycle := []string{"a", "b"}
	calls = 0
	_, err = listAll(func(*string) (*[]int, *string, error) {
		calls++
		if calls > 10 {
			return nil, nil, errors.New("listAll did not stop on the cycle")
		}
		return &[]int{calls}, &cycle[calls%2], nil
	})
	if err == nil || calls != 3 {
		t.Errorf("cycle: err %v after %d calls", err, calls)
	}
}

// Any protected: scope is worth pairing, not only protected:compute.
func TestElevateAcceptsAnyProtectedScope(t *testing.T) {
	h := newHarness(t)
	h.seed("default", &fakeapi.Grant{Scopes: []string{"read:network", "protected:network"}}, time.Now().Add(time.Hour))
	h.fake.PairOnPoll = 1
	h.browser = func(string) error { return nil }
	h.mustRun("auth", "elevate")
	if h.fake.ElevationCalls != 1 {
		t.Errorf("elevation calls = %d", h.fake.ElevationCalls)
	}
}
