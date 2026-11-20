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

// stepUpSlug reports whether err is the gateway refusing a protected action
// for want of elevation: not-paired (scope held, window lapsed) or
// insufficient-scope (the grant lacks the scope).
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
		if e.Scope == "" || strings.HasPrefix(e.Scope, "protected:") {
			return s, true
		}
	}
	return "", false
}

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

func hint(err error) string {
	var ae *cloudafrica.APIError
	if slug, ok := stepUpSlug(err); ok {
		if slug == "insufficient-scope" && errors.As(err, &ae) && ae.Scope != "" {
			return "This needs the " + ae.Scope + " scope. Run `ca auth login --elevate`, or in the portal open " +
				"Connected apps and choose \"Allow protected actions\"."
		}
		return "This is a protected action. Run `ca auth login --elevate`, or in the portal open " +
			"Connected apps and choose \"Allow protected actions\" to re-elevate this sign-in."
	}
	if errors.As(err, &ae) && ae.Status == http.StatusUnauthorized {
		return "The credential was refused. Run `ca auth login`, or check CLOUDAFRICA_TOKEN."
	}
	return ""
}
