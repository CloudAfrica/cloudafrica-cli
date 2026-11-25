package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LoginOptions configures one browser sign-in.
type LoginOptions struct {
	Scopes []string
	// Account, when set, is the account the sign-in is for: it is sent as the
	// authorize request's account parameter, so the consent page binds the
	// token to it instead of asking.
	Account string
	// Open opens url in the user's browser; an error means the URL is only printed.
	Open func(url string) error
	// Out receives the instructions (the URL to visit).
	Out io.Writer
	// Timeout bounds the wait for the browser to come back (default 5 minutes).
	Timeout time.Duration
}

// Login runs the authorization code + PKCE flow with an RFC 8252 loopback
// redirect: it listens on 127.0.0.1 on a free port, sends the user to the
// authorize URL, and exchanges the code that comes back for tokens.
func Login(ctx context.Context, hc *http.Client, m *Metadata, o LoginOptions) (*Token, error) {
	verifier, err := randomString(32)
	if err != nil {
		return nil, err
	}
	state, err := randomString(16)
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listening for the browser redirect: %w", err)
	}
	defer ln.Close()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("listening for the browser redirect: unexpected address %v", ln.Addr())
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", addr.Port)

	sum := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"client_id":             {ClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
		"resource":              {m.Resource},
		"state":                 {state},
		"scope":                 {strings.Join(o.Scopes, " ")},
	}
	if o.Account != "" {
		q.Set("account", o.Account)
	}
	authURL := m.AuthorizationEndpoint
	if strings.Contains(authURL, "?") {
		authURL += "&" + q.Encode()
	} else {
		authURL += "?" + q.Encode()
	}

	done := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		res := callback(r.URL.Query(), state)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if res.err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, page, "Sign-in failed", html.EscapeString(res.err.Error()))
		} else {
			fmt.Fprintf(w, page, "Signed in", "You can close this tab and return to the terminal.")
		}
		select {
		case done <- res:
		default:
		}
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer func() {
		sctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	if o.Out != nil {
		fmt.Fprintf(o.Out, "Opening your browser to sign in. If it does not open, visit:\n\n  %s\n\n", authURL)
	}
	if o.Open != nil {
		if err := o.Open(authURL); err != nil && o.Out != nil {
			fmt.Fprintf(o.Out, "(could not open a browser: %v)\n", err)
		}
	}

	timeout := o.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	wctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var res callbackResult
	select {
	case res = <-done:
	case <-wctx.Done():
		return nil, fmt.Errorf("waiting for the browser sign-in: %w", wctx.Err())
	}
	if res.err != nil {
		return nil, res.err
	}
	return exchange(ctx, hc, m, res.code, redirectURI, verifier)
}

type callbackResult struct {
	code string
	err  error
}

// callback validates the redirect's query: state first (a mismatch is a
// request this login did not start, whatever else it carries), then an
// error, then the code.
func callback(q url.Values, state string) callbackResult {
	if q.Get("state") != state {
		return callbackResult{err: errors.New("sign-in failed: the browser returned a state this login did not send")}
	}
	if e := q.Get("error"); e != "" {
		msg := "sign-in failed: " + e
		if d := q.Get("error_description"); d != "" {
			msg += ": " + d
		}
		return callbackResult{err: errors.New(msg)}
	}
	code := q.Get("code")
	if code == "" {
		return callbackResult{err: errors.New("sign-in failed: the browser returned no code")}
	}
	return callbackResult{code: code}
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

const page = `<!doctype html><meta charset="utf-8"><title>CloudAfrica CLI</title>
<body style="font-family:system-ui,sans-serif;max-width:32rem;margin:4rem auto;padding:0 1rem">
<h1>%s</h1><p>%s</p></body>`
