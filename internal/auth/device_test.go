package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
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
		"authorization_pending"}
	tok, waits, out, err := deviceRun(t, context.Background(), fake, LoginOptions{Account: "acc-9"})
	if err != nil {
		t.Fatal(err)
	}
	if tok.RefreshToken == "" || fake.GrantOf(tok.AccessToken) == nil || fake.DevicePolls != 7 {
		t.Errorf("token %+v after %d polls", tok, fake.DevicePolls)
	}
	if want := seconds(2, 2, 7, 7, 12, 17, 17); !slices.Equal(waits, want) {
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

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDeviceLoginEndsOnARefusal(t *testing.T) {
	for answer, want := range map[string]string{
		"access_denied":  "sign-in was denied (access_denied)",
		"expired_token":  "the code expired",
		"invalid_grant":  "the sign-in was cancelled",
		"invalid_client": "invalid_client",
		"404":            "HTTP 404",
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

// A sign-in that expires while the server is failing says why.
func TestDeviceLoginExpiryNamesTheLastFailure(t *testing.T) {
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
