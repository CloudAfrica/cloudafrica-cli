package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CloudAfrica/cloudafrica-cli/internal/update"
)

// cellar is an executable path inside a Homebrew Cellar.
func cellar(t *testing.T) func() (string, error) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "Cellar", "ca", "0.1.0", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "ca")
	if err := os.WriteFile(exe, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	return func() (string, error) { return exe, nil }
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
	h.executable = cellar(t)
	h.mustRun("update")
	if got := h.stdout.String(); !strings.Contains(got, "run 'brew upgrade ca'") {
		t.Errorf("stdout %q", got)
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
