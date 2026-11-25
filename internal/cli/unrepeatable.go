package cli

import (
	"context"
	"errors"
	"io"
	"net/http"
)

// unrepeatable lists the writes whose API contract says a retry after a 5xx
// can repeat them (a second reserved address, say), with the hint printed when
// one gets a 503; %[1]s is the first path parameter's given value.
var unrepeatable = map[string]string{
	"createVpcReservedIp": "The address may have been reserved anyway; check `ca vpcs reserved-ips list --vpc %[1]s` " +
		"before trying again.",
}

// errNotRetried is a 503 to an unrepeatable write.
var errNotRetried = errors.New("HTTP 503 Service Unavailable; not retried, because a retry could repeat the write")

type singleShotKey struct{}

func singleShot(ctx context.Context) context.Context {
	return context.WithValue(ctx, singleShotKey{}, true)
}

// singleShotTransport returns a 503 to a single-shot request as an error. The
// SDK retries a 503 response but returns a transport error as it is.
type singleShotTransport struct {
	base http.RoundTripper
}

func (t *singleShotTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusServiceUnavailable || req.Context().Value(singleShotKey{}) == nil {
		return resp, err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
	return nil, errNotRetried
}
