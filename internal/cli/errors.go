package cli

import (
	"context"
	"errors"
	"net/http"
	"strings"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/auth"
)

// Exit codes.
const (
	ExitOK      = 0
	ExitAPI     = 1
	ExitUsage   = 2
	ExitTimeout = 3
	ExitAuth    = 4
	ExitStepUp  = 5
)

type usageError struct{ err error }

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

func usageErr(err error) error {
	if err == nil {
		return nil
	}
	return &usageError{err}
}

// scopeDetailPrefix starts the gateway's insufficient-scope detail; the
// problem carries the scope nowhere else.
const scopeDetailPrefix = "missing the required scope: "

// requiredScope is the scope an insufficient-scope refusal names: the
// problem's scope member when present, else the word after
// scopeDetailPrefix in its detail, else "".
func requiredScope(e *cloudafrica.APIError) string {
	if e.Scope != "" {
		return e.Scope
	}
	_, after, ok := strings.Cut(e.Detail, scopeDetailPrefix)
	if !ok {
		return ""
	}
	if f := strings.Fields(after); len(f) > 0 {
		return strings.TrimRight(f[0], ".,;")
	}
	return ""
}

// stepUpSlug reports whether err is the gateway refusing a protected action:
// not-paired (scope held, this sign-in not paired) or insufficient-scope
// (the grant lacks a protected: scope).
func stepUpSlug(err error) (string, bool) {
	var e *cloudafrica.APIError
	if !errors.As(err, &e) || e.Status != http.StatusForbidden {
		return "", false
	}
	switch s := e.Slug(); s {
	case "not-paired":
		return s, true
	case "insufficient-scope":
		// A missing ordinary scope (read:/write:) is a permission error that
		// elevation cannot fix.
		if sc := requiredScope(e); sc == "" || strings.HasPrefix(sc, "protected:") {
			return s, true
		}
	}
	return "", false
}

// missingOrdinaryScope is the read:/write: scope an insufficient-scope
// refusal names, or "".
func missingOrdinaryScope(err error) string {
	var e *cloudafrica.APIError
	if !errors.As(err, &e) || e.Status != http.StatusForbidden || e.Slug() != "insufficient-scope" {
		return ""
	}
	if sc := requiredScope(e); !strings.HasPrefix(sc, "protected:") {
		return sc
	}
	return ""
}

// hintedError carries a remedy for one command's refusal.
type hintedError struct {
	err  error
	hint string
}

func (e *hintedError) Error() string { return e.err.Error() }
func (e *hintedError) Unwrap() error { return e.err }

// ExitCode maps an error to the documented exit code.
func ExitCode(err error) int {
	var ue *usageError
	var ae *cloudafrica.APIError
	switch {
	case err == nil:
		return ExitOK
	case errors.As(err, &ue):
		return ExitUsage
	case errors.Is(err, auth.ErrLoginRequired):
		return ExitAuth
	case errors.Is(err, context.DeadlineExceeded):
		return ExitTimeout
	}
	if _, ok := stepUpSlug(err); ok {
		return ExitStepUp
	}
	if errors.As(err, &ae) && ae.Status == http.StatusUnauthorized {
		return ExitAuth
	}
	return ExitAPI
}

// hint is the remedy printed under an error; envToken is whether
// CLOUDAFRICA_TOKEN supplied the credential.
func hint(err error, envToken bool) string {
	var he *hintedError
	if errors.As(err, &he) {
		return he.hint
	}
	var ae *cloudafrica.APIError
	if slug, ok := stepUpSlug(err); ok {
		if envToken {
			return "CLOUDAFRICA_TOKEN is a user API token: it uses protected actions through its own pairing " +
				"(POST /v1/elevation-requests with an X-CloudAfrica-Elevation header). Unset it to use your profile."
		}
		if slug == "insufficient-scope" {
			scope := "protected:compute"
			if errors.As(err, &ae) && requiredScope(ae) != "" {
				scope = requiredScope(ae)
			}
			return "This sign-in was not granted " + scope + ". Run `ca auth login` and tick it on the consent " +
				"page, then `ca auth elevate`."
		}
		return "This is a protected action, and this sign-in is not paired for it. Run `ca auth elevate`, or in " +
			"the portal open Connected apps and choose \"Allow protected actions\"."
	}
	if sc := missingOrdinaryScope(err); sc != "" && !envToken {
		return "This sign-in was not granted " + sc + ". Run `ca auth login` and tick it on the consent page."
	}
	if errors.As(err, &ae) && ae.Status == http.StatusUnauthorized {
		return "The credential was refused. Run `ca auth login`, or check CLOUDAFRICA_TOKEN."
	}
	return ""
}
