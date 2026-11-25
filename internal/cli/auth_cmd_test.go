package cli

import (
	"context"
	"os"
	"slices"
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
	// The token lacks protected:compute: another token, or the profile.
	if s := h.stderr.String(); !strings.Contains(s, "CLOUDAFRICA_TOKEN was not granted protected:compute") ||
		!strings.Contains(s, "use a token that has it, then `ca auth elevate`") ||
		!strings.Contains(s, "Or unset it, run `ca auth login`, then `ca auth elevate`") ||
		strings.Contains(s, "cannot pair") {
		t.Errorf("env-token insufficient-scope hint: %s", s)
	}
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp {
		t.Fatalf("not paired: exit %d", code)
	}
	if s := h.stderr.String(); !strings.Contains(s, "CLOUDAFRICA_TOKEN is not paired") ||
		!strings.Contains(s, "CLOUDAFRICA_ELEVATION=$(ca auth elevate)") ||
		strings.Contains(s, "not granted") || strings.Contains(s, "no longer paired") {
		t.Errorf("env-token not-paired hint: %s", s)
	}
	h.env["CLOUDAFRICA_ELEVATION"] = testSecret
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp {
		t.Fatalf("lapsed: exit %d", code)
	}
	if s := h.stderr.String(); !strings.Contains(s, "CLOUDAFRICA_ELEVATION is no longer paired") ||
		!strings.Contains(s, "with this same CLOUDAFRICA_ELEVATION set") || strings.Contains(s, "is not paired") {
		t.Errorf("env-token lapsed hint: %s", s)
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
	if got := h.fake.Authorizations[0].Get("account"); got != "acc-1" {
		t.Errorf("the step-up sign-in asked for account %q, want the profile's acc-1", got)
	}
}

// A refusal for a scope login does not request by default names the
// --scope that requests it, and the profile keeps that scope for later sign-ins.
func TestNonDefaultScopeHintsLoginScopeAndIsKept(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.Require = map[string]string{"/v1/billing/": "write:billing"}
	if code := h.run("api", "POST", "/v1/billing/top-ups"); code != ExitAPI {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(h.stderr.String(), "`ca auth login --scope write:billing`") {
		t.Errorf("hint: %s", h.stderr)
	}
	h.mustRun("auth", "login", "--scope", "write:billing")
	h.mustRun("api", "POST", "/v1/billing/top-ups")
	h.mustRun("auth", "login")
	if !slices.Contains(strings.Fields(h.fake.Authorizations[1].Get("scope")), "write:billing") {
		t.Errorf("a later sign-in dropped write:billing: %v", h.fake.Authorizations[1])
	}

	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	if code := h.run("api", "POST", "/v1/billing/top-ups"); code != ExitAPI {
		t.Fatalf("env token: exit %d", code)
	}
	if s := h.stderr.String(); !strings.Contains(s, "CLOUDAFRICA_TOKEN was not granted write:billing") {
		t.Errorf("env-token hint: %s", s)
	}
}

func TestLoginScopeFlagRequestsAndPersists(t *testing.T) {
	h := newHarness(t)
	if code := h.run("auth", "login", "--scope", "nocolon"); code != ExitUsage || len(h.fake.Authorizations) != 0 {
		t.Fatalf("bad scope: exit %d, authorizations %d", code, len(h.fake.Authorizations))
	}
	h.mustRun("auth", "login", "--scope", "write:billing", "--scope", "read:compute")
	got := strings.Fields(h.fake.Authorizations[0].Get("scope"))
	if !slices.Equal(got, append(slices.Clone(defaultScopes), "write:billing")) {
		t.Errorf("scope = %v", got)
	}
	if p, _ := h.store.Load("default"); !slices.Equal(p.ExtraScopes, []string{"write:billing"}) {
		t.Errorf("extra scopes = %v", p.ExtraScopes)
	}
}

// The account the sign-in is for travels as the authorize request's account
// parameter: named with --account, else the profile's own account on the same
// endpoint, else absent.
func TestLoginNamesTheAccountOnTheAuthorizeRequest(t *testing.T) {
	h := newHarness(t)
	h.mustRun("auth", "login")
	if _, has := h.fake.Authorizations[0]["account"]; has {
		t.Errorf("a first sign-in named an account: %v", h.fake.Authorizations[0])
	}
	h.mustRun("auth", "login")
	if got := h.fake.Authorizations[1].Get("account"); got != "acc-1" {
		t.Errorf("a re-sign-in asked for account %q, want the profile's acc-1", got)
	}
	// --account wins over the profile's own account, and the profile moves to it.
	h.fake.AccountID = "acc-9"
	h.mustRun("auth", "login", "--account", "acc-9")
	if got := h.fake.Authorizations[2].Get("account"); got != "acc-9" {
		t.Errorf("--account on a signed-in profile sent %q, want acc-9", got)
	}
	if p, _ := h.store.Load("default"); p.AccountID != "acc-9" {
		t.Errorf("profile account = %q, want acc-9", p.AccountID)
	}
	h.mustRun("auth", "login", "--profile", "other", "--account", "acc-9")
	if got := h.fake.Authorizations[3].Get("account"); got != "acc-9" {
		t.Errorf("--account on a new profile sent %q, want acc-9", got)
	}
}

// The hint names the selected profile, so following it cannot touch another one.
func TestLoginRefusedStoredAccountHintNamesTheProfile(t *testing.T) {
	h := newHarness(t)
	h.seed("globex", protected(), time.Now().Add(time.Hour))
	h.fake.DenyConsent = true
	if code := h.run("auth", "login", "--profile", "globex"); code != ExitAuth {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if s := h.stderr.String(); !strings.Contains(s, "`ca auth login --profile globex --account <id>`") ||
		!strings.Contains(s, "`ca auth logout --profile globex`") {
		t.Errorf("stderr: %s", s)
	}
}

// The same endpoint spelled differently is the same endpoint.
func TestLoginReusesTheProfileAccountForTheSameOriginSpelledDifferently(t *testing.T) {
	h := newHarness(t)
	p := h.seed("default", protected(), time.Now().Add(time.Hour))
	p.Endpoint = strings.Replace(p.Endpoint, "http://", "HTTP://", 1)
	if err := h.store.Save("default", p); err != nil {
		t.Fatal(err)
	}
	h.mustRun("auth", "login")
	if got := h.fake.Authorizations[0].Get("account"); got != "acc-1" {
		t.Errorf("asked for account %q, want acc-1", got)
	}
}

// A profile's account is not carried to another endpoint, where it is unknown.
func TestLoginDoesNotReuseTheProfileAccountOnAnotherEndpoint(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	other := fakeapi.New(t)
	h.env["CLOUDAFRICA_ENDPOINT"] = other.URL
	h.mustRun("auth", "login")
	if len(other.Authorizations) != 1 {
		t.Fatalf("authorizations on the other endpoint: %d", len(other.Authorizations))
	}
	if _, has := other.Authorizations[0]["account"]; has {
		t.Errorf("acc-1 of another endpoint was sent: %v", other.Authorizations[0])
	}
}

// A session for another account than the one asked for is a failure that
// saves nothing, so a script never gets a success bound to the wrong account.
func TestLoginRefusesASessionForAnotherAccount(t *testing.T) {
	h := newHarness(t)
	before := h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.AccountID = "acc-2"
	if code := h.run("auth", "login", "--account", "acc-9"); code != ExitAuth {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if s := h.stderr.String(); !strings.Contains(s, "asked to sign in to account acc-9 but the session is for acc-2") ||
		!strings.Contains(s, "--profile <name> --account <id>") || strings.Contains(s, "Signed in") {
		t.Errorf("stderr: %s", s)
	}
	if p, _ := h.store.Load("default"); p.AccountID != "acc-1" || p.AccessToken != before.AccessToken {
		t.Errorf("the profile changed: %+v", p)
	}
	if n := len(h.fake.Revoked); n != 1 || h.fake.Revoked[0] == before.RefreshToken {
		t.Errorf("revoked %v: want only the grant just issued, not the profile's %s", h.fake.Revoked, before.RefreshToken)
	}
	h.fake.RevokeFails = true
	h.stderr.Reset()
	if code := h.run("auth", "login", "--account", "acc-9"); code != ExitAuth ||
		!strings.Contains(h.stderr.String(), "the new session could not be revoked") {
		t.Errorf("exit %d, a failed cleanup was not reported: %s", code, h.stderr)
	}
}

// When the account cannot be read back, a named account cannot be confirmed,
// so nothing is saved and the stored account is not wiped.
func TestLoginRefusesASessionItCannotConfirm(t *testing.T) {
	h := newHarness(t)
	before := h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.Canned = map[string]fakeapi.Canned{"GET /v1/account": {Status: 500}}
	if code := h.run("auth", "login", "--account", "acc-9"); code != ExitAuth {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if s := h.stderr.String(); !strings.Contains(s, "could not confirm the session is for account acc-9") ||
		strings.Contains(s, "Signed in") {
		t.Errorf("stderr: %s", s)
	}
	if p, _ := h.store.Load("default"); p.AccountID != "acc-1" || p.AccessToken != before.AccessToken {
		t.Errorf("the profile changed: %+v", p)
	}
	if n := len(h.fake.Revoked); n != 1 || h.fake.Revoked[0] == before.RefreshToken {
		t.Errorf("revoked %v: want only the grant just issued, not the profile's %s", h.fake.Revoked, before.RefreshToken)
	}
	// With no account named there is nothing to confirm: the sign-in is kept, as before.
	h.mustRun("auth", "login", "--profile", "fresh")
	if p, _ := h.store.Load("fresh"); p == nil || p.AccessToken == "" || p.AccountID != "" {
		t.Errorf("an unnamed sign-in was not kept as before: %+v", p)
	}
}

// When the consent page refuses the account a profile reuses, the error names
// the way out instead of repeating the command that just failed.
func TestLoginRefusedStoredAccountNamesTheWayOut(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.DenyConsent = true
	if code := h.run("auth", "login"); code != ExitAuth {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if s := h.stderr.String(); !strings.Contains(s, "sign-in was declined for account acc-1") ||
		!strings.Contains(s, "`ca auth login --account <id>`") || !strings.Contains(s, "`ca auth logout`") {
		t.Errorf("stderr: %s", s)
	}
	h.stderr.Reset()
	if code := h.run("auth", "login", "--account", "acc-9"); code != ExitAuth ||
		strings.Contains(h.stderr.String(), "`ca auth logout`") {
		t.Errorf("a named account got the stored-account hint: exit %d, %s", code, h.stderr)
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

// A Require entry on a route the fake serves still answers with that route,
// so a test that only adds a scope check does not get an empty body.
func TestRequireOnAKnownRouteServesTheRoute(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	h.fake.VPCs = []map[string]any{{"id": "v1", "name": "web"}}
	h.fake.Require = map[string]string{"/v1/vpcs": "read:network"}
	h.mustRun("vpc", "list", "-o", "json")
	if !strings.Contains(h.stdout.String(), `"name": "web"`) {
		t.Errorf("stdout: %s", h.stdout)
	}
}
