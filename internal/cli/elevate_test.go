package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"
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

func TestElevateWithEnvTokenIsUsageError(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_test"
	if code := h.run("auth", "elevate"); code != ExitUsage || h.fake.ElevationCalls != 0 {
		t.Errorf("exit %d, calls %d", code, h.fake.ElevationCalls)
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
	if s := h.stderr.String(); !strings.Contains(s, "not granted protected:compute") || !strings.Contains(s, "`ca auth login`") {
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
	if s := h.stderr.String(); !strings.Contains(s, "conflict") || !strings.Contains(s, "ca vm shutdown vm-1 --wait") {
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
	h.seed("default", compute(), time.Now().Add(time.Hour))
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
