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
	if s := h.stderr.String(); !strings.Contains(s, "CLOUDAFRICA_TOKEN is a user API token") || strings.Contains(s, "ca auth") {
		t.Errorf("env-token hint: %s", s)
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
