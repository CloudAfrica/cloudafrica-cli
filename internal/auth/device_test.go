package auth

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

// deviceRun signs in against fake, recording each pause between polls.
func deviceRun(t *testing.T, ctx context.Context, fake *fakeapi.Server, o LoginOptions) (
	*Token, []time.Duration, string, error) {
	t.Helper()
	m, err := Discover(ctx, HTTPClient(), fake.URL)
	if err != nil {
		t.Fatal(err)
	}
	var waits []time.Duration
	var out bytes.Buffer
	o.Out, o.Scopes = &out, []string{"read:compute", "protected:compute"}
	if o.Wait == nil {
		o.Wait = func(ctx context.Context, d time.Duration) error {
			waits = append(waits, d)
			return ctx.Err()
		}
	}
	tok, err := DeviceLogin(ctx, HTTPClient(), m, o)
	return tok, waits, out.String(), err
}

func seconds(n ...int) []time.Duration {
	var out []time.Duration
	for _, v := range n {
		out = append(out, time.Duration(v)*time.Second)
	}
	return out
}

func TestDeviceLoginPollsUntilApproved(t *testing.T) {
	fake := fakeapi.New(t)
	fake.DeviceInterval = 2
	fake.DeviceAnswers = []string{"authorization_pending", "slow_down", "authorization_pending", "503", "429",
		"500", "502", "authorization_pending"}
	tok, waits, out, err := deviceRun(t, context.Background(), fake, LoginOptions{Account: "acc-9"})
	if err != nil {
		t.Fatal(err)
	}
	if tok.RefreshToken == "" || fake.GrantOf(tok.AccessToken) == nil || fake.DevicePolls != 9 {
		t.Errorf("token %+v after %d polls", tok, fake.DevicePolls)
	}
	if want := seconds(2, 2, 7, 7, 12, 17, 22, 27, 27); !slices.Equal(waits, want) {
		t.Errorf("waits = %v, want %v", waits, want)
	}
	form := fake.DeviceAuthorizations[0]
	if form.Get("scope") != "read:compute protected:compute" || form.Get("account") != "acc-9" {
		t.Errorf("form = %v", form)
	}
	for _, want := range []string{
		"To sign in, visit " + fake.URL + "/ui/device and enter BCDF-GHJK\n",
		fake.URL + "/ui/device?user_code=BCDF-GHJK\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q: %s", want, out)
		}
	}
	if strings.Count(out, "could not check the sign-in") != 1 {
		t.Errorf("want one warning: %s", out)
	}
}

func TestDeviceLoginDefaultsTheIntervalAndOmitsNoAccount(t *testing.T) {
	fake := fakeapi.New(t)
	_, waits, _, err := deviceRun(t, context.Background(), fake, LoginOptions{})
	if err != nil || !slices.Equal(waits, seconds(5)) || fake.DeviceAuthorizations[0].Has("account") {
		t.Errorf("err %v, waits %v, form %v", err, waits, fake.DeviceAuthorizations)
	}
}

func TestDeviceLoginRetriesADroppedPoll(t *testing.T) {
	fake := fakeapi.New(t)
	m, err := Discover(context.Background(), HTTPClient(), fake.URL)
	if err != nil {
		t.Fatal(err)
	}
	dropped := false
	hc := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/oauth/token" && !dropped {
			dropped = true
			return nil, errors.New("connection reset")
		}
		return http.DefaultTransport.RoundTrip(r)
	})}
	tok, err := DeviceLogin(context.Background(), hc, m, LoginOptions{
		Wait: func(context.Context, time.Duration) error { return nil }})
	if err != nil || tok.AccessToken == "" || !dropped {
		t.Errorf("tok %+v, err %v", tok, err)
	}
}

// An answer cut off in its body is a lost answer: the poll is repeated.
func TestDeviceLoginRetriesACutOffAnswer(t *testing.T) {
	fake := fakeapi.New(t)
	m, err := Discover(context.Background(), HTTPClient(), fake.URL)
	if err != nil {
		t.Fatal(err)
	}
	cut := false
	hc := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/oauth/token" && !cut {
			cut = true
			return &http.Response{StatusCode: 400, Header: http.Header{},
				Body: io.NopCloser(iotest.ErrReader(io.ErrUnexpectedEOF))}, nil
		}
		return http.DefaultTransport.RoundTrip(r)
	})}
	var out bytes.Buffer
	tok, err := DeviceLogin(context.Background(), hc, m, LoginOptions{Out: &out,
		Wait: func(context.Context, time.Duration) error { return nil }})
	if err != nil || tok.AccessToken == "" || !cut {
		t.Fatalf("tok %+v, err %v", tok, err)
	}
	if !strings.Contains(out.String(), `Post "`+fake.URL+`/oauth/token": unexpected EOF`) {
		t.Errorf("the warning does not name the request: %s", out.String())
	}
}

// A caller that gives up during a poll gets that, not a failed check.
func TestDeviceLoginCancelledDuringAPoll(t *testing.T) {
	fake := fakeapi.New(t)
	m, err := Discover(context.Background(), HTTPClient(), fake.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hc := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/oauth/token" {
			cancel()
			return nil, ctx.Err()
		}
		return http.DefaultTransport.RoundTrip(r)
	})}
	var out bytes.Buffer
	_, err = DeviceLogin(ctx, hc, m, LoginOptions{Out: &out,
		Wait: func(ctx context.Context, _ time.Duration) error { return ctx.Err() }})
	if !errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "last check") ||
		strings.Contains(out.String(), "Warning") {
		t.Errorf("err %v, out %s", err, out.String())
	}
}

// Without a Wait the polls are paced by the server's interval.
func TestDeviceLoginPacesItsPolls(t *testing.T) {
	fake := fakeapi.New(t)
	fake.DeviceInterval = 1
	m, err := Discover(context.Background(), HTTPClient(), fake.URL)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := DeviceLogin(context.Background(), HTTPClient(), m, LoginOptions{}); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < time.Second || fake.DevicePolls != 1 {
		t.Errorf("%d polls in %v", fake.DevicePolls, d)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled sleep: %v", err)
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDeviceLoginEndsOnARefusal(t *testing.T) {
	for answer, want := range map[string]string{
		"access_denied":  "sign-in was denied (access_denied)",
		"expired_token":  "the code expired",
		"invalid_grant":  "the sign-in was cancelled",
		"invalid_client": "invalid_client",
		"404":            "HTTP 404",
		"200":            "decoding",
		"empty":          "no access_token",
	} {
		t.Run(answer, func(t *testing.T) {
			fake := fakeapi.New(t)
			fake.DeviceAnswers = []string{answer}
			tok, _, _, err := deviceRun(t, context.Background(), fake, LoginOptions{})
			if tok != nil || err == nil || !strings.Contains(err.Error(), want) || fake.DevicePolls != 1 {
				t.Errorf("tok %v, err %v, polls %d; want %q after 1", tok, err, fake.DevicePolls, want)
			}
		})
	}
}

func TestDeviceLoginRefusesAnIncompleteOrInsecureResponse(t *testing.T) {
	good := map[string]any{"device_code": "d", "user_code": "u", "verification_uri": "https://p.example/ui/device",
		"expires_in": 600}
	for field, bad := range map[string]any{
		"device_code": "", "user_code": "", "verification_uri": "", "expires_in": 0,
	} {
		fake := fakeapi.New(t)
		body := map[string]any{}
		for k, v := range good {
			body[k] = v
		}
		body[field] = bad
		fake.DeviceBody = body
		if _, _, out, err := deviceRun(t, context.Background(), fake, LoginOptions{}); err == nil ||
			!strings.Contains(err.Error(), "incomplete") || out != "" || fake.DevicePolls != 0 {
			t.Errorf("%s: err %v, out %q", field, err, out)
		}
	}
	fake := fakeapi.New(t)
	good["verification_uri"] = "http://p.example/ui/device"
	fake.DeviceBody = good
	if _, _, out, err := deviceRun(t, context.Background(), fake, LoginOptions{}); err == nil ||
		!strings.Contains(err.Error(), "refusing the sign-in page") || out != "" {
		t.Errorf("http page: err %v, out %q", err, out)
	}
}

func TestDeviceLoginReportsARefusedStart(t *testing.T) {
	fake := fakeapi.New(t)
	fake.DeviceRefusal = "invalid_scope"
	_, _, out, err := deviceRun(t, context.Background(), fake, LoginOptions{})
	if err == nil || !strings.Contains(err.Error(), "starting the device sign-in: oauth: HTTP 400 invalid_scope") ||
		out != "" || fake.DevicePolls != 0 {
		t.Errorf("err %v, out %q, polls %d", err, out, fake.DevicePolls)
	}
}

// The direct link is optional, and is not printed when it is not TLS.
func TestDeviceLoginDirectLink(t *testing.T) {
	for name, link := range map[string]string{"absent": "", "insecure": "http://p.example/ui/device?user_code=u"} {
		fake := fakeapi.New(t)
		fake.DeviceBody = map[string]any{"device_code": "d", "user_code": "u", "expires_in": 600,
			"verification_uri": "https://p.example/ui/device", "verification_uri_complete": link}
		_, _, out, _ := deviceRun(t, context.Background(), fake, LoginOptions{})
		want := "To sign in, visit https://p.example/ui/device and enter u\nWaiting for approval...\n"
		if out != want {
			t.Errorf("%s: output %q, want %q", name, out, want)
		}
	}
}

// The wait ends at the code's lifetime or at Timeout, whichever is sooner,
// and a cancelled caller is not an expired code.
func TestDeviceLoginStopsWaiting(t *testing.T) {
	block := func(ctx context.Context, _ time.Duration) error {
		<-ctx.Done()
		return ctx.Err()
	}
	fake := fakeapi.New(t)
	_, _, _, err := deviceRun(t, context.Background(), fake,
		LoginOptions{Timeout: 50 * time.Millisecond, Wait: block})
	if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errDeviceCodeExpired) {
		t.Errorf("timeout: %v", err)
	}
	fake.DeviceBody = map[string]any{"device_code": "d", "user_code": "u", "verification_uri": fake.URL,
		"expires_in": 1}
	start := time.Now()
	_, _, _, err = deviceRun(t, context.Background(), fake, LoginOptions{Timeout: time.Minute, Wait: block})
	if !errors.Is(err, errDeviceCodeExpired) || time.Since(start) > 10*time.Second {
		t.Errorf("expiry: %v after %v", err, time.Since(start))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, _, err = deviceRun(t, ctx, fakeapi.New(t), LoginOptions{
		Wait: func(context.Context, time.Duration) error {
			cancel()
			return ctx.Err()
		}})
	if !errors.Is(err, context.Canceled) || errors.Is(err, errDeviceCodeExpired) {
		t.Errorf("cancel: %v", err)
	}
}

// A failure that a later answer followed is not named when the wait ends.
func TestDeviceLoginForgetsAnAnsweredFailure(t *testing.T) {
	fake := fakeapi.New(t)
	fake.DeviceAnswers = []string{"503", "authorization_pending"}
	polls := 0
	_, _, _, err := deviceRun(t, context.Background(), fake, LoginOptions{Timeout: time.Second,
		Wait: func(ctx context.Context, _ time.Duration) error {
			if polls++; polls > 2 {
				<-ctx.Done()
			}
			return ctx.Err()
		}})
	if !errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "last check") {
		t.Errorf("err = %v", err)
	}
}

// A sign-in that times out while the server is failing says why.
func TestDeviceLoginTimeoutNamesTheLastFailure(t *testing.T) {
	fake := fakeapi.New(t)
	fake.DeviceAnswers = []string{"503"}
	polls := 0
	_, _, _, err := deviceRun(t, context.Background(), fake, LoginOptions{Timeout: time.Second,
		Wait: func(ctx context.Context, _ time.Duration) error {
			if polls++; polls > 1 {
				<-ctx.Done()
			}
			return ctx.Err()
		}})
	if err == nil || !strings.Contains(err.Error(), "the last check failed: oauth: HTTP 503") {
		t.Errorf("err = %v", err)
	}
}

func TestDeviceLoginNeedsAnEndpoint(t *testing.T) {
	fake := fakeapi.New(t)
	fake.NoDeviceFlow = true
	m, err := Discover(context.Background(), HTTPClient(), fake.URL)
	if err != nil || m.DeviceAuthorizationEndpoint != "" {
		t.Fatalf("without a device endpoint: %+v, %v", m, err)
	}
	if _, err := DeviceLogin(context.Background(), HTTPClient(), m, LoginOptions{}); err == nil {
		t.Error("DeviceLogin ran without an endpoint")
	}
}
