package cli

import (
	"context"
	"errors"
	"net/http"
	"slices"
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

// scopeDetailPrefix starts the gateway's insufficient-scope detail, which
// is where the gateway names the missing scope.
const scopeDetailPrefix = "missing the required scope: "

// requiredScope is the scope an insufficient-scope refusal names: the word
// after scopeDetailPrefix in its detail, or a scope member should the
// gateway send one, else "".
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

// refusedScope is the scope an insufficient-scope refusal names, or "".
func refusedScope(err error) string {
	var e *cloudafrica.APIError
	if !errors.As(err, &e) || e.Status != http.StatusForbidden || e.Slug() != "insufficient-scope" {
		return ""
	}
	return requiredScope(e)
}

// loginGrants reports whether signing in again can add scope: it can for any
// scope, a non-default one through --scope. An unknown scope ("") counts as yes.
func loginGrants(scope string) bool {
	return scope == "" || strings.Contains(scope, ":")
}

// loginFlag is the `ca auth login` flag that requests scope ("" for a default one).
func loginFlag(scope string) string {
	if scope == "" || slices.Contains(defaultScopes, scope) {
		return ""
	}
	return " --scope " + scope
}

// errStillNotPaired is the refusal when a sign-in approved on the consent page
// is still not paired: the page did not pair it (the approver chose no protected
// scope, or the pairing failed).
var errStillNotPaired = errors.New("this sign-in is still not paired for protected actions")

// errNoProtectedScope refuses to pair a sign-in that was not granted any
// protected: scope: pairing cannot add one.
var errNoProtectedScope = errors.New("this sign-in was not granted any protected: scope, so pairing it would not " +
	"allow protected actions")

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
	if sc := refusedScope(err); !strings.HasPrefix(sc, "protected:") {
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
	case errors.Is(err, errNoProtectedScope), errors.Is(err, errStillNotPaired):
		return ExitStepUp
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
// CLOUDAFRICA_TOKEN supplied the credential, elevation whether
// CLOUDAFRICA_ELEVATION went with it.
func hint(err error, envToken, elevation bool) string {
	var he *hintedError
	if errors.As(err, &he) {
		return he.hint
	}
	var ae *cloudafrica.APIError
	if slug, ok := stepUpSlug(err); ok {
		if slug == "not-paired" {
			if envToken && elevation {
				return "CLOUDAFRICA_ELEVATION is no longer paired: the elevation window ended, or the portal session " +
					"of the person who confirmed it did. Run `ca auth elevate` again with this same CLOUDAFRICA_ELEVATION set; " +
					"it pairs that value again."
			}
			if envToken {
				return "CLOUDAFRICA_TOKEN is not paired for protected actions. A person runs " +
					"`CLOUDAFRICA_ELEVATION=$(ca auth elevate)`, confirms in the portal, and exports it for the script."
			}
			return "This is a protected action, and this sign-in is not paired for it. Run `ca auth login` and " +
				"approve it on the consent page, or in the portal open Connected apps and choose " +
				"\"Allow protected actions\"."
		}
		return scopeHint(refusedScope(err), envToken, true)
	}
	if sc := missingOrdinaryScope(err); sc != "" {
		return scopeHint(sc, envToken, false)
	}
	if errors.As(err, &ae) && ae.Status == http.StatusUnauthorized {
		return "The credential was refused. Run `ca auth login`, or check CLOUDAFRICA_TOKEN."
	}
	return ""
}

// scopeHint is the remedy for a refusal naming scope ("" when unknown);
// protected is whether the action also needs pairing afterwards
// (for CLOUDAFRICA_TOKEN, `ca auth elevate`; a sign-in is paired on the consent page).
func scopeHint(scope string, envToken, protected bool) string {
	named := scope
	if named == "" {
		named = "a scope this action needs"
	}
	switch {
	case protected && !loginGrants(scope) && envToken:
		return "CLOUDAFRICA_TOKEN was not granted " + scope + "; use a token that has it, then `ca auth elevate`."
	case protected && !loginGrants(scope):
		return "A `ca auth login` sign-in cannot hold " + scope + ". Put a user API token that has it in " +
			"CLOUDAFRICA_TOKEN, then pair it with `ca auth elevate`."
	case protected && envToken:
		return "CLOUDAFRICA_TOKEN was not granted " + named + "; use a token that has it, then `ca auth elevate`. " +
			"Or unset it and run `ca auth login" + loginFlag(scope) + "`."
	case envToken && scope == "":
		return "CLOUDAFRICA_TOKEN was not granted a scope this action needs; use a token that has it."
	case envToken:
		return "CLOUDAFRICA_TOKEN was not granted " + scope + "; use a token that has it."
	case !loginGrants(scope):
		return "A `ca auth login` sign-in cannot be granted " + scope + ". Use a user API token that has it, " +
			"in CLOUDAFRICA_TOKEN."
	}
	if scope == "" {
		return "This sign-in was not granted a scope this action needs. Run `ca auth login` and tick every scope " +
			"on the consent page."
	}
	return "This sign-in was not granted " + scope + ". Run `ca auth login" + loginFlag(scope) +
		"` and tick it on the consent page."
}
