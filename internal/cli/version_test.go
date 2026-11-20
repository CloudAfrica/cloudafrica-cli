package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// releases serves the GitHub latest-release document and counts requests.
func releases(t *testing.T, status int, tag string) *atomic.Int32 {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	}))
	t.Cleanup(srv.Close)
	old := LatestReleaseURL
	LatestReleaseURL = srv.URL
	t.Cleanup(func() { LatestReleaseURL = old })
	return &n
}

func TestVersionNoticeOnTerminalOnceADay(t *testing.T) {
	n := releases(t, http.StatusOK, "v0.2.0")
	h := newHarness(t)
	h.cacheDir = t.TempDir()
	h.tty = true
	h.mustRun("version")
	if !strings.Contains(h.stderr.String(), "A newer release is available: v0.2.0") {
		t.Errorf("no notice: %s", h.stderr)
	}
	if _, err := os.Stat(filepath.Join(h.cacheDir, "update-check.json")); err != nil {
		t.Errorf("cache not written: %v", err)
	}
	h.mustRun("version")
	if n.Load() != 1 || !strings.Contains(h.stderr.String(), "v0.2.0") {
		t.Errorf("second run: %d requests, stderr %s", n.Load(), h.stderr)
	}
	// The cache is not a profile.
	h.mustRun("auth", "list", "-o", "json")
	if got := strings.TrimSpace(h.stdout.String()); got != "[]" {
		t.Errorf("auth list = %s", got)
	}
}

func TestVersionNoCheckOffTerminalOrOnDevBuildOrWithoutCache(t *testing.T) {
	n := releases(t, http.StatusOK, "v0.2.0")
	h := newHarness(t)
	h.cacheDir = t.TempDir()
	h.mustRun("version")
	if n.Load() != 0 {
		t.Fatalf("off a terminal: %d requests", n.Load())
	}
	h.tty = true
	h.cacheDir = ""
	h.mustRun("version")
	if n.Load() != 0 {
		t.Fatalf("without a cache dir: %d requests", n.Load())
	}
	h.cacheDir = t.TempDir()
	h.mustRun("version")
	h.app.Version = "dev"
	h.app.CacheDir = t.TempDir()
	if code := h.app.Run(t.Context(), []string{"version"}); code != ExitOK {
		t.Fatalf("dev: exit %d", code)
	}
	if n.Load() != 1 {
		t.Errorf("requests = %d, want 1 (the cached TTY run only)", n.Load())
	}
}

func TestVersionGitHubFailureIsSilentAndCached(t *testing.T) {
	n := releases(t, http.StatusInternalServerError, "")
	h := newHarness(t)
	h.cacheDir = t.TempDir()
	h.tty = true
	h.mustRun("version")
	h.mustRun("version")
	if h.stderr.Len() != 0 || n.Load() != 1 {
		t.Errorf("stderr %q, requests %d", h.stderr, n.Load())
	}
}
