package cli

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

const bucketList = `{"data":[{"id":"b-1","name":"logs-7f3a","displayName":"Logs","status":"active",
	"quotaBytes":2147483648,"sizeActualBytes":1073741824,"objectCount":42,"createdAt":"2026-11-01T08:00:00Z"}]}`

const keyList = `{"data":[
	{"id":"k-0","name":"master","accessKey":"AKM","accessType":"full","master":true},
	{"id":"k-1","name":"ci","accessKey":"AKCI","accessType":"read","master":false,"bucketName":"logs-7f3a"}]}`

func s3Harness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	g := protected()
	g.Elevated = true
	h.seed("default", g, time.Now().Add(time.Hour))
	return h
}

func contains(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

// The bucket table shows the quota and the on-disk size in bytes units, -q
// prints ids, and a bucket named on the command line resolves to its id.
func TestBucketListGetAndNameResolution(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/buckets":     {Body: bucketList},
		"GET /v1/buckets/b-1": {Body: `{"id":"b-1","name":"logs-7f3a","status":"active","objectCount":42}`},
	})
	h.mustRun("bucket", "list", "-o", "table")
	contains(t, h.stdout.String(), "QUOTA", "ON DISK", "2.0 GiB", "1.0 GiB", "logs-7f3a", "42")
	h.mustRun("bucket", "list", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "b-1" {
		t.Errorf("-q = %q", got)
	}
	h.mustRun("bucket", "get", "logs-7f3a", "-o", "json")
	contains(t, h.stdout.String(), `"objectCount": 42`)
}

// A bucket delete sends the confirmation name the API compares exactly, and
// refuses to write when the flag is missing, even with --yes.
func TestBucketDeleteSendsConfirmationName(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/buckets":                     {Body: bucketList},
		"POST /v1/buckets/b-1/actions/delete": {Status: 202, Body: `{"operationId":"op/bucket-del"}`},
	})
	if code := h.run("bucket", "delete", "logs-7f3a", "--yes"); code != ExitUsage || len(h.fake.Writes) != 0 {
		t.Errorf("without --confirm-bucket-name: exit %d, writes %d", code, len(h.fake.Writes))
	}
	h.mustRun("bucket", "delete", "logs-7f3a", "--confirm-bucket-name", "logs-7f3a", "--yes", "--no-wait")
	if len(h.fake.WriteURLs) != 1 || h.fake.WriteURLs[0] != "POST /v1/buckets/b-1/actions/delete" {
		t.Fatalf("writes %v", h.fake.WriteURLs)
	}
	if got := h.fake.Bodies[0]["confirmBucketName"]; got != "logs-7f3a" {
		t.Errorf("body = %v", h.fake.Bodies[0])
	}
}

// An order carries the optional fields, an Idempotency-Key, and refuses a
// versioning value the API does not know.
func TestBucketCreateBodyAndKey(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{"POST /v1/buckets": {Status: 202, Body: `{"operationId":"op/bucket-new"}`}})
	h.mustRun("bucket", "create", "--display-name", "audit", "--object-lock", "--versioning", "enabled", "--no-wait")
	b := h.fake.Bodies[0]
	if b["displayName"] != "audit" || b["objectLock"] != true || b["versioning"] != "enabled" || len(b) != 3 {
		t.Errorf("body = %v", b)
	}
	if len(h.fake.IdemKeys) != 1 || h.fake.IdemKeys[0] == "" {
		t.Errorf("idempotency keys = %q", h.fake.IdemKeys)
	}
	if code := h.run("bucket", "create", "--versioning", "unknown", "--no-wait"); code != ExitUsage || len(h.fake.Writes) != 1 {
		t.Errorf("unknown versioning: exit %d, writes %d", code, len(h.fake.Writes))
	}
}

// Versioning reads its mode, sets one of the three the API accepts, and
// refuses "unknown" (a state the API only reports) before writing.
func TestBucketVersioningGetAndSet(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/buckets":                  {Body: bucketList},
		"GET /v1/buckets/b-1/versioning":   {Body: `{"mode":"suspended"}`},
		"PATCH /v1/buckets/b-1/versioning": {Status: 204},
	})
	h.mustRun("bucket", "versioning", "get", "logs-7f3a", "-o", "table")
	contains(t, h.stdout.String(), "MODE", "suspended")
	h.mustRun("bucket", "versioning", "set", "logs-7f3a", "--mode", "enabled", "--yes")
	if len(h.fake.WriteURLs) != 1 || h.fake.WriteURLs[0] != "PATCH /v1/buckets/b-1/versioning" || h.fake.Bodies[0]["mode"] != "enabled" {
		t.Errorf("writes %v bodies %v", h.fake.WriteURLs, h.fake.Bodies)
	}
	for _, mode := range []string{"unknown", ""} {
		if code := h.run("bucket", "versioning", "set", "logs-7f3a", "--mode", mode, "--yes"); code != ExitUsage || len(h.fake.Writes) != 1 {
			t.Errorf("--mode %q: exit %d, writes %d", mode, code, len(h.fake.Writes))
		}
	}
}

// Usage sends the window as a query and prints the traffic in bytes units.
func TestBucketUsageSendsHoursAndPrintsTraffic(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/buckets": {Body: bucketList},
		"GET /v1/buckets/b-1/usage": {Body: `{"bucketName":"logs-7f3a","sizeBytes":2048,"objectCount":3,
			"windowHours":168,"bytesUploaded":1048576,"bytesDownloaded":0,"operations":9}`},
	})
	h.mustRun("bucket", "usage", "logs-7f3a", "--hours", "168", "-o", "table")
	contains(t, h.stdout.String(), "UPLOADED", "1.0 MiB", "REQUESTS", "168")
	if !slices.Contains(h.fake.ReadURLs, "GET /v1/buckets/b-1/usage?hours=168") {
		t.Errorf("reads %v", h.fake.ReadURLs)
	}
}

// The key list shows the master key and the account-level key's empty bucket.
func TestBucketKeyList(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{"GET /v1/bucket-keys": {Body: keyList}})
	h.mustRun("bucket-key", "list", "-o", "table")
	contains(t, h.stdout.String(), "ACCESS KEY", "AKM", "AKCI", "logs-7f3a", "yes")
	h.mustRun("bucket-key", "list", "-q")
	if got := strings.Fields(h.stdout.String()); len(got) != 2 || got[0] != "k-0" || got[1] != "k-1" {
		t.Errorf("-q = %v", got)
	}
}

// A key create prints the secret the response carries (once), -q prints the
// key's id, JSON is the body verbatim, and a replay without a secret still
// prints the key.
func TestBucketKeyCreatePrintsSecret(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{"POST /v1/bucket-keys": {Status: 201, Body: `{"key":{"id":"k-2","name":"ci2",
		"accessKey":"AKCI2","accessType":"full"},"secretKey":"s3cr3t"}`}})
	h.mustRun("bucket-key", "create", "--name", "ci2", "--access-type", "full", "--yes", "-o", "table")
	contains(t, h.stdout.String(), "SECRET KEY", "s3cr3t", "AKCI2")
	if b := h.fake.Bodies[0]; b["name"] != "ci2" || b["accessType"] != "full" || len(b) != 2 {
		t.Errorf("body = %v", b)
	}
	if len(h.fake.IdemKeys) != 1 || h.fake.IdemKeys[0] == "" {
		t.Errorf("idempotency keys = %q", h.fake.IdemKeys)
	}
	h.mustRun("bucket-key", "create", "--name", "ci2", "--access-type", "full", "--yes", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "k-2" {
		t.Errorf("-q = %q", got)
	}
	h.mustRun("bucket-key", "create", "--name", "ci2", "--access-type", "full", "--yes", "-o", "json")
	contains(t, h.stdout.String(), `"secretKey": "s3cr3t"`)
	canned(h, map[string]fakeapi.Canned{"POST /v1/bucket-keys": {Status: 201, Body: `{"key":{"id":"k-2","name":"ci2",
		"accessKey":"AKCI2","accessType":"full"}}`}})
	h.mustRun("bucket-key", "create", "--name", "ci2", "--access-type", "full", "--yes", "-o", "table")
	contains(t, h.stdout.String(), "AKCI2")
	if strings.Contains(h.stdout.String(), "s3cr3t") {
		t.Errorf("a replay printed a secret: %s", h.stdout)
	}
	for name, args := range map[string][]string{
		"no name":        {"--access-type", "read", "--yes"},
		"no access type": {"--name", "x", "--yes"},
		"unknown access": {"--name", "x", "--access-type", "admin", "--yes"},
	} {
		n := len(h.fake.Writes)
		if code := h.run(append([]string{"bucket-key", "create"}, args...)...); code != ExitUsage || len(h.fake.Writes) != n {
			t.Errorf("%s: exit %d, writes %d", name, code, len(h.fake.Writes)-n)
		}
	}
}

// A key is named by its name, and the bucket flag resolves to the bucket's id
// before the grant or revoke is written.
func TestBucketKeyGrantAndRevokeResolveBothNames(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/buckets":                        {Body: bucketList},
		"GET /v1/bucket-keys":                    {Body: keyList},
		"PUT /v1/bucket-keys/k-1/buckets/b-1":    {Status: 204},
		"DELETE /v1/bucket-keys/k-1/buckets/b-1": {Status: 204},
	})
	h.mustRun("bucket-key", "grant", "ci", "--bucket", "logs-7f3a", "--yes")
	h.mustRun("bucket-key", "revoke", "ci", "--bucket", "logs-7f3a", "--yes")
	want := []string{"PUT /v1/bucket-keys/k-1/buckets/b-1", "DELETE /v1/bucket-keys/k-1/buckets/b-1"}
	if strings.Join(h.fake.WriteURLs, "|") != strings.Join(want, "|") {
		t.Errorf("writes %v", h.fake.WriteURLs)
	}
	if code := h.run("bucket-key", "grant", "ci", "--yes"); code != ExitUsage || len(h.fake.Writes) != 2 {
		t.Errorf("without --bucket: exit %d, writes %d", code, len(h.fake.Writes))
	}
}

// reveal-secret prints the secret when the backend holds one and says so when
// it does not; a delete names the key it removes.
func TestBucketKeyRevealAndDelete(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/bucket-keys":                            {Body: keyList},
		"POST /v1/bucket-keys/k-1/actions/reveal-secret": {Body: `{"keyId":"k-1","available":true,"secretKey":"s3cr3t"}`},
		"POST /v1/bucket-keys/k-0/actions/reveal-secret": {Body: `{"keyId":"k-0","available":false}`},
		"DELETE /v1/bucket-keys/k-1":                     {Status: 204},
	})
	h.mustRun("bucket-key", "reveal-secret", "ci", "--yes", "-o", "table")
	contains(t, h.stdout.String(), "s3cr3t", "yes")
	h.mustRun("bucket-key", "reveal-secret", "master", "--yes", "-o", "table")
	contains(t, h.stdout.String(), "AVAILABLE", "no")
	if strings.Contains(h.stdout.String(), "s3cr3t") {
		t.Errorf("an unavailable secret printed one: %s", h.stdout)
	}
	h.mustRun("bucket-key", "delete", "ci", "--yes")
	if last := h.fake.WriteURLs[len(h.fake.WriteURLs)-1]; last != "DELETE /v1/bucket-keys/k-1" {
		t.Errorf("writes %v", h.fake.WriteURLs)
	}
}

// A secret is the command's deliverable: if stdout cannot take it, the command
// exits non-zero, so a script that captures it does not read an empty value.
func TestBucketKeySecretsFailWhenStdoutCannotTakeThem(t *testing.T) {
	for name, args := range map[string][]string{
		"create":        {"bucket-key", "create", "--name", "x", "--access-type", "read", "--yes"},
		"reveal-secret": {"bucket-key", "reveal-secret", "ci", "--yes"},
	} {
		h := s3Harness(t)
		h.failStdout = true
		canned(h, map[string]fakeapi.Canned{
			"GET /v1/bucket-keys":                            {Body: keyList},
			"POST /v1/bucket-keys":                           {Status: 201, Body: `{"key":{"id":"k-2"},"secretKey":"s3cr3t"}`},
			"POST /v1/bucket-keys/k-1/actions/reveal-secret": {Body: `{"keyId":"k-1","available":true,"secretKey":"s3cr3t"}`},
		})
		if code := h.run(args...); code == 0 {
			t.Errorf("%s: exit 0 with stdout failing", name)
		}
	}
}
