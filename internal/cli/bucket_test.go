package cli

import (
	"fmt"
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

// table runs a command in table form and returns its header line and rows,
// each split on whitespace (a blank cell drops out of its row).
func table(h *harness, args ...string) (string, string) {
	h.t.Helper()
	h.mustRun(append(args, "-o", "table")...)
	lines := strings.Split(strings.TrimSpace(h.stdout.String()), "\n")
	var rows [][]string
	for _, l := range lines[1:] {
		rows = append(rows, strings.Fields(l))
	}
	return strings.Join(strings.Fields(lines[0]), " "), fmt.Sprint(rows)
}

// The bucket table shows the billed quota and the on-disk size in bytes units, -q
// prints ids, and a bucket named on the command line resolves to its id.
func TestBucketListGetAndNameResolution(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/buckets":     {Body: bucketList},
		"GET /v1/buckets/b-1": {Body: `{"id":"b-1","name":"logs-7f3a","status":"active","objectCount":42}`},
	})
	head, rows := table(h, "bucket", "list")
	if head != "ID NAME DISPLAY NAME STATUS BILLED ON DISK OBJECTS CREATED" {
		t.Errorf("header %q", head)
	}
	if want := "[[b-1 logs-7f3a Logs active 2.0 GiB 1.0 GiB 42 2026-11-01"; !strings.HasPrefix(rows, want) {
		t.Errorf("rows %s, want prefix %s", rows, want)
	}
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
	head, rows := table(h, "bucket", "versioning", "get", "logs-7f3a")
	if head != "MODE" || rows != "[[suspended]]" {
		t.Errorf("versioning table %q %s", head, rows)
	}
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
		"GET /v1/buckets/b-1/usage": {Body: `{"bucketName":"logs-7f3a","sizeBytes":2048,"sizeActualBytes":4096,
			"objectCount":3,"windowHours":168,"bytesUploaded":1048576,"bytesDownloaded":3145728,"operations":9}`},
	})
	head, rows := table(h, "bucket", "usage", "logs-7f3a", "--hours", "168")
	if head != "BUCKET SIZE ON DISK OBJECTS WINDOW H UPLOADED DOWNLOADED REQUESTS" {
		t.Errorf("header %q", head)
	}
	if rows != "[[logs-7f3a 2.0 KiB 4.0 KiB 3 168 1.0 MiB 3.0 MiB 9]]" {
		t.Errorf("rows %s", rows)
	}
	if !slices.Contains(h.fake.ReadURLs, "GET /v1/buckets/b-1/usage?hours=168") {
		t.Errorf("reads %v", h.fake.ReadURLs)
	}
}

// The key list marks the master key, which is attached to no bucket.
func TestBucketKeyList(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{"GET /v1/bucket-keys": {Body: keyList}})
	head, rows := table(h, "bucket-key", "list")
	if head != "ID NAME ACCESS KEY ACCESS MASTER BUCKET" {
		t.Errorf("header %q", head)
	}
	if rows != "[[k-0 master AKM full yes] [k-1 ci AKCI read no logs-7f3a]]" {
		t.Errorf("rows %s", rows)
	}
	h.mustRun("bucket-key", "list", "-q")
	if got := strings.Fields(h.stdout.String()); len(got) != 2 || got[0] != "k-0" || got[1] != "k-1" {
		t.Errorf("-q = %v", got)
	}
}

// A key create prints the secret the response carries (once); -q prints that
// secret, JSON is the body verbatim, and a replay without one still prints the
// key but exits non-zero with the remedy on stderr.
func TestBucketKeyCreatePrintsSecret(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{"POST /v1/bucket-keys": {Status: 201, Body: `{"key":{"id":"k-2","name":"ci2",
		"accessKey":"AKCI2","accessType":"full","bucketName":"logs-7f3a"},"secretKey":"s3cr3t"}`}})
	head, rows := table(h, "bucket-key", "create", "--name", "ci2", "--access-type", "full", "--yes")
	if head != "ID NAME ACCESS KEY ACCESS BUCKET SECRET KEY" || rows != "[[k-2 ci2 AKCI2 full logs-7f3a s3cr3t]]" {
		t.Errorf("create table %q %s", head, rows)
	}
	if b := h.fake.Bodies[0]; b["name"] != "ci2" || b["accessType"] != "full" || len(b) != 2 {
		t.Errorf("body = %v", b)
	}
	if len(h.fake.IdemKeys) != 1 || h.fake.IdemKeys[0] == "" {
		t.Errorf("idempotency keys = %q", h.fake.IdemKeys)
	}
	h.mustRun("bucket-key", "create", "--name", "ci2", "--access-type", "full", "--yes", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "s3cr3t" {
		t.Errorf("-q = %q", got)
	}
	h.mustRun("bucket-key", "create", "--name", "ci2", "--access-type", "full", "--yes", "-o", "json")
	contains(t, h.stdout.String(), `"secretKey": "s3cr3t"`)
	h.mustRun("bucket-key", "create", "--name", "ci2", "--access-type", "full", "--bucket-resource-id", "b-1", "--yes")
	if b := h.fake.Bodies[len(h.fake.Bodies)-1]; b["bucketResourceId"] != "b-1" || len(b) != 3 {
		t.Errorf("body = %v", b)
	}
	canned(h, map[string]fakeapi.Canned{"POST /v1/bucket-keys": {Status: 201, Body: `{"key":{"id":"k-2","name":"ci2",
		"accessKey":"AKCI2","accessType":"full"}}`}})
	if code := h.run("bucket-key", "create", "--name", "ci2", "--access-type", "full", "--yes", "-o", "table"); code == 0 {
		t.Error("a replay without a secret exited 0")
	}
	contains(t, h.stdout.String(), "AKCI2")
	contains(t, h.stderr.String(), "no secret key", "ca bucket-key reveal-secret ci2")
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

// reveal-secret prints the secret (-q prints only it); when the backend holds
// none, or the response omits it, it prints the row, exits non-zero and names
// the remedy. A delete names the key it removes.
func TestBucketKeyRevealAndDelete(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/bucket-keys":                            {Body: keyList},
		"POST /v1/bucket-keys/k-1/actions/reveal-secret": {Body: `{"keyId":"k-1","available":true,"secretKey":"s3cr3t"}`},
		"POST /v1/bucket-keys/k-0/actions/reveal-secret": {Body: `{"keyId":"k-0","available":false}`},
		"DELETE /v1/bucket-keys/k-1":                     {Status: 204},
	})
	head, rows := table(h, "bucket-key", "reveal-secret", "ci", "--yes")
	if head != "KEY ID AVAILABLE SECRET KEY" || rows != "[[k-1 yes s3cr3t]]" {
		t.Errorf("reveal table %q %s", head, rows)
	}
	h.mustRun("bucket-key", "reveal-secret", "ci", "--yes", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "s3cr3t" {
		t.Errorf("-q = %q", got)
	}
	if code := h.run("bucket-key", "reveal-secret", "master", "--yes", "-o", "table"); code == 0 {
		t.Error("an unavailable secret exited 0")
	}
	if got := strings.Join(strings.Fields(h.stdout.String()), " "); got != "KEY ID AVAILABLE SECRET KEY k-0 no" {
		t.Errorf("unavailable row %q", got)
	}
	contains(t, h.stderr.String(), "holds no secret", "ca bucket-key delete master")
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/bucket-keys":                            {Body: keyList},
		"POST /v1/bucket-keys/k-1/actions/reveal-secret": {Body: `{"keyId":"k-1","available":true}`},
		"DELETE /v1/bucket-keys/k-1":                     {Status: 204},
	})
	if code := h.run("bucket-key", "reveal-secret", "ci", "--yes", "-q"); code == 0 || strings.TrimSpace(h.stdout.String()) != "" {
		t.Errorf("an available key with no secret: exit %d, stdout %q", code, h.stdout)
	}
	contains(t, h.stderr.String(), "no secret key", "ca bucket-key reveal-secret ci")
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
