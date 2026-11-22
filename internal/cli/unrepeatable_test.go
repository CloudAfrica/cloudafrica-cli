package cli

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type status503 struct{}

func (status503) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("busy"))}, nil
}

// Only a single-shot request has its 503 turned into an error; any other write
// gets the response, which the SDK retries.
func TestSingleShotTransportTouchesOnlySingleShotRequests(t *testing.T) {
	rt := &singleShotTransport{base: status503{}}
	plain, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://x/v1/vpcs", nil)
	resp, err := rt.RoundTrip(plain)
	if err != nil || resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("plain request: %v, %v", resp, err)
	}
	_ = resp.Body.Close()
	once, _ := http.NewRequestWithContext(singleShot(context.Background()), http.MethodPost, "http://x/v1/vpcs", nil)
	if resp, err := rt.RoundTrip(once); !errors.Is(err, errNotRetried) || resp != nil {
		t.Fatalf("single-shot request: %v, %v", resp, err)
	}
}
