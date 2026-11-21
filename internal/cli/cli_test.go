package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
	"github.com/CloudAfrica/cloudafrica-cli/internal/profile"
)

type harness struct {
	t      *testing.T
	fake   *fakeapi.Server
	app    *App
	env    map[string]string
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	stdin  string
	tty    bool
	store  profile.Store
	// browser replaces the default "browser", which follows the authorize
	// redirect to the loopback callback.
	browser func(u string) error
	// cacheDir is App.CacheDir ("" disables the update-check cache).
	cacheDir       string
	loginTimeout   time.Duration
	elevateTimeout time.Duration
	// failStdout makes every write to stdout fail.
	failStdout bool
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("stdout closed") }

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, fake: fakeapi.New(t), store: profile.Store{Dir: t.TempDir()}}
	h.env = map[string]string{"CLOUDAFRICA_ENDPOINT": h.fake.URL}
	return h
}

// followBrowser plays the user's browser: it opens the authorize URL and
// follows the consent redirect to the CLI's loopback callback.
func followBrowser(u string) error {
	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.Body.Close()
}

func (h *harness) run(args ...string) int {
	h.t.Helper()
	h.stdout, h.stderr = &bytes.Buffer{}, &bytes.Buffer{}
	open := h.browser
	if open == nil {
		open = followBrowser
	}
	lt := h.loginTimeout
	if lt == 0 {
		lt = 5 * time.Second
	}
	var stdout io.Writer = h.stdout
	if h.failStdout {
		stdout = failWriter{}
	}
	h.app = &App{
		Version:      "0.1.0",
		Stdout:       stdout,
		Stderr:       h.stderr,
		Stdin:        strings.NewReader(h.stdin),
		Getenv:       func(k string) string { return h.env[k] },
		TTY:          h.tty,
		OpenBrowser:  open,
		Store:        h.store,
		CacheDir:     h.cacheDir,
		LoginTimeout: lt,
		ElevatePoll:  10 * time.Millisecond,
		ElevateTimeout: func() time.Duration {
			if h.elevateTimeout > 0 {
				return h.elevateTimeout
			}
			return 5 * time.Second
		}(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return h.app.Run(ctx, args)
}

func (h *harness) mustRun(args ...string) {
	h.t.Helper()
	if code := h.run(args...); code != 0 {
		h.t.Fatalf("ca %v: exit %d\nstdout: %s\nstderr: %s", args, code, h.stdout, h.stderr)
	}
}

// seed stores a signed-in profile backed by a live fake grant.
func (h *harness) seed(name string, g *fakeapi.Grant, expiry time.Time) *profile.Profile {
	h.t.Helper()
	at, rt := h.fake.IssueRefresh(g)
	p := &profile.Profile{Endpoint: h.fake.URL, AccountID: "acc-1", AccountName: "Globex",
		Scope: strings.Join(g.Scopes, " "), AccessToken: at, Expiry: expiry, RefreshToken: rt}
	if err := h.store.Save(name, p); err != nil {
		h.t.Fatal(err)
	}
	return p
}

func compute() *fakeapi.Grant {
	return &fakeapi.Grant{Scopes: []string{"read:compute", "write:compute"}}
}

// protected holds every default scope but is not paired.
func protected() *fakeapi.Grant {
	return &fakeapi.Grant{Scopes: append([]string{}, defaultScopes...)}
}

func TestLoginStoresProfileWithPKCEAndAccount(t *testing.T) {
	h := newHarness(t)
	h.mustRun("auth", "login")

	p, err := h.store.Load("default")
	if err != nil {
		t.Fatal(err)
	}
	if p.AccountName != "Globex" || p.AccountID != "acc-1" || p.Endpoint != h.fake.URL {
		t.Errorf("profile = %+v", p)
	}
	if p.RefreshToken == "" || p.AccessToken == "" || time.Until(p.Expiry) < 50*time.Minute {
		t.Errorf("tokens not stored: %+v", p)
	}
	if g := h.fake.GrantOf(p.AccessToken); g == nil {
		t.Error("stored access token is not one the server issued")
	}
	q := h.fake.Authorizations[0]
	if q.Get("scope") != "read:compute write:compute protected:compute read:network read:ssh-keys" || q.Get("state") == "" || q.Get("code_challenge_method") != "S256" {
		t.Errorf("authorize query = %v", q)
	}
	if runtime.GOOS != "windows" {
		st, err := os.Stat(h.store.Dir + "/default.json")
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Perm() != 0o600 {
			t.Errorf("profile mode = %v, want 0600", st.Mode().Perm())
		}
	}
}

// Sign-in requests protected:compute but never pairs; --elevate is gone.
func TestLoginDoesNotPairAndElevateFlagIsGone(t *testing.T) {
	h := newHarness(t)
	if code := h.run("auth", "login", "--elevate"); code != ExitUsage || len(h.fake.Authorizations) != 0 {
		t.Fatalf("--elevate: exit %d, authorizations %d", code, len(h.fake.Authorizations))
	}
	h.mustRun("auth", "login")
	p, _ := h.store.Load("default")
	if g := h.fake.GrantOf(p.AccessToken); g == nil || !g.Has("protected:compute") || g.Elevated {
		t.Errorf("grant = %+v", g)
	}
	if h.fake.ElevationCalls != 0 {
		t.Errorf("login asked for a pairing")
	}
}

func TestLoginRefusesStateMismatch(t *testing.T) {
	h := newHarness(t)
	h.browser = func(u string) error {
		// Consent happens, but the redirect that reaches the loopback carries
		// a state this login did not send.
		c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err := c.Get(u)
		if err != nil {
			return err
		}
		resp.Body.Close()
		loc, _ := url.Parse(resp.Header.Get("Location"))
		q := loc.Query()
		q.Set("state", "forged")
		loc.RawQuery = q.Encode()
		return followBrowser(loc.String())
	}
	if code := h.run("auth", "login"); code != ExitAuth {
		t.Fatalf("exit = %d, want %d; stderr: %s", code, ExitAuth, h.stderr)
	}
	if !strings.Contains(h.stderr.String(), "state") {
		t.Errorf("stderr does not name the state: %s", h.stderr)
	}
	if _, err := h.store.Load("default"); err == nil {
		t.Error("a profile was saved after a state mismatch")
	}
}

func TestLoginIntoExistingProfileRevokesPreviousGrant(t *testing.T) {
	h := newHarness(t)
	h.mustRun("auth", "login")
	first, _ := h.store.Load("default")
	h.mustRun("auth", "login")
	if len(h.fake.Revoked) != 1 || h.fake.Revoked[0] != first.RefreshToken {
		t.Errorf("revoked = %v, want [%s]", h.fake.Revoked, first.RefreshToken)
	}
}

func TestLoginWarnsWhenScopeNotGranted(t *testing.T) {
	h := newHarness(t)
	h.fake.GrantScopes = func([]string) []string { return []string{"read:compute"} }
	h.mustRun("auth", "login")
	if !strings.Contains(h.stderr.String(), "not granted: write:compute") {
		t.Errorf("no warning: %s", h.stderr)
	}
}

func TestRefreshRotatesAndPersistsBeforeUse(t *testing.T) {
	h := newHarness(t)
	old := h.seed("default", compute(), time.Now().Add(30*time.Second)) // inside the 60 s margin
	h.mustRun("vm", "list")
	if h.fake.RefreshCalls != 1 {
		t.Fatalf("refresh calls = %d, want 1", h.fake.RefreshCalls)
	}
	p, _ := h.store.Load("default")
	if p.RefreshToken == old.RefreshToken || p.AccessToken == old.AccessToken {
		t.Fatal("profile still holds the old tokens")
	}
	h.mustRun("vm", "list")
	if h.fake.RefreshCalls != 1 {
		t.Errorf("refreshed again with a fresh token: %d calls", h.fake.RefreshCalls)
	}
}

func TestRefreshInvalidGrantExitsAuth(t *testing.T) {
	h := newHarness(t)
	g := compute()
	h.seed("default", g, time.Now().Add(-time.Minute))
	g.Revoked = true
	if code := h.run("vm", "list"); code != ExitAuth {
		t.Fatalf("exit = %d, want %d", code, ExitAuth)
	}
	if !strings.Contains(h.stderr.String(), "ca auth login") {
		t.Errorf("stderr: %s", h.stderr)
	}
}

func TestNotSignedInExitsAuth(t *testing.T) {
	h := newHarness(t)
	if code := h.run("vm", "list"); code != ExitAuth {
		t.Fatalf("exit = %d", code)
	}
	if code := h.run("vm", "list", "--profile", "work"); code != ExitAuth || !strings.Contains(h.stderr.String(), "--profile work") {
		t.Fatalf("exit = %d, stderr %s", code, h.stderr)
	}
}

func TestProfileSelectionFlagBeatsEnv(t *testing.T) {
	h := newHarness(t)
	h.seed("work", compute(), time.Now().Add(time.Hour))
	h.seed("home", &fakeapi.Grant{Scopes: []string{"read:compute"}}, time.Now().Add(time.Hour))
	h.env["CLOUDAFRICA_PROFILE"] = "home"

	h.mustRun("auth", "status", "-o", "json")
	var st map[string]any
	_ = json.Unmarshal(h.stdout.Bytes(), &st)
	if st["profile"] != "home" {
		t.Errorf("env profile not used: %v", st)
	}
	h.mustRun("auth", "status", "-o", "json", "--profile", "work")
	_ = json.Unmarshal(h.stdout.Bytes(), &st)
	if st["profile"] != "work" {
		t.Errorf("flag did not win: %v", st)
	}

	h.mustRun("auth", "list", "-o", "json")
	var rows []map[string]any
	_ = json.Unmarshal(h.stdout.Bytes(), &rows)
	if len(rows) != 2 || rows[0]["profile"] != "home" || rows[0]["selected"] != true || rows[1]["selected"] != false {
		t.Errorf("list = %v", rows)
	}
}

func TestEnvTokenOverridesProfile(t *testing.T) {
	h := newHarness(t)
	h.fake.VMs = []map[string]any{{"id": "vm-1", "name": "a"}}
	h.env["CLOUDAFRICA_TOKEN"] = "cak_test"
	h.mustRun("vm", "list", "-q")
	if strings.TrimSpace(h.stdout.String()) != "vm-1" {
		t.Errorf("stdout = %q", h.stdout)
	}
	if h.fake.RefreshCalls != 0 {
		t.Error("refreshed with an env token")
	}
}

func TestEndpointDisagreeingWithProfileIsUsageError(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.env["CLOUDAFRICA_ENDPOINT"] = "https://elsewhere.example"
	if code := h.run("vm", "list"); code != ExitUsage {
		t.Fatalf("exit = %d, want %d; %s", code, ExitUsage, h.stderr)
	}
}

func TestVMListFollowsCursor(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.fake.VMs = []map[string]any{{"id": "vm-1"}, {"id": "vm-2"}, {"id": "vm-3"}}
	h.mustRun("vm", "list", "-o", "json")
	var vms []map[string]any
	if err := json.Unmarshal(h.stdout.Bytes(), &vms); err != nil {
		t.Fatal(err, h.stdout.String())
	}
	if len(vms) != 3 {
		t.Errorf("got %d VMs, want 3", len(vms))
	}
}

func TestWritesCarryIdempotencyKey(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.mustRun("vm", "create", "--name", "web", "--datacenter", "jb1", "--image", "img", "--vpc", "vpc-1",
		"--vcpu", "2", "--ram-gib", "4", "--boot-disk-gib", "20", "--accept-terms")
	h.mustRun("vm", "start", "vm-1")
	if len(h.fake.IdemKeys) != 2 || h.fake.IdemKeys[0] == "" || h.fake.IdemKeys[1] == "" || h.fake.IdemKeys[0] == h.fake.IdemKeys[1] {
		t.Errorf("idempotency keys = %q", h.fake.IdemKeys)
	}
}

func TestCreateRequiresTerms(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	code := h.run("vm", "create", "--name", "web", "--datacenter", "jb1", "--image", "img", "--vpc", "vpc-1",
		"--vcpu", "2", "--ram-gib", "4", "--boot-disk-gib", "20")
	if code != ExitUsage || len(h.fake.Writes) != 0 {
		t.Errorf("exit = %d, writes = %d", code, len(h.fake.Writes))
	}
}

func TestWaitPrintsCreatedResource(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.mustRun("vm", "create", "--name", "web", "--datacenter", "jb1", "--image", "img", "--vpc", "vpc-1",
		"--vcpu", "2", "--ram-gib", "4", "--boot-disk-gib", "20", "--accept-terms", "--wait", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "vm-new" {
		t.Errorf("stdout = %q, want the created VM id", got)
	}
}

func TestWaitTimeoutExitsTimeout(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.fake.OpStates["op/start-vm-1"] = []string{"running"}
	if code := h.run("vm", "start", "vm-1", "--wait", "--timeout", "300ms"); code != ExitTimeout {
		t.Fatalf("exit = %d, want %d; %s", code, ExitTimeout, h.stderr)
	}
}

func TestWaitFailedOperationExitsAPI(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.fake.OpStates["op/start-vm-1"] = []string{"failed"}
	if code := h.run("vm", "start", "vm-1", "--wait", "-o", "json"); code != ExitAPI {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(h.stdout.String(), "it broke") || !strings.Contains(h.stderr.String(), "it broke") {
		t.Errorf("stdout %s / stderr %s", h.stdout, h.stderr)
	}
}

func TestOperationGetJoinsSlashedID(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.mustRun("operation", "get", "op/start-vm-1", "-o", "json")
	var op map[string]any
	_ = json.Unmarshal(h.stdout.Bytes(), &op)
	if op["id"] != "op/start-vm-1" {
		t.Errorf("op = %v", op)
	}
}

// Off a terminal each refusal exits 5 and names its own remedy.
func TestStepUpNonInteractiveExits5WithTheMatchingCommand(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.stdin = "y\n" // off a terminal the CLI must not prompt, even with an answer waiting
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp {
		t.Fatalf("exit = %d, want %d", code, ExitStepUp)
	}
	if len(h.fake.Authorizations) != 0 || h.fake.ElevationCalls != 0 || strings.Contains(h.stderr.String(), "[y/N]") {
		t.Errorf("prompted, re-authorized or paired off a terminal: %s", h.stderr)
	}
	if s := h.stderr.String(); !strings.Contains(s, "not granted protected:compute") || !strings.Contains(s, "Run `ca auth login`") {
		t.Errorf("insufficient-scope hint: %s", s)
	}
	h.seed("default", protected(), time.Now().Add(time.Hour))
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp || !strings.Contains(h.stderr.String(), "not-paired") {
		t.Fatalf("not paired: exit = %d; %s", code, h.stderr)
	}
	if s := h.stderr.String(); !strings.Contains(s, "Run `ca auth elevate`") || !strings.Contains(s, "Allow protected actions") ||
		strings.Contains(s, "ca auth login") {
		t.Errorf("not-paired hint: %s", s)
	}
}

// A sign-in without protected:compute is offered `ca auth login`; the new
// sign-in is then refused not-paired and offered `ca auth elevate`.
func TestStepUpInteractiveSignsInThenPairs(t *testing.T) {
	h := newHarness(t)
	old := h.seed("default", compute(), time.Now().Add(time.Hour))
	h.fake.PairOnPoll = 2
	h.tty = true
	h.stdin = "y\ny\n"
	h.mustRun("vm", "delete", "vm-1", "--yes")
	if h.fake.DeleteCalls != 3 || h.fake.ElevationCalls != 2 || len(h.fake.Authorizations) != 1 {
		t.Errorf("delete calls %d (want 3), elevation calls %d (want 2), authorizations %d (want 1)",
			h.fake.DeleteCalls, h.fake.ElevationCalls, len(h.fake.Authorizations))
	}
	s := h.stderr.String()
	if li, ei := strings.Index(s, "`ca auth login`"), strings.Index(s, "`ca auth elevate`"); li < 0 || ei < 0 || li > ei {
		t.Errorf("remedies missing or out of order: %s", s)
	}
	if got := h.fake.Authorizations[0].Get("scope"); !strings.Contains(got, "protected:compute") {
		t.Errorf("re-authorize scope = %q", got)
	}
	if len(h.fake.Revoked) != 1 || h.fake.Revoked[0] != old.RefreshToken {
		t.Errorf("old grant not revoked: %v", h.fake.Revoked)
	}
}

func TestStepUpInteractiveDeclinedKeepsError(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.tty = true
	h.stdin = "n\n"
	if code := h.run("vm", "delete", "vm-1", "--yes"); code != ExitStepUp {
		t.Fatalf("exit = %d", code)
	}
	if h.fake.DeleteCalls != 1 || len(h.fake.Authorizations) != 0 {
		t.Errorf("delete calls %d, authorizations %d", h.fake.DeleteCalls, len(h.fake.Authorizations))
	}
}

func TestDeleteWithoutYesOffTerminalIsUsageError(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	if code := h.run("vm", "delete", "vm-1"); code != ExitUsage || h.fake.DeleteCalls != 0 {
		t.Fatalf("exit = %d, delete calls %d", code, h.fake.DeleteCalls)
	}
}

func TestAPIErrorExits1AndUsageExits2(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	if code := h.run("vm", "get", "missing"); code != ExitAPI {
		t.Errorf("404: exit = %d", code)
	}
	for _, args := range [][]string{{"vm", "get"}, {"bogus"}, {"vm", "bogus"}, {"vm", "list", "--nope"}, {"vm", "list", "-o", "xml"}} {
		if code := h.run(args...); code != ExitUsage {
			t.Errorf("%v: exit = %d, want %d", args, code, ExitUsage)
		}
	}
}

func TestRawAPICall(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.mustRun("api", "GET", "/v1/account")
	if !strings.Contains(h.stdout.String(), "Globex") {
		t.Errorf("stdout = %s", h.stdout)
	}
	h.mustRun("api", "post", "/v1/vms/vm-1/actions/reboot")
	if len(h.fake.IdemKeys) != 1 || h.fake.IdemKeys[0] == "" {
		t.Errorf("raw write had no idempotency key: %q", h.fake.IdemKeys)
	}
}

func TestLogoutRevokesAndDeletes(t *testing.T) {
	h := newHarness(t)
	p := h.seed("default", compute(), time.Now().Add(time.Hour))
	h.mustRun("auth", "logout")
	if len(h.fake.Revoked) != 1 || h.fake.Revoked[0] != p.RefreshToken {
		t.Errorf("revoked = %v", h.fake.Revoked)
	}
	if _, err := h.store.Load("default"); err == nil {
		t.Error("profile still present")
	}
}

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"v0.2.0", "0.1.0", true},
		{"v0.1.0", "0.1.0", false},
		{"v0.10.0", "0.9.0", true},
		{"v0.1.0", "0.1.0-rc.1", true},
		{"v0.1.0-rc.2", "0.1.0", false},
		{"v0.1.0-rc.10", "0.1.0-rc.2", true},
		{"v0.1.0-rc.2", "0.1.0-rc.10", false},
		{"v0.1.0-rc.1.1", "0.1.0-rc.1", true},
		{"garbage", "0.1.0", false},
	} {
		if got := newer(c.a, c.b); got != c.want {
			t.Errorf("newer(%q, %q) = %v", c.a, c.b, got)
		}
	}
}
