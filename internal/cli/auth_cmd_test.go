package cli

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

func TestLogoutKeepsProfileWhenRevokeFails(t *testing.T) {
	h := signedIn(t)
	h.fake.RevokeFails = true
	if code := h.run("auth", "logout"); code == ExitOK {
		t.Fatal("logout succeeded although the revoke failed")
	}
	if _, err := h.store.Load("default"); err != nil {
		t.Errorf("profile gone after a failed revoke: %v", err)
	}
	if !strings.Contains(h.stderr.String(), h.store.Path("default")) {
		t.Errorf("stderr does not name the file to delete: %s", h.stderr)
	}
}

func TestLoginWarnsWhenOldGrantRevokeFails(t *testing.T) {
	h := signedIn(t)
	h.fake.RevokeFails = true
	h.mustRun("auth", "login")
	if !strings.Contains(h.stderr.String(), "could not be revoked") {
		t.Errorf("no warning: %s", h.stderr)
	}
}

func TestLoginConsentDeniedStoresNothing(t *testing.T) {
	h := newHarness(t)
	h.fake.DenyConsent = true
	if code := h.run("auth", "login"); code != ExitAuth || !strings.Contains(h.stderr.String(), "access_denied") {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if names, _ := h.store.Names(); len(names) != 0 {
		t.Errorf("profiles = %v", names)
	}
}

func TestLoginTimesOutWhenBrowserNeverReturns(t *testing.T) {
	h := newHarness(t)
	h.browser = func(string) error { return nil }
	h.loginTimeout = 300 * time.Millisecond
	if code := h.run("auth", "login"); code != ExitAuth {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if names, _ := h.store.Names(); len(names) != 0 {
		t.Errorf("profiles = %v", names)
	}
}

func TestCorruptProfileIsReplacedByLoginAndDeletedByLogout(t *testing.T) {
	h := newHarness(t)
	corrupt := func() {
		if err := os.WriteFile(h.store.Path("default"), []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	corrupt()
	h.mustRun("auth", "login")
	if strings.Count(h.stderr.String(), "replacing it") != 1 {
		t.Errorf("login stderr: %s", h.stderr)
	}
	if p, err := h.store.Load("default"); err != nil || p.RefreshToken == "" {
		t.Fatalf("profile after login = %+v, %v", p, err)
	}
	corrupt()
	h.mustRun("auth", "logout")
	if !strings.Contains(h.stderr.String(), "deleting it") || len(h.fake.Revoked) != 0 {
		t.Errorf("logout stderr %s, revoked %v", h.stderr, h.fake.Revoked)
	}
	if _, err := os.Stat(h.store.Path("default")); !os.IsNotExist(err) {
		t.Errorf("corrupt file still present: %v", err)
	}
}

// logout waits for a holder of the profile lock (a refresh in another
// process), so that refresh cannot save the profile back after the delete.
func TestLogoutWaitsForProfileLock(t *testing.T) {
	h := signedIn(t)
	release, err := h.store.Lock(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan int)
	go func() { done <- h.run("auth", "logout") }()
	select {
	case code := <-done:
		release()
		t.Fatalf("logout finished (exit %d) while the lock was held", code)
	case <-time.After(300 * time.Millisecond):
	}
	release()
	if code := <-done; code != ExitOK {
		t.Fatalf("logout exit %d", code)
	}
}

func TestEnvToken401ExitsAuthWithHint(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "not-a-key"
	if code := h.run("vm", "list"); code != ExitAuth || !strings.Contains(h.stderr.String(), "CLOUDAFRICA_TOKEN") {
		t.Errorf("exit %d, stderr %s", code, h.stderr)
	}
}

// With CLOUDAFRICA_TOKEN there is no profile to re-authorize, so even on a
// terminal a step-up refusal exits 5 without a prompt.
func TestEnvTokenStepUpOnTerminalDoesNotPrompt(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_test"
	h.tty = true
	h.stdin = "y\n"
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(h.stderr.String(), "[y/N]") || len(h.fake.Authorizations) != 0 {
		t.Errorf("prompted or re-authorized: %s", h.stderr)
	}
	// The token lacks protected:compute, and ca cannot pair a token: the
	// remedy is the profile.
	if s := h.stderr.String(); !strings.Contains(s, "CLOUDAFRICA_TOKEN was not granted protected:compute") ||
		!strings.Contains(s, "Unset it, run `ca auth login`, then `ca auth elevate`") ||
		strings.Contains(s, "use a token that has it") {
		t.Errorf("env-token insufficient-scope hint: %s", s)
	}
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp {
		t.Fatalf("not paired: exit %d", code)
	}
	if s := h.stderr.String(); !strings.Contains(s, "CLOUDAFRICA_TOKEN is a user API token") ||
		strings.Contains(s, "not granted") || strings.Contains(s, "ca auth") {
		t.Errorf("env-token not-paired hint: %s", s)
	}
}

// not-paired offers pairing, never a new sign-in.
func TestStepUpInteractiveNotPairedOffersElevate(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.PairOnPoll = 2
	h.tty = true
	h.stdin = "y\n"
	h.mustRun("vm", "delete", "vm-1", "--yes")
	if !strings.Contains(h.stderr.String(), "`ca auth elevate`") || h.fake.DeleteCalls != 2 || len(h.fake.Authorizations) != 0 {
		t.Errorf("delete calls %d, authorizations %d, stderr %s", h.fake.DeleteCalls, len(h.fake.Authorizations), h.stderr)
	}
}

func TestStepUpInteractiveNotPairedDeclinedPairsNothing(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.tty = true
	h.stdin = "n\n"
	h.browser = func(u string) error { t.Errorf("opened %s", u); return nil }
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp {
		t.Fatalf("exit %d", code)
	}
	if h.fake.DeleteCalls != 1 || h.fake.ElevationCalls != 0 || len(h.fake.Authorizations) != 0 {
		t.Errorf("delete calls %d, elevation calls %d, authorizations %d",
			h.fake.DeleteCalls, h.fake.ElevationCalls, len(h.fake.Authorizations))
	}
}

// A not-paired refusal that persists after pairing ends the loop: pairing
// is offered once.
func TestStepUpPersistentNotPairedOffersElevateOnce(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.PairOnPoll = 1
	h.fake.AlwaysNotPaired = true
	h.tty = true
	h.stdin = "y\ny\n"
	h.browser = func(string) error { return nil }
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if n := strings.Count(h.stderr.String(), "[y/N]"); n != 1 || h.fake.DeleteCalls != 2 || h.fake.ElevationCalls != 1 {
		t.Errorf("prompts %d, delete calls %d, elevation calls %d", n, h.fake.DeleteCalls, h.fake.ElevationCalls)
	}
}

// A sign-in that is still not granted the scope after signing in again is
// not offered a second sign-in.
func TestStepUpScopeStillMissingAfterLoginOffersLoginOnce(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.fake.GrantScopes = func([]string) []string { return []string{"read:compute", "write:compute"} }
	h.tty = true
	h.stdin = "y\ny\n"
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if n := strings.Count(h.stderr.String(), "[y/N]"); n != 1 || len(h.fake.Authorizations) != 1 || h.fake.DeleteCalls != 2 {
		t.Errorf("prompts %d, authorizations %d, delete calls %d", n, len(h.fake.Authorizations), h.fake.DeleteCalls)
	}
}

// A scope `ca auth login` never requests is not offered a sign-in.
func TestScopeLoginCannotGrantIsNotOfferedLogin(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.tty = true
	h.stdin = "y\n"
	if code := h.run("api", "DELETE", "/v1/vpcs/vpc-1"); code != ExitStepUp {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	s := h.stderr.String()
	if strings.Contains(s, "[y/N]") || len(h.fake.Authorizations) != 0 {
		t.Errorf("offered a sign-in: %s", s)
	}
	if !strings.Contains(s, "`ca` cannot perform this action: it needs protected:network") ||
		strings.Contains(s, "Run `ca auth login`") || strings.Contains(s, "CLOUDAFRICA_TOKEN") {
		t.Errorf("hint: %s", s)
	}
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	if code := h.run("api", "DELETE", "/v1/vpcs/vpc-1"); code != ExitStepUp {
		t.Fatalf("env token: exit %d", code)
	}
	if s := h.stderr.String(); !strings.Contains(s, "`ca` cannot perform this action: it needs protected:network") ||
		strings.Contains(s, "use a token that has it") {
		t.Errorf("env-token hint: %s", s)
	}
}

// When the pairing is not confirmed, the step-up refusal is not retried.
func TestStepUpInteractiveElevateTimeoutStops(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.elevateTimeout = 50 * time.Millisecond
	h.tty = true
	h.stdin = "y\n"
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitTimeout || h.fake.DeleteCalls != 1 {
		t.Errorf("exit %d, delete calls %d", code, h.fake.DeleteCalls)
	}
}

func TestEnvTokenRefusesPlainHTTPEndpoint(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_test"
	h.env["CLOUDAFRICA_ENDPOINT"] = "http://api.example"
	if code := h.run("vm", "list"); code != ExitUsage || !strings.Contains(h.stderr.String(), "plain http") {
		t.Errorf("exit %d, stderr %s", code, h.stderr)
	}
}

// A missing ordinary scope is a permission error: no step-up prompt, exit 1.
func TestOrdinaryScopeRefusalIsNotStepUp(t *testing.T) {
	h := newHarness(t)
	h.seed("default", &fakeapi.Grant{Scopes: []string{"read:compute"}}, time.Now().Add(time.Hour))
	h.tty = true
	h.stdin = "y\n"
	if code := h.run("vm", "start", "vm-1"); code != ExitAPI {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if s := h.stderr.String(); strings.Contains(s, "[y/N]") || strings.Contains(s, "elevate") || len(h.fake.Authorizations) != 0 {
		t.Errorf("treated as step-up: %s", s)
	}
	// The gateway names the scope only in detail.
	if s := h.stderr.String(); !strings.Contains(s, "not granted write:compute") || !strings.Contains(s, "`ca auth login`") {
		t.Errorf("hint does not name the scope: %s", s)
	}
}
