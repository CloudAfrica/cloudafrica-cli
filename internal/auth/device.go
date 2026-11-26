package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	deviceGrantType = "urn:ietf:params:oauth:grant-type:device_code"
	// RFC 8628 §3.2 and §3.5.
	deviceDefaultInterval = 5 * time.Second
	deviceSlowDown        = 5 * time.Second
)

var errDeviceCodeExpired = errors.New("the code expired before the sign-in was approved")

type deviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

// DeviceLogin runs the RFC 8628 device flow, for a browser on another
// machine: it prints a URL and a code, then polls the token endpoint until the
// sign-in is approved or denied, or the code or o.Timeout runs out. A poll
// that gets no answer, a 429 or a 5xx is repeated less often. m must name a
// device authorization endpoint.
func DeviceLogin(ctx context.Context, hc *http.Client, m *Metadata, o LoginOptions) (*Token, error) {
	if m.DeviceAuthorizationEndpoint == "" {
		return nil, errors.New("this server has no device sign-in")
	}
	form := url.Values{
		"client_id": {ClientID},
		"scope":     {strings.Join(o.Scopes, " ")},
		"resource":  {m.Resource},
	}
	if o.Account != "" {
		form.Set("account", o.Account)
	}
	var d deviceAuthorization
	if err := postForm(ctx, hc, m.DeviceAuthorizationEndpoint, form, &d); err != nil {
		return nil, fmt.Errorf("starting the device sign-in: %w", err)
	}
	if d.DeviceCode == "" || d.UserCode == "" || d.VerificationURI == "" || d.ExpiresIn <= 0 {
		return nil, errors.New("oauth: the device authorization response is incomplete")
	}
	if err := RequireTLS(d.VerificationURI); err != nil {
		return nil, fmt.Errorf("refusing the sign-in page: %w", err)
	}
	if o.Out != nil {
		fmt.Fprintf(o.Out, "To sign in, visit %s and enter %s\n", d.VerificationURI, d.UserCode)
		if d.VerificationURIComplete != "" && RequireTLS(d.VerificationURIComplete) == nil {
			fmt.Fprintf(o.Out, "Direct link (the page may still ask for the code): %s\n", d.VerificationURIComplete)
		}
		fmt.Fprintln(o.Out, "Waiting for approval...")
	}
	return pollDevice(ctx, hc, m, &d, o)
}

func pollDevice(ctx context.Context, hc *http.Client, m *Metadata, d *deviceAuthorization,
	o LoginOptions) (*Token, error) {
	interval := time.Duration(d.Interval) * time.Second
	if interval <= 0 {
		interval = deviceDefaultInterval
	}
	limit, byExpiry := time.Duration(d.ExpiresIn)*time.Second, true
	if o.Timeout > 0 && o.Timeout < limit {
		limit, byExpiry = o.Timeout, false
	}
	wait := o.Wait
	if wait == nil {
		wait = sleep
	}
	wctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	var unanswered error // the last poll's failure to get an answer
	gaveUp := func() error {
		err := fmt.Errorf("waiting for the sign-in: %w", wctx.Err())
		if byExpiry && ctx.Err() == nil {
			err = errDeviceCodeExpired
		}
		if unanswered != nil {
			err = fmt.Errorf("%w (the last check failed: %v)", err, unanswered)
		}
		return err
	}
	warned := false
	for {
		if err := wait(wctx, interval); err != nil || wctx.Err() != nil {
			return nil, gaveUp()
		}
		var t Token
		err := postForm(wctx, hc, m.TokenEndpoint, url.Values{
			"grant_type":  {deviceGrantType},
			"client_id":   {ClientID},
			"device_code": {d.DeviceCode},
		}, &t)
		if err == nil {
			if t.AccessToken == "" {
				return nil, errors.New("oauth: token response carries no access_token")
			}
			return &t, nil
		}
		if wctx.Err() != nil {
			return nil, gaveUp()
		}
		var oe *OAuthError
		var ue *url.Error
		switch {
		case errors.As(err, &oe) && (oe.Status == http.StatusTooManyRequests || oe.Status >= 500),
			errors.As(err, &ue):
			unanswered = err
			interval += deviceSlowDown
			if !warned && o.Out != nil {
				warned = true
				fmt.Fprintf(o.Out, "Warning: could not check the sign-in (%v); still waiting.\n", err)
			}
			continue
		case oe == nil:
			return nil, err
		}
		unanswered = nil
		switch oe.Code {
		case "authorization_pending":
		case "slow_down":
			interval += deviceSlowDown
		case "access_denied":
			return nil, errors.New("sign-in was denied (access_denied)")
		case "expired_token":
			return nil, errDeviceCodeExpired
		case "invalid_grant":
			// The server cancels a grant whose device code is redeemed twice,
			// which a repeat of a poll whose answer was lost does.
			return nil, fmt.Errorf("the sign-in was cancelled: the server no longer accepts this code, "+
				"as when it is used twice: %w", err)
		default:
			return nil, err
		}
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
