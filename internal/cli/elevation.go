package cli

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

const (
	// elevationEnv holds a user API token's pairing secret, which `ca auth
	// elevate` prints for a script to set.
	elevationEnv = "CLOUDAFRICA_ELEVATION"
	// elevationHeader carries it.
	elevationHeader = "X-CloudAfrica-Elevation"
	elevationPrefix = "cae_"
	// elevationMinRandom is the shortest random part accepted: 32 bytes,
	// base64url, unpadded.
	elevationMinRandom = 43
)

var errElevationWithoutToken = errors.New(elevationEnv + " is set but CLOUDAFRICA_TOKEN is not: a sign-in pairs " +
	"through `ca auth elevate`, not this value. Run `unset " + elevationEnv + "`")

var errElevationMalformed = errors.New(elevationEnv + " must be " + elevationPrefix +
	" followed by at least 43 base64url characters, as `ca auth elevate` prints it")

// wellFormedElevation reports whether v is cae_ followed by at least 43
// base64url characters.
const base64URLAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

func wellFormedElevation(v string) bool {
	rest, ok := strings.CutPrefix(v, elevationPrefix)
	if !ok || len(rest) < elevationMinRandom {
		return false
	}
	for _, c := range rest {
		if !strings.ContainsRune(base64URLAlphabet, c) {
			return false
		}
	}
	return true
}

func newElevationSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return elevationPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// elevationTransport adds the pairing secret to every request. It is only
// the API client's base transport, which the SDK reaches after refusing any
// other origin, so the secret never leaves the endpoint.
type elevationTransport struct {
	base   http.RoundTripper
	secret string
}

func (t *elevationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set(elevationHeader, t.secret)
	return t.base.RoundTrip(r)
}
