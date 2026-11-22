// Package fakeapi is an in-memory CloudAfrica API and authorization server
// for the CLI's tests: discovery, authorize/token/revoke with PKCE and
// refresh rotation, and enough of /v1 to drive the VM commands.
package fakeapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// Grant is one authorize → token chain.
type Grant struct {
	Scopes   []string
	Elevated bool
	Revoked  bool
}

func (g *Grant) Has(scope string) bool {
	for _, s := range g.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

type pendingCode struct {
	challenge, redirectURI, resource string
	grant                            *Grant
}

// Server is the fake. Lock Mu before reading or changing exported state
// while a command runs.
type Server struct {
	*httptest.Server
	T  testing.TB
	Mu sync.Mutex

	// GrantScopes, when set, overrides the scopes the consent "grants".
	GrantScopes func(requested []string) []string
	// PairOnPoll, when > 0, confirms a grant's pairing on its PairOnPoll-th
	// elevation request, as the user confirming on /pair would.
	PairOnPoll int
	// PairURL, when set, replaces the confirm page an elevation request returns.
	PairURL string
	// NoPairCode makes an elevation request answer a page with no code.
	NoPairCode bool
	// AlwaysNotPaired makes VM delete answer not-paired even once paired.
	AlwaysNotPaired bool
	// ExpiresIn is the access-token lifetime in seconds (default 3600).
	ExpiresIn int64
	// Failure switches.
	DenyConsent        bool   // authorize redirects with error=access_denied
	RevokeFails        bool   // revoke answers 503
	RefreshStatus      int    // when set, refresh answers this status
	TokenRedirect      string // when set, the token endpoint 307s here
	OmitOperationID    bool   // 202s carry only the Location header
	EmptyOperationBody bool   // GET /v1/operations/* answers 200 with no body

	codes map[string]*pendingCode
	// tokenPairs holds paired (API token, elevation secret) pairs; a token
	// pairs by the secret it sends, never by itself.
	tokenPairs map[string]bool
	access     map[string]*Grant
	refresh    map[string]*Grant // live refresh tokens
	spent      map[string]bool   // used refresh tokens
	seq        int
	VMs        []map[string]any
	OpStates   map[string][]string // operation id → states served in turn

	// Observations.
	RefreshCalls   int
	Revoked        []string
	Authorizations []url.Values
	Writes         []*http.Request
	WriteURLs      []string         // method, path and query of each /v1 write
	Bodies         []map[string]any // decoded JSON body of each /v1 write (nil if none)
	IdemKeys       []string
	DeleteCalls    int
	ElevationCalls int
	ElevationBody  []map[string]any // decoded body of each elevation request
	// ElevationHeaders is "METHOD path header" for every /v1 request
	// (header "" when absent).
	ElevationHeaders []string
	SSHKeys          []map[string]any
	VPCs             []map[string]any
	Datacenters      []map[string]any
	// Clusters are the Kubernetes clusters; a "nodes" entry is served only
	// by get, as the detail's nodes.
	Clusters []map[string]any
	// UpgradeConflict, when set, is the detail of a 409 to upgrade-workers.
	UpgradeConflict string
	// UpgradeNoOperation makes upgrade-workers answer 202 with only the
	// cluster's Location, as when no operation started.
	UpgradeNoOperation bool
}

// New starts a fake server.
func New(t testing.TB) *Server {
	s := &Server{
		T:          t,
		codes:      map[string]*pendingCode{},
		tokenPairs: map[string]bool{},
		access:     map[string]*Grant{},
		refresh:    map[string]*Grant{},
		spent:      map[string]bool{},
		OpStates:   map[string][]string{},
	}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Close)
	return s
}

func (s *Server) next(prefix string) string {
	s.seq++
	return fmt.Sprintf("%s%d", prefix, s.seq)
}

// IssueRefresh registers a live refresh token for a grant (to seed a profile).
func (s *Server) IssueRefresh(g *Grant) (access, refresh string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	access, refresh = s.next("at_"), s.next("rt_")
	s.access[access] = g
	s.refresh[refresh] = g
	return
}

// GrantOf returns the grant an access token belongs to.
func (s *Server) GrantOf(access string) *Grant {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.access[access]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func problem(w http.ResponseWriter, status int, slug, title string, extra map[string]any) {
	body := map[string]any{"type": "https://docs.cloudafrica.net/errors/" + slug, "title": title, "status": status}
	for k, v := range extra {
		body[k] = v
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// insufficientScope mirrors the gateway: the scope is named only in detail.
func insufficientScope(w http.ResponseWriter, scope string) {
	problem(w, 403, "insufficient-scope", "Insufficient scope",
		map[string]any{"detail": "This token is missing the required scope: " + scope})
}

func oauthErr(w http.ResponseWriter, code, desc string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": code, "error_description": desc})
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	p := r.URL.Path
	switch {
	case p == "/.well-known/oauth-authorization-server":
		writeJSON(w, 200, map[string]any{
			"issuer":                 s.URL,
			"authorization_endpoint": s.URL + "/oauth/authorize",
			"token_endpoint":         s.URL + "/oauth/token",
			"revocation_endpoint":    s.URL + "/oauth/revoke",
		})
	case p == "/.well-known/oauth-protected-resource/v1/mcp":
		writeJSON(w, 200, map[string]any{"resource": s.URL + "/v1/mcp"})
	case p == "/oauth/authorize":
		s.authorize(w, r)
	case p == "/oauth/token":
		s.token(w, r)
	case p == "/oauth/revoke":
		if s.RevokeFails {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		_ = r.ParseForm()
		s.Revoked = append(s.Revoked, r.PostForm.Get("token"))
		if g := s.refresh[r.PostForm.Get("token")]; g != nil {
			g.Revoked = true
		}
		w.WriteHeader(200)
	case strings.HasPrefix(p, "/v1/"):
		s.api(w, r)
	default:
		http.NotFound(w, r)
	}
}

// authorize plays the consent page: it "approves" at once and redirects to
// the loopback with a code, as a browser would after the user consents.
func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s.Authorizations = append(s.Authorizations, q)
	redirect := q.Get("redirect_uri")
	if q.Get("client_id") != "cloudafrica-cli" || q.Get("response_type") != "code" ||
		q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" ||
		!strings.HasPrefix(redirect, "http://127.0.0.1:") || q.Get("resource") != s.URL+"/v1/mcp" {
		http.Error(w, "bad authorize request", 400)
		return
	}
	if s.DenyConsent {
		http.Redirect(w, r, redirect+"?error=access_denied&state="+url.QueryEscape(q.Get("state")), http.StatusFound)
		return
	}
	requested := strings.Fields(q.Get("scope"))
	granted := requested
	if s.GrantScopes != nil {
		granted = s.GrantScopes(requested)
	}
	// Consent grants scopes; it never pairs (elevation is its own step).
	g := &Grant{Scopes: granted}
	code := s.next("code_")
	s.codes[code] = &pendingCode{challenge: q.Get("code_challenge"), redirectURI: redirect, resource: q.Get("resource"), grant: g}
	http.Redirect(w, r, redirect+"?code="+url.QueryEscape(code)+"&state="+url.QueryEscape(q.Get("state")), http.StatusFound)
}

func (s *Server) issue(w http.ResponseWriter, g *Grant) {
	at, rt := s.next("at_"), s.next("rt_")
	s.access[at] = g
	s.refresh[rt] = g
	exp := s.ExpiresIn
	if exp == 0 {
		exp = 3600
	}
	writeJSON(w, 200, map[string]any{
		"access_token": at, "token_type": "Bearer", "expires_in": exp,
		"refresh_token": rt, "scope": strings.Join(g.Scopes, " "),
	})
}

func (s *Server) token(w http.ResponseWriter, r *http.Request) {
	if s.TokenRedirect != "" {
		http.Redirect(w, r, s.TokenRedirect, http.StatusTemporaryRedirect)
		return
	}
	_ = r.ParseForm()
	f := r.PostForm
	if f.Get("client_id") != "cloudafrica-cli" {
		oauthErr(w, "invalid_client", "unknown client")
		return
	}
	switch f.Get("grant_type") {
	case "authorization_code":
		pc := s.codes[f.Get("code")]
		delete(s.codes, f.Get("code"))
		if pc == nil {
			oauthErr(w, "invalid_grant", "invalid authorization code")
			return
		}
		sum := sha256.Sum256([]byte(f.Get("code_verifier")))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != pc.challenge {
			oauthErr(w, "invalid_grant", "PKCE verification failed")
			return
		}
		if f.Get("redirect_uri") != pc.redirectURI || f.Get("resource") != pc.resource {
			oauthErr(w, "invalid_grant", "redirect_uri or resource mismatch")
			return
		}
		s.issue(w, pc.grant)
	case "refresh_token":
		s.RefreshCalls++
		if s.RefreshStatus != 0 {
			http.Error(w, "refresh failed", s.RefreshStatus)
			return
		}
		rt := f.Get("refresh_token")
		if s.spent[rt] {
			oauthErr(w, "invalid_grant", "refresh token already used")
			return
		}
		g := s.refresh[rt]
		if g == nil || g.Revoked {
			oauthErr(w, "invalid_grant", "invalid refresh token")
			return
		}
		delete(s.refresh, rt)
		s.spent[rt] = true
		s.issue(w, g)
	default:
		oauthErr(w, "unsupported_grant_type", f.Get("grant_type"))
	}
}

func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	g := s.access[tok]
	if strings.HasPrefix(tok, "cak_") {
		g = &Grant{Scopes: []string{"read:compute", "write:compute"}}
		if strings.HasPrefix(tok, "cak_protected") { // holds the scope; paired only by a confirmed secret
			g.Scopes = append(g.Scopes, "protected:compute")
		}
		g.Elevated = s.tokenPairs[tok+"|"+r.Header.Get("X-CloudAfrica-Elevation")]
	}
	s.ElevationHeaders = append(s.ElevationHeaders, r.Method+" "+r.URL.Path+" "+r.Header.Get("X-CloudAfrica-Elevation"))
	if g == nil || g.Revoked {
		problem(w, 401, "invalid-token", "Invalid token", nil)
		return
	}
	p := r.URL.Path
	if r.Method != http.MethodGet {
		raw, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(raw))
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		s.Writes = append(s.Writes, r)
		s.WriteURLs = append(s.WriteURLs, r.Method+" "+r.URL.RequestURI())
		s.Bodies = append(s.Bodies, body)
		s.IdemKeys = append(s.IdemKeys, r.Header.Get("Idempotency-Key"))
	}
	need := "read:compute"
	if r.Method != http.MethodGet {
		need = "write:compute"
	}
	switch p {
	case "/v1/ssh-keys":
		need = "read:ssh-keys"
	case "/v1/vpcs":
		need = "read:network"
	}
	if strings.HasPrefix(p, "/v1/vpcs/") && r.Method == http.MethodDelete {
		need = "protected:network"
	}
	if p != "/v1/account" && p != "/v1/elevation-requests" && !g.Has(need) {
		insufficientScope(w, need)
		return
	}
	switch {
	case p == "/v1/elevation-requests" && r.Method == http.MethodPost:
		s.elevationRequest(w, r, tok, g)
	case p == "/v1/ssh-keys" && r.Method == http.MethodGet:
		page(w, r, s.SSHKeys)
	case p == "/v1/vpcs" && r.Method == http.MethodGet:
		page(w, r, s.VPCs)
	case p == "/v1/placement-groups/datacenters" && r.Method == http.MethodGet:
		writeJSON(w, 200, map[string]any{"data": orEmpty(s.Datacenters)})
	case p == "/v1/account":
		writeJSON(w, 200, map[string]any{"accountId": "acc-1", "accountName": "Globex"})
	case p == "/v1/vms" && r.Method == http.MethodGet:
		s.listVMs(w, r)
	case p == "/v1/vms" && r.Method == http.MethodPost:
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.accept(w, "op/create-"+fmt.Sprint(body["name"]))
	case strings.HasPrefix(p, "/v1/vms/") && strings.Contains(p, "/actions/"):
		parts := strings.Split(strings.TrimPrefix(p, "/v1/vms/"), "/actions/")
		s.accept(w, "op/"+parts[1]+"-"+parts[0])
	case strings.HasPrefix(p, "/v1/vms/") && (r.Method == http.MethodPatch || r.Method == http.MethodPut):
		s.accept(w, "op/update-"+strings.TrimPrefix(p, "/v1/vms/"))
	case strings.HasPrefix(p, "/v1/vms/") && r.Method == http.MethodDelete:
		s.DeleteCalls++
		switch {
		case !g.Has("protected:compute"):
			insufficientScope(w, "protected:compute")
		case !g.Elevated || s.AlwaysNotPaired:
			problem(w, 403, "not-paired", "Client not paired", nil)
		case s.vmStatus(strings.TrimPrefix(p, "/v1/vms/")) == "running":
			problem(w, 409, "conflict", "Conflict", map[string]any{"detail": "The VM must be stopped first."})
		default:
			s.accept(w, "op/delete-"+strings.TrimPrefix(p, "/v1/vms/"))
		}
	case strings.HasPrefix(p, "/v1/vms/"):
		id := strings.TrimPrefix(p, "/v1/vms/")
		for _, vm := range s.VMs {
			if vm["id"] == id {
				writeJSON(w, 200, vm)
				return
			}
		}
		problem(w, 404, "not-found", "Not found", nil)
	case p == "/v1/kubernetes/clusters" && r.Method == http.MethodGet:
		rows := []map[string]any{}
		for _, c := range s.Clusters {
			row := map[string]any{}
			for k, v := range c {
				if k != "nodes" {
					row[k] = v
				}
			}
			rows = append(rows, row)
		}
		page(w, r, rows)
	case strings.HasPrefix(p, "/v1/kubernetes/clusters/") && strings.HasSuffix(p, "/actions/upgrade-workers"):
		id := strings.TrimSuffix(strings.TrimPrefix(p, "/v1/kubernetes/clusters/"), "/actions/upgrade-workers")
		switch {
		case !g.Has("protected:compute"):
			insufficientScope(w, "protected:compute")
		case !g.Elevated:
			problem(w, 403, "not-paired", "Client not paired", nil)
		case s.UpgradeConflict != "":
			problem(w, 409, "conflict", "Conflict", map[string]any{"detail": s.UpgradeConflict})
		case s.UpgradeNoOperation:
			w.Header().Set("Location", "/v1/kubernetes/clusters/"+id)
			w.WriteHeader(http.StatusAccepted)
		default:
			s.accept(w, "op/upgrade-"+id)
		}
	case strings.HasPrefix(p, "/v1/kubernetes/clusters/") && r.Method == http.MethodGet:
		id := strings.TrimPrefix(p, "/v1/kubernetes/clusters/")
		for _, c := range s.Clusters {
			if c["id"] == id {
				cluster := map[string]any{}
				for k, v := range c {
					if k != "nodes" {
						cluster[k] = v
					}
				}
				writeJSON(w, 200, map[string]any{"cluster": cluster, "nodes": c["nodes"]})
				return
			}
		}
		problem(w, 404, "not-found", "Not found", map[string]any{"detail": "No such Kubernetes cluster in this account."})
	case strings.HasPrefix(p, "/v1/operations/"):
		id := strings.TrimPrefix(p, "/v1/operations/")
		if s.EmptyOperationBody {
			w.WriteHeader(http.StatusOK)
			return
		}
		states := s.OpStates[id]
		state := "complete"
		if len(states) > 0 {
			state = states[0]
			if len(states) > 1 {
				s.OpStates[id] = states[1:]
			}
		}
		op := map[string]any{"id": id, "kind": "vm.action", "state": state}
		if strings.HasPrefix(id, "op/create-") && state == "complete" {
			op["createdResource"] = map[string]any{"id": "vm-new"}
		}
		if state == "failed" {
			op["error"] = map[string]any{"kind": "fault", "message": "it broke"}
		}
		writeJSON(w, 200, op)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) accept(w http.ResponseWriter, opID string) {
	w.Header().Set("Location", "/v1/operations/"+opID)
	if s.OmitOperationID {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, 202, map[string]any{"operationId": opID})
}

// listVMs pages two VMs at a time.
func (s *Server) listVMs(w http.ResponseWriter, r *http.Request) { page(w, r, s.VMs) }

// page serves rows two at a time, following ?cursor=.
func page(w http.ResponseWriter, r *http.Request, rows []map[string]any) {
	start := 0
	if c := r.URL.Query().Get("cursor"); c != "" {
		_, _ = fmt.Sscanf(c, "c%d", &start)
	}
	start = min(start, len(rows))
	end := min(start+2, len(rows))
	body := map[string]any{"data": orEmpty(rows[start:end])}
	if end < len(rows) {
		body["nextCursor"] = fmt.Sprintf("c%d", end)
	}
	writeJSON(w, 200, body)
}

func orEmpty(rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	return rows
}

func (s *Server) vmStatus(id string) string {
	for _, vm := range s.VMs {
		if vm["id"] == id {
			status, _ := vm["status"].(string)
			return status
		}
	}
	return ""
}

// elevationRequest mirrors POST /v1/elevation-requests: an OAuth grant pairs
// with no header; a user API token (cak_) must send its elevation header.
// The page and code stay the same while the request is pending.
func (s *Server) elevationRequest(w http.ResponseWriter, r *http.Request, tok string, g *Grant) {
	s.ElevationCalls++
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	s.ElevationBody = append(s.ElevationBody, body)
	secret := r.Header.Get("X-CloudAfrica-Elevation")
	if strings.HasPrefix(tok, "cak_") && !wellFormed(secret) {
		problem(w, 400, "validation-failed", "Validation failed", nil)
		return
	}
	if r.Header.Get("X-CloudAfrica-Elevation") != "" && !strings.HasPrefix(tok, "cak_") {
		s.T.Errorf("an OAuth sign-in sent an elevation header")
	}
	if s.PairOnPoll > 0 && s.ElevationCalls >= s.PairOnPoll {
		g.Elevated = true
		if strings.HasPrefix(tok, "cak_") {
			s.tokenPairs[tok+"|"+secret] = true
		}
	}
	if g.Elevated {
		writeJSON(w, 200, map[string]any{"paired": true})
		return
	}
	page := s.URL + "/pair?h=abc"
	if s.PairURL != "" {
		page = s.PairURL
	}
	resp := map[string]any{"paired": false, "url": page, "code": "ABCD-1234"}
	if s.NoPairCode {
		delete(resp, "code")
	}
	writeJSON(w, 200, resp)
}

// PairToken records tok's pairing by secret, as a confirm on /pair would.
func (s *Server) PairToken(tok, secret string) { s.tokenPairs[tok+"|"+secret] = true }

const base64URLAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

// wellFormed is the gateway's rule for the elevation header.
func wellFormed(v string) bool {
	rest, ok := strings.CutPrefix(v, "cae_")
	if !ok || len(rest) < 43 {
		return false
	}
	for _, c := range rest {
		if !strings.ContainsRune(base64URLAlphabet, c) {
			return false
		}
	}
	return true
}
