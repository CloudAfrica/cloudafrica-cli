package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/CloudAfrica/cloudafrica-cli/internal/profile"
)

// RefreshMargin is how close to expiry an access token is refreshed.
const RefreshMargin = 60 * time.Second

// ErrLoginRequired means the profile cannot produce a token without a new
// browser sign-in.
var ErrLoginRequired = errors.New("sign-in required")

// LoginRequiredError carries why, and which profile to sign in again.
type LoginRequiredError struct {
	Profile string
	Reason  string
}

func (e *LoginRequiredError) Error() string {
	cmd := "ca auth login"
	if e.Profile != profile.Default {
		cmd += " --profile " + e.Profile
	}
	return fmt.Sprintf("%s; run `%s`", e.Reason, cmd)
}

func (e *LoginRequiredError) Unwrap() error { return ErrLoginRequired }

// Source is a cloudafrica.TokenSource backed by a stored profile. It
// refreshes the access token when it is within RefreshMargin of expiry and
// writes the rotated refresh token to the profile before handing out the new
// access token. It is safe for concurrent use, and across processes through
// the profile lock.
type Source struct {
	Store profile.Store
	Name  string
	HTTP  *http.Client
	Now   func() time.Time

	mu   sync.Mutex
	cur  *profile.Profile
	meta *Metadata
}

func (s *Source) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Source) fresh(p *profile.Profile) bool {
	return p != nil && p.AccessToken != "" && s.now().Add(RefreshMargin).Before(p.Expiry)
}

// Profile returns the profile as last loaded or refreshed.
func (s *Source) Profile() (*profile.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur == nil {
		p, err := s.Store.Load(s.Name)
		if err != nil {
			return nil, err
		}
		s.cur = p
	}
	cp := *s.cur
	return &cp, nil
}

// Token returns a live access token, refreshing it if needed.
func (s *Source) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur == nil {
		p, err := s.Store.Load(s.Name)
		if errors.Is(err, profile.ErrNotFound) {
			return "", &LoginRequiredError{Profile: s.Name, Reason: fmt.Sprintf("profile %q is not signed in", s.Name)}
		}
		if err != nil {
			return "", err
		}
		s.cur = p
	}
	if s.fresh(s.cur) {
		return s.cur.AccessToken, nil
	}

	release, err := s.Store.Lock(ctx, s.Name)
	if err != nil {
		return "", err
	}
	defer release()
	// Another process may have refreshed while we waited for the lock; its
	// rotation spent the refresh token we hold.
	p, err := s.Store.Load(s.Name)
	if errors.Is(err, profile.ErrNotFound) {
		return "", &LoginRequiredError{Profile: s.Name, Reason: fmt.Sprintf("profile %q was signed out", s.Name)}
	}
	if err != nil {
		return "", err
	}
	if s.fresh(p) {
		s.cur = p
		return p.AccessToken, nil
	}
	if p.RefreshToken == "" {
		return "", &LoginRequiredError{Profile: s.Name, Reason: "the session has expired"}
	}
	if s.meta == nil {
		m, err := Discover(ctx, s.HTTP, p.Endpoint)
		if err != nil {
			return "", err
		}
		s.meta = m
	}
	t, err := Refresh(ctx, s.HTTP, s.meta, p.RefreshToken)
	if IsInvalidGrant(err) {
		return "", &LoginRequiredError{Profile: s.Name, Reason: "the session has ended (refresh refused: " + err.Error() + ")"}
	}
	if err != nil {
		return "", fmt.Errorf("refreshing the access token: %w", err)
	}
	next := *p
	next.AccessToken = t.AccessToken
	next.Expiry = t.Expiry(s.now())
	next.RefreshToken = t.RefreshToken
	if t.Scope != "" {
		next.Scope = t.Scope
	}
	if err := s.Store.Save(s.Name, &next); err != nil {
		// The old refresh token is spent; the new one is lost with this save.
		return "", &LoginRequiredError{Profile: s.Name,
			Reason: fmt.Sprintf("the refreshed session could not be saved (%v)", err)}
	}
	s.cur = &next
	return next.AccessToken, nil
}
