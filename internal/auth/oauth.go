// Package auth signs the CLI in with OAuth 2.1 authorization code + PKCE
// against the CloudAfrica authorization server, and refreshes and revokes the
// tokens it gets.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientID is the first-party client the server admits on the REST /v1 routes.
const ClientID = "cloudafrica-cli"

// protectedResourcePath is the RFC 9728 metadata of the resource every
// access token is issued for. The server refuses an authorize request whose
// resource is anything else.
const protectedResourcePath = "/.well-known/oauth-protected-resource/v1/mcp"

const authorizationServerPath = "/.well-known/oauth-authorization-server"

const maxBody = 1 << 20

// Metadata is what the CLI needs from the server's discovery documents.
type Metadata struct {
	AuthorizationEndpoint string
	TokenEndpoint         string
	RevocationEndpoint    string
	Resource              string
}

// Discover reads the authorization-server and protected-resource documents
// from endpoint. Every URL they name must share endpoint's origin: the token
// and revocation endpoints receive refresh tokens, so a document pointing
// elsewhere is refused rather than followed.
func Discover(ctx context.Context, hc *http.Client, endpoint string) (*Metadata, error) {
	if err := RequireTLS(endpoint); err != nil {
		return nil, err
	}
	origin, err := Origin(endpoint)
	if err != nil {
		return nil, err
	}
	var as struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		RevocationEndpoint    string `json:"revocation_endpoint"`
	}
	if err := getJSON(ctx, hc, origin+authorizationServerPath, &as); err != nil {
		return nil, err
	}
	var pr struct {
		Resource string `json:"resource"`
	}
	if err := getJSON(ctx, hc, origin+protectedResourcePath, &pr); err != nil {
		return nil, err
	}
	m := &Metadata{
		AuthorizationEndpoint: as.AuthorizationEndpoint,
		TokenEndpoint:         as.TokenEndpoint,
		RevocationEndpoint:    as.RevocationEndpoint,
		Resource:              pr.Resource,
	}
	for name, u := range map[string]string{
		"authorization_endpoint": m.AuthorizationEndpoint,
		"token_endpoint":         m.TokenEndpoint,
		"revocation_endpoint":    m.RevocationEndpoint,
	} {
		o, err := Origin(u)
		if err != nil {
			return nil, fmt.Errorf("discovery: %s %q: %w", name, u, err)
		}
		if o != origin {
			return nil, fmt.Errorf("discovery: %s %q is not on %s; refusing to send tokens there", name, u, origin)
		}
	}
	if m.Resource == "" {
		return nil, fmt.Errorf("discovery: %s names no resource", origin+protectedResourcePath)
	}
	return m, nil
}

// RequireTLS refuses a plain-http endpoint unless its host is loopback:
// credentials sent to it would cross the network in cleartext.
func RequireTLS(endpoint string) error {
	o, err := Origin(endpoint)
	if err != nil {
		return err
	}
	u, _ := url.Parse(o)
	if u.Scheme == "https" {
		return nil
	}
	h := u.Hostname()
	if ip := net.ParseIP(h); h == "localhost" || (ip != nil && ip.IsLoopback()) {
		return nil
	}
	return fmt.Errorf("%s is plain http; credentials are only sent over https (or http to a loopback address)", o)
}

// Origin is scheme://host of u, lowercased, with the scheme's default port dropped.
func Origin(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	scheme, host := strings.ToLower(u.Scheme), strings.ToLower(u.Host)
	if (scheme != "https" && scheme != "http") || host == "" {
		return "", fmt.Errorf("%q is not an http(s) URL with a host", raw)
	}
	if (scheme == "https" && strings.HasSuffix(host, ":443")) || (scheme == "http" && strings.HasSuffix(host, ":80")) {
		host = host[:strings.LastIndex(host, ":")]
	}
	return scheme + "://" + host, nil
}

func getJSON(ctx context.Context, hc *http.Client, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return fmt.Errorf("discovery: reading %s: %w", u, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discovery: GET %s: HTTP %d", u, resp.StatusCode)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("discovery: decoding %s: %w", u, err)
	}
	return nil
}

// Token is a token endpoint response (RFC 6749 §5.1).
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// Expiry is when the access token expires, counted from now.
func (t *Token) Expiry(now time.Time) time.Time {
	return now.Add(time.Duration(t.ExpiresIn) * time.Second)
}

// OAuthError is an RFC 6749 §5.2 error response.
type OAuthError struct {
	Status      int
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e *OAuthError) Error() string {
	s := fmt.Sprintf("oauth: HTTP %d", e.Status)
	if e.Code != "" {
		s += " " + e.Code
	}
	if e.Description != "" {
		s += ": " + e.Description
	}
	return s
}

// IsInvalidGrant reports whether err is the server refusing a code or refresh token.
func IsInvalidGrant(err error) bool {
	var e *OAuthError
	return errors.As(err, &e) && e.Code == "invalid_grant"
}

func postForm(ctx context.Context, hc *http.Client, u string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		oe := &OAuthError{Status: resp.StatusCode}
		_ = json.Unmarshal(data, oe) // a non-JSON body still yields the status
		return oe
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("oauth: decoding %s response: %w", u, err)
	}
	return nil
}

func exchange(ctx context.Context, hc *http.Client, m *Metadata, code, redirectURI, verifier string) (*Token, error) {
	var t Token
	err := postForm(ctx, hc, m.TokenEndpoint, url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {ClientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
		"resource":      {m.Resource},
	}, &t)
	if err != nil {
		return nil, err
	}
	if t.AccessToken == "" {
		return nil, errors.New("oauth: token response carries no access_token")
	}
	return &t, nil
}

// Refresh trades a refresh token for a new pair. The old refresh token is
// spent whatever happens next, so the caller must persist the new one before
// using the access token.
func Refresh(ctx context.Context, hc *http.Client, m *Metadata, refreshToken string) (*Token, error) {
	var t Token
	err := postForm(ctx, hc, m.TokenEndpoint, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {ClientID},
		"refresh_token": {refreshToken},
	}, &t)
	if err != nil {
		return nil, err
	}
	if t.AccessToken == "" || t.RefreshToken == "" {
		return nil, errors.New("oauth: refresh response lacks a token")
	}
	return &t, nil
}

// Revoke revokes the grant a refresh token belongs to (RFC 7009).
func Revoke(ctx context.Context, hc *http.Client, m *Metadata, refreshToken string) error {
	return postForm(ctx, hc, m.RevocationEndpoint, url.Values{
		"token":     {refreshToken},
		"client_id": {ClientID},
	}, nil)
}

// HTTPClient is the client for the OAuth endpoints. It follows no redirects:
// a 307 would re-send a refresh token in the body to wherever it points.
func HTTPClient() *http.Client {
	return &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}
