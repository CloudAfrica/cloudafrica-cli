package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CloudAfrica/cloudafrica-cli/internal/update"
)

// cellar is Homebrew's layout: bin/ca links to a file in the Cellar, and
// os.Executable reports the link.
func cellar(t *testing.T) (find func() (string, error), link string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Homebrew does not run on Windows")
	}
	prefix := t.TempDir()
	dir := filepath.Join(prefix, "Cellar", "ca", "0.1.0", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ca"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	link = filepath.Join(prefix, "bin", "ca")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../Cellar/ca/0.1.0/bin/ca", link); err != nil {
		t.Fatal(err)
	}
	return func() (string, error) { return link, nil }, link
}

func TestUpdateCheckReports(t *testing.T) {
	for _, c := range []struct {
		tag, want string
	}{
		{"v0.2.0", "ca 0.2.0 is available (this is 0.1.0)\n"},
		{"v0.1.0", "ca 0.1.0 is the latest release\n"},
		{"v0.0.9", "ca 0.1.0 is newer than the latest release 0.0.9\n"},
	} {
		releases(t, http.StatusOK, c.tag)
		h := newHarness(t)
		h.mustRun("update", "--check")
		if got := h.stdout.String(); got != c.want {
			t.Errorf("%s: %q, want %q", c.tag, got, c.want)
		}
	}
}

func TestUpdateLeavesHomebrewToBrew(t *testing.T) {
	releases(t, http.StatusOK, "v0.2.0")
	h := newHarness(t)
	var link string
	h.executable, link = cellar(t)
	h.mustRun("update")
	if got := h.stdout.String(); !strings.Contains(got, "run 'brew upgrade ca'") {
		t.Errorf("stdout %q", got)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("brew's link was replaced: %v", err)
	}
	h.cacheDir = t.TempDir()
	h.tty = true
	h.mustRun("version")
	if got := h.stderr.String(); !strings.Contains(got, "A newer release is available: v0.2.0. Run 'brew upgrade ca'.") {
		t.Errorf("notice %q", got)
	}
}

func TestVersionNoticeNamesCaUpdate(t *testing.T) {
	releases(t, http.StatusOK, "v0.2.0")
	h := newHarness(t)
	h.executable = func() (string, error) { return filepath.Join(t.TempDir(), "ca"), nil }
	h.cacheDir = t.TempDir()
	h.tty = true
	h.mustRun("version")
	if got := h.stderr.String(); got != "A newer release is available: v0.2.0. Run 'ca update'.\n" {
		t.Errorf("notice %q", got)
	}
	// Installed only as cloudafrica beside another program's ca.
	h.run("version") // builds h.app
	h.app.Exe = "cloudafrica"
	h.stderr.Reset()
	h.app.Run(t.Context(), []string{"version"})
	if got := h.stderr.String(); got != "A newer release is available: v0.2.0. Run 'cloudafrica update'.\n" {
		t.Errorf("notice as cloudafrica %q", got)
	}
}

func TestUpdateExitCodes(t *testing.T) {
	releases(t, http.StatusNotFound, "")
	h := newHarness(t)
	if code := h.run("update"); code != ExitAPI || !strings.Contains(h.stderr.String(), "no release found") {
		t.Errorf("404: exit %d, stderr %q", code, h.stderr)
	}
	h.run("version") // builds h.app
	h.app.Version = "dev"
	if code := h.app.Run(t.Context(), []string{"update", "--check"}); code != ExitUsage ||
		!strings.Contains(h.stderr.String(), "development build") {
		t.Errorf("dev: exit %d, stderr %q", code, h.stderr)
	}
}

func TestUpdateRefusesAnUnresolvableExecutable(t *testing.T) {
	releases(t, http.StatusOK, "v0.2.0")
	h := newHarness(t)
	h.executable = func() (string, error) { return filepath.Join(t.TempDir(), "gone", "ca"), nil }
	if code := h.run("update"); code != ExitAPI || !strings.Contains(h.stderr.String(), "resolving") {
		t.Errorf("exit %d, stderr %q", code, h.stderr)
	}
}

// The pairing secret goes only to the API endpoint, never to GitHub.
func TestUpdateNeverCarriesTheElevationSecret(t *testing.T) {
	var got atomic.Value
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		got.Store(r.Header.Get(elevationHeader) + r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0"}`))
	}))
	t.Cleanup(srv.Close)
	old := update.LatestReleaseURL
	update.LatestReleaseURL = srv.URL
	t.Cleanup(func() { update.LatestReleaseURL = old })

	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	h.env["CLOUDAFRICA_ELEVATION"] = testSecret
	h.mustRun("update", "--check")
	if n.Load() != 1 || got.Load() != "" {
		t.Errorf("%d checks, headers %q", n.Load(), got.Load())
	}
}

// A full update through the command: `cloudafrica update`, run through the
// link install.sh makes, replaces the ca file it points at.
func TestUpdateReplacesTheBinaryThroughTheLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake release is a shell script")
	}
	const fresh = "#!/bin/sh\necho 'ca 0.2.0 (fake)'\n"
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "ca", Mode: 0o755, Size: int64(len(fresh)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write([]byte(fresh))
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("cloudafrica-cli_0.2.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	sum := sha256.Sum256(archive.Bytes())
	var headers atomic.Value
	headers.Store("")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ := headers.Load().(string)
		headers.Store(seen + r.Header.Get(elevationHeader) + r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/latest":
			_, _ = w.Write([]byte(`{"tag_name":"v0.2.0"}`))
		case "/download/v0.2.0/cloudafrica-cli_0.2.0_SHA256SUMS":
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), name)
		case "/download/v0.2.0/" + name:
			_, _ = w.Write(archive.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	oldLatest, oldBase := update.LatestReleaseURL, update.DownloadBase
	update.LatestReleaseURL, update.DownloadBase = srv.URL+"/latest", srv.URL+"/download"
	t.Cleanup(func() { update.LatestReleaseURL, update.DownloadBase = oldLatest, oldBase })

	dir := t.TempDir()
	exe := filepath.Join(dir, "ca")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\necho 'ca 0.1.0 (old)'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "cloudafrica")
	if err := os.Symlink("ca", link); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t)
	h.executable = func() (string, error) { return link, nil }
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	h.env["CLOUDAFRICA_ELEVATION"] = testSecret
	h.mustRun("update")
	if got := h.stdout.String(); got != "Updated ca 0.1.0 -> 0.2.0\n" {
		t.Errorf("stdout %q", got)
	}
	if got, _ := os.ReadFile(exe); string(got) != fresh {
		t.Errorf("ca holds %q", got)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the link was replaced: %v", err)
	}
	if got := headers.Load(); got != "" {
		t.Errorf("credentials sent to GitHub: %q", got)
	}
}
