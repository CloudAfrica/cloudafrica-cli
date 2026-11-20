package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
	"github.com/CloudAfrica/cloudafrica-cli/internal/profile"
)

func TestDiscoverRefusesForeignTokenEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case authorizationServerPath:
			_, _ = w.Write([]byte(`{"authorization_endpoint":"http://` + r.Host + `/oauth/authorize",
				"token_endpoint":"https://evil.example/oauth/token","revocation_endpoint":"http://` + r.Host + `/oauth/revoke"}`))
		case protectedResourcePath:
			_, _ = w.Write([]byte(`{"resource":"http://` + r.Host + `/v1/mcp"}`))
		}
	}))
	defer srv.Close()
	_, err := Discover(context.Background(), HTTPClient(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "token_endpoint") {
		t.Fatalf("err = %v, want a refusal naming token_endpoint", err)
	}
}

func TestCallbackChecksStateFirst(t *testing.T) {
	q := url.Values{"state": {"other"}, "code": {"c"}}
	if r := callback(q, "mine"); r.err == nil || r.code != "" {
		t.Errorf("mismatched state accepted: %+v", r)
	}
	q = url.Values{"state": {"mine"}, "error": {"access_denied"}}
	if r := callback(q, "mine"); r.err == nil || !strings.Contains(r.err.Error(), "access_denied") {
		t.Errorf("error not surfaced: %+v", r)
	}
	q = url.Values{"state": {"mine"}, "code": {"c"}}
	if r := callback(q, "mine"); r.err != nil || r.code != "c" {
		t.Errorf("good callback refused: %+v", r)
	}
}

// Two processes holding the same expired profile refresh it once: the second
// waits on the profile lock, re-reads the file and uses the first's tokens.
// Without the lock both present the same refresh token and the second is
// refused (outside the grace window, the server revokes the grant).
func TestConcurrentSourcesRefreshOnce(t *testing.T) {
	fake := fakeapi.New(t)
	store := profile.Store{Dir: t.TempDir()}
	at, rt := fake.IssueRefresh(&fakeapi.Grant{Scopes: []string{"read:compute"}})
	if err := store.Save("default", &profile.Profile{Endpoint: fake.URL, AccessToken: at,
		Expiry: time.Now().Add(-time.Minute), RefreshToken: rt}); err != nil {
		t.Fatal(err)
	}
	const n = 4
	tokens := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		src := &Source{Store: store, Name: "default", HTTP: HTTPClient()}
		wg.Add(1)
		go func() {
			defer wg.Done()
			tokens[i], errs[i] = src.Token(context.Background())
		}()
	}
	wg.Wait()
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("source %d: %v", i, errs[i])
		}
		if tokens[i] != tokens[0] || tokens[i] == at {
			t.Errorf("source %d got %q, want the one refreshed token (first %q, old %q)", i, tokens[i], tokens[0], at)
		}
	}
	fake.Mu.Lock()
	defer fake.Mu.Unlock()
	if fake.RefreshCalls != 1 {
		t.Errorf("refresh calls = %d, want 1", fake.RefreshCalls)
	}
}

func discoveryServer(t *testing.T, doc func(host string) (as, pr string)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		as, pr := doc(r.Host)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case authorizationServerPath:
			_, _ = w.Write([]byte(as))
		case protectedResourcePath:
			_, _ = w.Write([]byte(pr))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDiscoverChecksEveryEndpointAndTheResource(t *testing.T) {
	for _, c := range []struct {
		name              string
		authz, token, rev string
		resource, wantErr string
	}{
		{"foreign authorize", "https://evil.example/a", "", "", "/v1/mcp", "authorization_endpoint"},
		{"foreign revoke", "", "", "https://evil.example/r", "/v1/mcp", "revocation_endpoint"},
		{"other scheme", "", "https://HOST/oauth/token", "", "/v1/mcp", "token_endpoint"},
		{"other port", "", "http://127.0.0.1:1/oauth/token", "", "/v1/mcp", "token_endpoint"},
		{"no resource", "", "", "", "", "resource"},
	} {
		srv := discoveryServer(t, func(host string) (string, string) {
			or := func(v, path string) string {
				if v == "" {
					return "http://" + host + path
				}
				return strings.ReplaceAll(v, "HOST", host)
			}
			res := ""
			if c.resource != "" {
				res = "http://" + host + c.resource
			}
			return `{"authorization_endpoint":"` + or(c.authz, "/oauth/authorize") + `","token_endpoint":"` +
					or(c.token, "/oauth/token") + `","revocation_endpoint":"` + or(c.rev, "/oauth/revoke") + `"}`,
				`{"resource":"` + res + `"}`
		})
		_, err := Discover(context.Background(), HTTPClient(), srv.URL)
		if err == nil || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: err = %v, want one naming %s", c.name, err, c.wantErr)
		}
	}
}

func TestOrigin(t *testing.T) {
	for raw, want := range map[string]string{
		"https://API.CloudAfrica.net:443/v1":         "https://api.cloudafrica.net",
		"http://h:80":                                "http://h",
		"https://h:8443/x":                           "https://h:8443",
		"https://api.cloudafrica.net@evil.example/x": "https://evil.example",
		"ftp://h":   "",
		"/relative": "",
	} {
		got, err := Origin(raw)
		if got != want || (err == nil) != (want != "") {
			t.Errorf("Origin(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
}

// expired seeds an expired profile on the fake and returns its Source.
func expired(t *testing.T, fake *fakeapi.Server, refresh bool) (*Source, profile.Store) {
	t.Helper()
	store := profile.Store{Dir: t.TempDir()}
	at, rt := fake.IssueRefresh(&fakeapi.Grant{Scopes: []string{"read:compute"}})
	if !refresh {
		rt = ""
	}
	if err := store.Save("default", &profile.Profile{Endpoint: fake.URL, AccessToken: at,
		Expiry: time.Now().Add(-time.Minute), RefreshToken: rt}); err != nil {
		t.Fatal(err)
	}
	return &Source{Store: store, Name: "default", HTTP: HTTPClient()}, store
}

func TestSourceExpiredWithoutRefreshTokenNeedsLogin(t *testing.T) {
	src, _ := expired(t, fakeapi.New(t), false)
	if _, err := src.Token(context.Background()); !errors.Is(err, ErrLoginRequired) {
		t.Errorf("err = %v", err)
	}
}

func TestSourceSignedOutWhileWaitingNeedsLogin(t *testing.T) {
	src, store := expired(t, fakeapi.New(t), true)
	if _, err := src.Profile(); err != nil { // cached before the logout
		t.Fatal(err)
	}
	if err := store.Delete("default"); err != nil {
		t.Fatal(err)
	}
	_, err := src.Token(context.Background())
	if !errors.Is(err, ErrLoginRequired) || !strings.Contains(err.Error(), "signed out") {
		t.Errorf("err = %v", err)
	}
}

// A refresh the server fails (5xx) is not a signed-out session: retrying
// later may work, so it must not read as "run ca auth login".
func TestSourceServerErrorIsNotLoginRequired(t *testing.T) {
	fake := fakeapi.New(t)
	fake.RefreshStatus = http.StatusServiceUnavailable
	src, _ := expired(t, fake, true)
	_, err := src.Token(context.Background())
	if err == nil || errors.Is(err, ErrLoginRequired) {
		t.Errorf("err = %v", err)
	}
}

func TestSourceUnsavableRefreshNeedsLogin(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a directory the process cannot write")
	}
	src, store := expired(t, fakeapi.New(t), true)
	release, err := store.Lock(context.Background(), "default") // create the lock file first
	if err != nil {
		t.Fatal(err)
	}
	release()
	if err := os.Chmod(store.Dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(store.Dir, 0o700) })
	_, err = src.Token(context.Background())
	if !errors.Is(err, ErrLoginRequired) || !strings.Contains(err.Error(), "could not be saved") {
		t.Errorf("err = %v", err)
	}
}

// The token endpoint's redirect is not followed: a 307 would re-send the
// refresh token to wherever it points.
func TestRefreshDoesNotFollowRedirects(t *testing.T) {
	var hits atomic.Int32
	elsewhere := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	defer elsewhere.Close()
	fake := fakeapi.New(t)
	fake.TokenRedirect = elsewhere.URL + "/steal"
	src, _ := expired(t, fake, true)
	if _, err := src.Token(context.Background()); err == nil {
		t.Error("refresh succeeded through a redirect")
	}
	if hits.Load() != 0 {
		t.Errorf("redirect target received %d requests", hits.Load())
	}
}

func TestRequireTLS(t *testing.T) {
	for ep, ok := range map[string]bool{
		"https://api.cloudafrica.net": true,
		"http://127.0.0.1:8080":       true,
		"http://localhost:8080":       true,
		"http://[::1]:8080":           true,
		"http://api.example":          false,
		"http://10.0.0.5":             false,
	} {
		if err := RequireTLS(ep); (err == nil) != ok {
			t.Errorf("RequireTLS(%q) = %v", ep, err)
		}
	}
	if _, err := Discover(context.Background(), HTTPClient(), "http://api.example"); err == nil {
		t.Error("Discover accepted a plain-http non-loopback endpoint")
	}
}

// A profile signed in to another endpoint since the session began is refused,
// not refreshed against this source's endpoint.
func TestSourceRefusesProfileMovedToAnotherEndpoint(t *testing.T) {
	src, store := expired(t, fakeapi.New(t), true)
	before, _ := store.Load("default")
	src.Endpoint = "https://other.example"
	_, err := src.Token(context.Background())
	if !errors.Is(err, ErrLoginRequired) || !strings.Contains(err.Error(), "other.example") {
		t.Errorf("err = %v", err)
	}
	if after, _ := store.Load("default"); after.RefreshToken != before.RefreshToken {
		t.Error("refresh token was spent")
	}
}

// Without a refresh token the access token is used until it actually expires.
func TestSourceUsesUnrefreshableTokenUntilExpiry(t *testing.T) {
	store := profile.Store{Dir: t.TempDir()}
	if err := store.Save("default", &profile.Profile{Endpoint: "https://api.cloudafrica.net", AccessToken: "at",
		Expiry: time.Now().Add(30 * time.Second)}); err != nil {
		t.Fatal(err)
	}
	src := &Source{Store: store, Name: "default", HTTP: HTTPClient()}
	if tok, err := src.Token(context.Background()); err != nil || tok != "at" {
		t.Errorf("token %q, err %v", tok, err)
	}
}
