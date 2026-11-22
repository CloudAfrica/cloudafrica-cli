package update

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// start runs path with "sleep", so Windows holds the file as a running exe.
func start(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command(path, "sleep")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
}

// Windows refuses to overwrite a running exe; the swap renames it aside, and a
// second update succeeds while that aside is still running and locked.
func TestWindowsSwapWhileRunning(t *testing.T) {
	dir := t.TempDir()
	exe := install(t, dir, "ca.exe", "0.3.1")
	start(t, exe)

	r := newRelease(t, "v0.4.0")
	r.publish(t, "windows", map[string][]byte{"ca.exe": fake["0.4.0"], "cloudafrica.exe": fake["0.4.0"]})
	o := r.options(exe, "windows", "0.3.1")
	o.RetryFor = 2 * time.Second
	if _, err := Run(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	if got := versionOf(t, exe); got != "ca 0.4.0 (fake)" {
		t.Fatalf("after the first update: %q", got)
	}
	first, _ := filepath.Glob(exe + asideSuffix + "*")
	if len(first) != 1 {
		t.Fatalf("asides after the first update: %v", first)
	}

	start(t, exe) // 0.4.0 now runs too
	r2 := newRelease(t, "v0.5.0")
	r2.publish(t, "windows", map[string][]byte{"ca.exe": fake["0.5.0"], "cloudafrica.exe": fake["0.5.0"]})
	o = r2.options(exe, "windows", "0.4.0")
	o.RetryFor = 2 * time.Second
	if _, err := Run(t.Context(), o); err != nil {
		t.Fatalf("second update with a locked aside: %v", err)
	}
	if got := versionOf(t, exe); got != "ca 0.5.0 (fake)" {
		t.Errorf("after the second update: %q", got)
	}
	if !exists(first[0]) {
		t.Error("the running aside was deleted")
	}
}
