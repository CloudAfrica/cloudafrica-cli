package cli

import (
	"strings"
	"testing"
	"time"
)

// The flow ca picks: the code for SSH and for Linux or BSD without a display,
// else the browser on this machine; the flags force either.
func TestLoginFlowSelection(t *testing.T) {
	for _, c := range []struct {
		name   string
		goos   string
		env    map[string]string
		flag   string
		device bool
	}{
		{name: "linux desktop", goos: "linux", env: map[string]string{"DISPLAY": ":0"}},
		{name: "wayland", goos: "linux", env: map[string]string{"WAYLAND_DISPLAY": "wayland-0"}},
		{name: "linux headless", goos: "linux", device: true},
		{name: "freebsd headless", goos: "freebsd", device: true},
		{name: "openbsd headless", goos: "openbsd", device: true},
		{name: "netbsd headless", goos: "netbsd", device: true},
		{name: "dragonfly headless", goos: "dragonfly", device: true},
		{name: "macos", goos: "darwin"},
		{name: "windows", goos: "windows"},
		{name: "ssh connection", goos: "darwin", env: map[string]string{"SSH_CONNECTION": "a b c d"}, device: true},
		{name: "ssh tty with display", goos: "linux",
			env: map[string]string{"SSH_TTY": "/dev/pts/0", "DISPLAY": ":10"}, device: true},
		{name: "--device", goos: "darwin", flag: "--device", device: true},
		{name: "--no-device", goos: "linux", env: map[string]string{"SSH_TTY": "/dev/pts/0"}, flag: "--no-device"},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(t)
			delete(h.env, "DISPLAY")
			for k, v := range c.env {
				h.env[k] = v
			}
			h.goos = c.goos
			args := []string{"auth", "login"}
			if c.flag != "" {
				args = append(args, c.flag)
			}
			h.mustRun(args...)
			device, loopback := len(h.fake.DeviceAuthorizations), len(h.fake.Authorizations)
			if c.device != (device == 1) || c.device == (loopback == 1) {
				t.Errorf("device %d, loopback %d; want device=%v", device, loopback, c.device)
			}
		})
	}
}

func TestDeviceLoginStoresTheProfileWithoutABrowser(t *testing.T) {
	h := newHarness(t)
	opened := 0
	h.browser = func(string) error { opened++; return nil }
	h.mustRun("auth", "login", "--device")
	p, err := h.store.Load("default")
	if err != nil || p.RefreshToken == "" || p.AccountID != "acc-1" {
		t.Fatalf("profile = %+v, %v", p, err)
	}
	if got := h.fake.DeviceAuthorizations[0].Get("scope"); got != strings.Join(defaultScopes, " ") {
		t.Errorf("scope = %q", got)
	}
	if opened != 0 || h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), "enter BCDF-GHJK") {
		t.Errorf("opened %d, stdout %q, stderr %s", opened, h.stdout, h.stderr)
	}
}

func TestDeviceLoginDeniedStoresNothing(t *testing.T) {
	h := newHarness(t)
	h.fake.DeviceAnswers = []string{"access_denied"}
	if code := h.run("auth", "login", "--device", "--profile", "globex"); code != ExitAuth {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if s := h.stderr.String(); !strings.Contains(s, "sign-in was denied") ||
		!strings.Contains(s, "`ca auth login --profile globex`") {
		t.Errorf("stderr = %s", s)
	}
	if names, _ := h.store.Names(); len(names) != 0 {
		t.Errorf("profiles = %v", names)
	}
}

// A denial on a profile's own account points at the other accounts, as on the
// loopback flow.
func TestDeviceLoginDeniedForTheProfilesAccount(t *testing.T) {
	h := signedIn(t)
	h.fake.DeviceAnswers = []string{"access_denied"}
	if code := h.run("auth", "login", "--device"); code != ExitAuth {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if s := h.stderr.String(); !strings.Contains(s, "declined for account acc-1") {
		t.Errorf("stderr = %s", s)
	}
}

func TestLoginWithoutAServerDeviceFlow(t *testing.T) {
	h := newHarness(t)
	h.fake.NoDeviceFlow = true
	code := h.run("auth", "login", "--device", "--profile", "globex")
	if code == ExitOK || len(h.fake.Authorizations) != 0 {
		t.Fatalf("--device: exit %d, stderr %s", code, h.stderr)
	}
	if s := h.stderr.String(); !strings.Contains(s, "no device sign-in") ||
		!strings.Contains(s, "`ca auth login --no-device --profile globex`") {
		t.Errorf("stderr = %s", s)
	}
	delete(h.env, "DISPLAY")
	h.mustRun("auth", "login")
	if len(h.fake.Authorizations) != 1 || !strings.Contains(h.stderr.String(), "Note: ") {
		t.Errorf("no fallback with a note: %s", h.stderr)
	}
}

// The sign-in offered after an insufficient-scope refusal picks its flow as
// `ca auth login` does.
func TestStepUpSignInOverSSHUsesTheDeviceFlow(t *testing.T) {
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	h.env["SSH_TTY"] = "/dev/pts/0"
	h.fake.PairOnPoll = 2
	h.tty = true
	h.stdin = "y\ny\n"
	h.mustRun("vm", "delete", "vm-1", "--yes")
	if len(h.fake.DeviceAuthorizations) != 1 || len(h.fake.Authorizations) != 0 {
		t.Fatalf("device %d, loopback %d", len(h.fake.DeviceAuthorizations), len(h.fake.Authorizations))
	}
	if got := h.fake.DeviceAuthorizations[0].Get("scope"); !strings.Contains(got, "protected:compute") {
		t.Errorf("scope = %q", got)
	}
}

func TestLoginRefusesBothFlowFlags(t *testing.T) {
	h := newHarness(t)
	if code := h.run("auth", "login", "--device", "--no-device"); code != ExitUsage ||
		len(h.fake.Authorizations)+len(h.fake.DeviceAuthorizations) != 0 {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
}
