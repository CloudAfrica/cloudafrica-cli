package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
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
