package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fake holds tiny binaries that answer `version` like ca does, keyed by the
// version they print; "broken" exits 1 with a stderr line, "slow" never
// answers, and "foreign" prints another program's
// line. Run with "sleep" they block, standing in for a ca still running.
var fake = map[string][]byte{}

const fakeSource = `package main

import (
	"fmt"
	"os"
	"time"
)

var v string

func main() {
	if len(os.Args) > 1 && os.Args[1] == "sleep" {
		time.Sleep(time.Minute)
	}
	switch v {
	case "slow":
		time.Sleep(time.Minute)
	case "env":
		fmt.Printf("%s%s|%t\n", os.Getenv("CLOUDAFRICA_TOKEN"), os.Getenv("CLOUDAFRICA_ELEVATION"), os.Getenv("PATH") != "")
	case "broken":
		fmt.Fprintln(os.Stderr, "fake ca is broken")
		os.Exit(1)
	case "foreign":
		fmt.Println("foreign 1.0")
	default:
		fmt.Printf("ca %s (fake)\n", v)
	}
}
`

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ca-update-fake-")
	if err != nil {
		panic(err)
	}
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(fakeSource), 0o600); err != nil {
		panic(err)
	}
	for _, v := range []string{"0.3.1", "0.4.0", "0.5.0", "broken", "foreign", "slow", "env"} {
		out := filepath.Join(dir, v)
		cmd := exec.Command("go", "build", "-o", out, "-ldflags", "-X main.v="+v, src)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOFLAGS=")
		if b, err := cmd.CombinedOutput(); err != nil {
			panic(fmt.Sprintf("building the fake ca %s: %v\n%s", v, err, b))
		}
		if fake[v], err = os.ReadFile(out); err != nil {
			panic(err)
		}
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// release serves the GitHub latest-release document and one release's
// assets, and counts the asset downloads.
type release struct {
	srv       *httptest.Server
	tag       string
	status    int
	delay     time.Duration
	stall     time.Duration // an archive download sends half, then waits this long
	assets    map[string][]byte
	downloads atomic.Int32
}

func newRelease(t *testing.T, tag string) *release {
	t.Helper()
	r := &release{tag: tag, status: http.StatusOK, assets: map[string][]byte{}}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/latest" {
			time.Sleep(r.delay)
			w.WriteHeader(r.status)
			fmt.Fprintf(w, `{"tag_name":%q}`, r.tag)
			return
		}
		r.downloads.Add(1)
		b, ok := r.assets[strings.TrimPrefix(req.URL.Path, "/download/"+r.tag+"/")]
		if !ok {
			http.NotFound(w, req)
			return
		}
		if r.stall > 0 && !strings.HasSuffix(req.URL.Path, "SHA256SUMS") {
			_, _ = w.Write(b[:len(b)/2])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			select {
			case <-time.After(r.stall):
			case <-req.Context().Done():
			}
			return
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(r.srv.Close)
	return r
}

// publish adds the archive for goos holding members, and a SHA256SUMS that
// lists it.
func (r *release) publish(t *testing.T, goos string, members map[string][]byte) (archive string) {
	t.Helper()
	version := strings.TrimPrefix(r.tag, "v")
	var b []byte
	if goos == "windows" {
		archive = fmt.Sprintf("cloudafrica-cli_%s_%s_%s.zip", version, goos, runtime.GOARCH)
		b = zipOf(t, members)
	} else {
		archive = fmt.Sprintf("cloudafrica-cli_%s_%s_%s.tar.gz", version, goos, runtime.GOARCH)
		b = tarOf(t, members)
	}
	r.assets[archive] = b
	sum := sha256.Sum256(b)
	r.assets[fmt.Sprintf("cloudafrica-cli_%s_SHA256SUMS", version)] =
		[]byte(fmt.Sprintf("%s  %s\n%s  cloudafrica-cli_%s_other.tar.gz\n", hex.EncodeToString(sum[:]), archive, strings.Repeat("0", 64), version))
	return archive
}

func tarOf(t *testing.T, members map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, b := range map[string][]byte{"LICENSE": []byte("license"), "README.md": []byte("readme")} {
		if _, ok := members[name]; !ok {
			members[name] = b
		}
	}
	for name, b := range members {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(b)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(b); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipOf(t *testing.T, members map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, b := range members {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(b); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// install writes the fake ca of version v at dir/name.
func install(t *testing.T, dir, name, v string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, fake[v], 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func (r *release) options(exe, goos, current string) Options {
	return Options{
		Exe:          exe,
		GOOS:         goos,
		GOARCH:       runtime.GOARCH,
		Current:      current,
		APIURL:       r.srv.URL + "/latest",
		DownloadBase: r.srv.URL + "/download",
		Client:       r.srv.Client(),
		UserAgent:    "ca/test",
		RetryFor:     time.Millisecond,
	}
}

func versionOf(t *testing.T, path string) string {
	t.Helper()
	line, err := probe(t.Context(), path, 10*time.Second)
	if err != nil {
		t.Fatalf("running %s: %v", path, err)
	}
	return line
}

// unixName is the binary name on the platform the tests run on, which is
// where the swapped binary must actually run.
func unixName() (goos, name string) {
	if runtime.GOOS == "windows" {
		return "windows", "ca.exe"
	}
	return runtime.GOOS, "ca"
}

func TestUpdatesTheBinary(t *testing.T) {
	goos, name := unixName()
	r := newRelease(t, "v0.4.0")
	members := map[string][]byte{name: fake["0.4.0"]}
	if goos == "windows" {
		members["cloudafrica.exe"] = fake["0.4.0"]
	}
	r.publish(t, goos, members)
	dir := t.TempDir()
	exe := install(t, dir, name, "0.3.1")
	if goos != "windows" {
		if err := os.Symlink(name, filepath.Join(dir, "cloudafrica")); err != nil {
			t.Fatal(err)
		}
	}

	res, err := Run(t.Context(), r.options(exe, goos, "0.3.1"))
	if err != nil {
		t.Fatal(err)
	}
	if res.State != Updated || res.Current != "0.3.1" || res.Latest != "0.4.0" {
		t.Errorf("result = %+v", res)
	}
	if got := versionOf(t, exe); got != "ca 0.4.0 (fake)" {
		t.Errorf("after update: %q", got)
	}
	if goos != "windows" {
		if got := versionOf(t, filepath.Join(dir, "cloudafrica")); got != "ca 0.4.0 (fake)" {
			t.Errorf("through the link: %q", got)
		}
	}
	if left, _ := filepath.Glob(filepath.Join(dir, tempPrefix+"*")); len(left) != 0 {
		t.Errorf("temp left behind: %v", left)
	}
}

func TestUpToDateAheadAndCheckDownloadNothing(t *testing.T) {
	goos, name := unixName()
	r := newRelease(t, "v0.4.0")
	r.publish(t, goos, map[string][]byte{name: fake["0.4.0"]})
	dir := t.TempDir()
	exe := install(t, dir, name, "0.3.1")

	for _, c := range []struct {
		current string
		check   bool
		want    State
	}{
		{"0.4.0", false, UpToDate},
		{"0.5.0-rc.1", false, Ahead},
		{"0.3.1", true, Available},
		{"0.4.0", true, UpToDate},
	} {
		o := r.options(exe, goos, c.current)
		o.Check = c.check
		res, err := Run(t.Context(), o)
		if err != nil || res.State != c.want {
			t.Errorf("%s check=%v: %+v, %v", c.current, c.check, res, err)
		}
	}
	if n := r.downloads.Load(); n != 0 {
		t.Errorf("%d downloads", n)
	}
	if got := versionOf(t, exe); got != "ca 0.3.1 (fake)" {
		t.Errorf("binary changed: %q", got)
	}
}

func TestNoRelease(t *testing.T) {
	for _, c := range []struct {
		status int
		tag    string
	}{{http.StatusNotFound, ""}, {http.StatusOK, "latest"}, {http.StatusOK, ""}} {
		r := newRelease(t, c.tag)
		r.status = c.status
		_, err := Run(t.Context(), r.options(filepath.Join(t.TempDir(), "ca"), "linux", "0.3.1"))
		if !errors.Is(err, ErrNoRelease) {
			t.Errorf("%d %q: %v", c.status, c.tag, err)
		}
	}
}

func TestOtherGitHubFailureIsNotNoRelease(t *testing.T) {
	r := newRelease(t, "")
	r.status = http.StatusInternalServerError
	res, err := Run(t.Context(), r.options(filepath.Join(t.TempDir(), "ca"), "linux", "0.3.1"))
	if err == nil || errors.Is(err, ErrNoRelease) || res.State == UpToDate {
		t.Errorf("%+v, %v", res, err)
	}
}

func TestTimeoutIsDeadlineExceeded(t *testing.T) {
	r := newRelease(t, "v0.4.0")
	r.delay = 200 * time.Millisecond
	o := r.options(filepath.Join(t.TempDir(), "ca"), "linux", "0.3.1")
	o.LatestTimeout = 20 * time.Millisecond
	_, err := Run(t.Context(), o)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v", err)
	}
}

func TestStalledDownloadTimesOutAndReplacesNothing(t *testing.T) {
	goos, name := unixName()
	r := newRelease(t, "v0.4.0")
	r.publish(t, goos, map[string][]byte{name: fake["0.4.0"], "cloudafrica.exe": fake["0.4.0"]})
	r.stall = 5 * time.Second
	dir := t.TempDir()
	exe := install(t, dir, name, "0.3.1")
	o := r.options(exe, goos, "0.3.1")
	o.DownloadTimeout = 100 * time.Millisecond
	res, err := Run(t.Context(), o)
	if !errors.Is(err, context.DeadlineExceeded) || res.State == UpToDate {
		t.Errorf("%+v, %v", res, err)
	}
	if got, _ := os.ReadFile(exe); !bytes.Equal(got, fake["0.3.1"]) {
		t.Error("the installed binary changed")
	}
	if left, _ := filepath.Glob(filepath.Join(dir, tempPrefix+"*")); len(left) != 0 {
		t.Errorf("temp left behind: %v", left)
	}
}

// A directory that cannot hold a temp dir for a reason sudo does not fix is
// an error, not a refusal that suggests sudo.
func TestNonPermissionWriteFailureIsNotARefusal(t *testing.T) {
	r := newRelease(t, "v0.4.0")
	r.publish(t, "linux", map[string][]byte{"ca": fake["0.4.0"]})
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Run(t.Context(), r.options(filepath.Join(file, "ca"), "linux", "0.3.1"))
	var refused *RefusedError
	if err == nil || errors.As(err, &refused) || !strings.Contains(err.Error(), "creating a temporary directory") {
		t.Errorf("err = %v", err)
	}
}

func TestRefusesADevBuildEvenForCheck(t *testing.T) {
	r := newRelease(t, "v0.4.0")
	for _, check := range []bool{false, true} {
		o := r.options(filepath.Join(t.TempDir(), "ca"), "linux", "dev")
		o.Check = check
		_, err := Run(t.Context(), o)
		var refused *RefusedError
		if !errors.As(err, &refused) || !strings.Contains(err.Error(), "development build") {
			t.Errorf("check=%v: %v", check, err)
		}
	}
}

func TestBadDownloadsReplaceNothing(t *testing.T) {
	goos, name := unixName()
	members := func(v string) map[string][]byte {
		m := map[string][]byte{name: fake[v]}
		if goos == "windows" {
			m["cloudafrica.exe"] = fake[v]
		}
		return m
	}
	for _, c := range []struct {
		name  string
		setup func(t *testing.T, r *release)
		want  string
	}{
		{"checksum mismatch", func(t *testing.T, r *release) {
			archive := r.publish(t, goos, members("0.4.0"))
			r.assets[archive] = append(r.assets[archive], 0)
		}, "checksum mismatch"},
		{"not listed", func(t *testing.T, r *release) {
			r.publish(t, goos, members("0.4.0"))
			r.assets["cloudafrica-cli_0.4.0_SHA256SUMS"] = []byte(strings.Repeat("0", 64) + "  something-else.tar.gz\n")
		}, "not listed in cloudafrica-cli_0.4.0_SHA256SUMS"},
		{"broken binary", func(t *testing.T, r *release) {
			r.publish(t, goos, members("broken"))
		}, "the downloaded ca does not run (exit status 1: fake ca is broken); nothing was replaced"},
		{"wrong version", func(t *testing.T, r *release) {
			r.publish(t, goos, members("0.5.0"))
		}, `the downloaded ca reports "ca 0.5.0 (fake)", not version 0.4.0; nothing was replaced`},
		{"missing binary", func(t *testing.T, r *release) {
			r.publish(t, goos, map[string][]byte{"other": fake["0.4.0"]})
		}, "does not hold"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRelease(t, "v0.4.0")
			c.setup(t, r)
			dir := t.TempDir()
			exe := install(t, dir, name, "0.3.1")
			_, err := Run(t.Context(), r.options(exe, goos, "0.3.1"))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want %q", err, c.want)
			}
			if got, _ := os.ReadFile(exe); !bytes.Equal(got, fake["0.3.1"]) {
				t.Error("the installed binary changed")
			}
			if left, _ := filepath.Glob(filepath.Join(dir, tempPrefix+"*")); len(left) != 0 {
				t.Errorf("temp left behind: %v", left)
			}
		})
	}
}

func TestArchiveMemberWithAPathIsRefused(t *testing.T) {
	r := newRelease(t, "v0.4.0")
	r.publish(t, "linux", map[string][]byte{"ca": fake["0.4.0"], "../ca": fake["0.4.0"]})
	exe := install(t, t.TempDir(), "ca", "0.3.1")
	_, err := Run(t.Context(), r.options(exe, "linux", "0.3.1"))
	if err == nil || !strings.Contains(err.Error(), `unexpected archive member "../ca"`) {
		t.Errorf("err = %v", err)
	}
}

func TestSweepsOnlyStaleTempDirs(t *testing.T) {
	goos, name := unixName()
	r := newRelease(t, "v0.4.0")
	members := map[string][]byte{name: fake["0.4.0"]}
	if goos == "windows" {
		members["cloudafrica.exe"] = fake["0.4.0"]
	}
	r.publish(t, goos, members)
	dir := t.TempDir()
	exe := install(t, dir, name, "0.3.1")
	stale, fresh := filepath.Join(dir, tempPrefix+"stale"), filepath.Join(dir, tempPrefix+"fresh")
	for _, d := range []string{stale, fresh} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(t.Context(), r.options(exe, goos, "0.3.1")); err != nil {
		t.Fatal(err)
	}
	if exists(stale) || !exists(fresh) {
		t.Errorf("stale kept %v, fresh kept %v", exists(stale), exists(fresh))
	}
}

func TestUnwritableDirectoryIsRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a read-only attribute does not stop creating files on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root writes anywhere")
	}
	r := newRelease(t, "v0.4.0")
	r.publish(t, "linux", map[string][]byte{"ca": fake["0.4.0"]})
	dir := t.TempDir()
	exe := install(t, dir, "ca", "0.3.1")
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	o := r.options(exe, runtime.GOOS, "0.3.1")
	_, err := Run(t.Context(), o)
	var refused *RefusedError
	if !errors.As(err, &refused) || !strings.Contains(err.Error(), dir) || !strings.Contains(err.Error(), "sudo "+exe+" update") {
		t.Errorf("err = %v", err)
	}
	if r.downloads.Load() != 0 {
		t.Error("downloaded before refusing")
	}
	o.Check = true
	if res, err := Run(t.Context(), o); err != nil || res.State != Available {
		t.Errorf("--check: %+v, %v", res, err)
	}
}

func TestPackageManagerInstallsAreLeftToThem(t *testing.T) {
	r := newRelease(t, "v0.4.0")
	cellar := filepath.Join(t.TempDir(), "Cellar", "ca", "0.3.1", "bin")
	if err := os.MkdirAll(cellar, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := install(t, cellar, "ca", "0.3.1")
	res, err := Run(t.Context(), r.options(exe, "darwin", "0.3.1"))
	if err != nil || res.State != Available || res.Managed != "brew upgrade ca" {
		t.Errorf("%+v, %v", res, err)
	}
	if r.downloads.Load() != 0 {
		t.Error("downloaded")
	}
}

func TestManaged(t *testing.T) {
	scoop := t.TempDir()
	for _, f := range []string{"scoop-install.json", "scoop-manifest.json"} {
		if err := os.WriteFile(filepath.Join(scoop, f), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	oldScoop := t.TempDir()
	for _, f := range []string{"install.json", "manifest.json"} {
		if err := os.WriteFile(filepath.Join(oldScoop, f), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	halfScoop := t.TempDir()
	if err := os.WriteFile(filepath.Join(halfScoop, "install.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"ProgramData": `C:\ProgramData`, "SCOOP_GLOBAL": `D:\GlobalApps\`}
	getenv := func(k string) string { return env[k] }

	for _, c := range []struct {
		exe, goos, want string
	}{
		{"/opt/homebrew/Cellar/ca/0.3.1/bin/ca", "darwin", "brew upgrade ca"},
		{"/home/linuxbrew/.linuxbrew/Cellar/ca/0.3.1/bin/ca", "linux", "brew upgrade ca"},
		{"/usr/local/bin/ca", "linux", ""},
		{`C:\Users\me\scoop\apps\ca\current\ca.exe`, "windows", "scoop update ca"},
		{`C:\Users\me\SCOOP\Apps\ca\current\ca.exe`, "windows", "scoop update ca"},
		{`C:\ProgramData\scoop\apps\ca\current\ca.exe`, "windows", "scoop update -g ca"},
		{`D:\GlobalApps\apps\ca\current\ca.exe`, "windows", ""},
		{filepath.Join(scoop, "ca.exe"), "windows", "scoop update ca"},
		{filepath.Join(oldScoop, "ca.exe"), "windows", "scoop update ca"},
		{filepath.Join(halfScoop, "ca.exe"), "windows", ""},
		{filepath.Join(scoop, "ca"), "linux", ""},
		{`C:\Users\me\AppData\Local\Programs\cloudafrica\ca.exe`, "windows", ""},
	} {
		if got := Managed(c.exe, c.goos, getenv); got != c.want {
			t.Errorf("Managed(%q, %s) = %q, want %q", c.exe, c.goos, got, c.want)
		}
	}
	// A global root found by its files.
	global := filepath.Join(t.TempDir(), "apps", "ca", "current")
	if err := os.MkdirAll(global, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"scoop-install.json", "scoop-manifest.json"} {
		if err := os.WriteFile(filepath.Join(global, f), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	env["SCOOP_GLOBAL"] = filepath.Dir(filepath.Dir(filepath.Dir(global)))
	if got := Managed(filepath.Join(global, "ca.exe"), "windows", getenv); got != "scoop update -g ca" {
		t.Errorf("global by files = %q", got)
	}
}

// The Windows swap, run on any OS: renames behave the same apart from the
// lock, which TestWindowsSwapWhileRunning covers on Windows.
func TestWindowsSwapReplacesOurSiblingOnly(t *testing.T) {
	for _, c := range []struct {
		name, sibling, want string
	}{
		{"ours", "0.3.1", "ca 0.4.0 (fake)"},
		{"foreign", "foreign", "foreign 1.0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRelease(t, "v0.4.0")
			r.publish(t, "windows", map[string][]byte{"ca.exe": fake["0.4.0"], "cloudafrica.exe": fake["0.4.0"]})
			dir := t.TempDir()
			exe := install(t, dir, "cloudafrica.exe", "0.3.1")
			sib := install(t, dir, "ca.exe", c.sibling)
			res, err := Run(t.Context(), r.options(exe, "windows", "0.3.1"))
			if err != nil || res.State != Updated || res.SiblingErr != nil {
				t.Fatalf("%+v, %v", res, err)
			}
			if got := versionOf(t, exe); got != "ca 0.4.0 (fake)" {
				t.Errorf("running exe: %q", got)
			}
			if got := versionOf(t, sib); got != c.want {
				t.Errorf("sibling: %q, want %q", got, c.want)
			}
		})
	}
}

// A sibling that cannot be started might be ours, so it is kept and
// reported; one that runs and exits non-zero is foreign, kept quietly.
func TestWindowsSiblingThatWillNotRun(t *testing.T) {
	for _, c := range []struct {
		name    string
		content []byte
		warn    bool
	}{
		{"cannot start", []byte("not an executable"), true},
		{"exits non-zero", fake["broken"], false},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRelease(t, "v0.4.0")
			r.publish(t, "windows", map[string][]byte{"ca.exe": fake["0.4.0"], "cloudafrica.exe": fake["0.4.0"]})
			dir := t.TempDir()
			exe := install(t, dir, "cloudafrica.exe", "0.3.1")
			sib := filepath.Join(dir, "ca.exe")
			if err := os.WriteFile(sib, c.content, 0o755); err != nil {
				t.Fatal(err)
			}
			res, err := Run(t.Context(), r.options(exe, "windows", "0.3.1"))
			if err != nil || res.State != Updated {
				t.Fatalf("%+v, %v", res, err)
			}
			warned := res.SiblingErr != nil &&
				strings.Contains(res.SiblingErr.Error(), sib+" was not updated: running it to check it is ca")
			if warned != c.warn || (!c.warn && res.SiblingErr != nil) {
				t.Errorf("SiblingErr = %v", res.SiblingErr)
			}
			if got, _ := os.ReadFile(sib); !bytes.Equal(got, c.content) {
				t.Error("the sibling changed")
			}
		})
	}
}

// A probed sibling may be another program, so it gets no credentials.
func TestProbeWithholdsCredentials(t *testing.T) {
	t.Setenv("CLOUDAFRICA_TOKEN", "cak_protected")
	t.Setenv("CLOUDAFRICA_ELEVATION", "cae_secret")
	line, err := probe(t.Context(), install(t, t.TempDir(), "env.exe", "env"), 10*time.Second)
	if err != nil || line != "|true" {
		t.Errorf("%q, %v", line, err)
	}
}

func TestShellQuote(t *testing.T) {
	for in, want := range map[string]string{
		"/usr/local/bin/ca": "/usr/local/bin/ca",
		"/opt/my tools/ca":  "'/opt/my tools/ca'",
		"/opt/it's/ca":      `'/opt/it'\''s/ca'`,
	} {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProbeCarriesStderrAndNamesATimeout(t *testing.T) {
	dir := t.TempDir()
	_, err := probe(t.Context(), install(t, dir, "broken.exe", "broken"), 10*time.Second)
	var exited *exec.ExitError
	if !errors.As(err, &exited) || !strings.Contains(err.Error(), "fake ca is broken") {
		t.Errorf("broken: %v", err)
	}
	slow := install(t, dir, "slow.exe", "slow")
	_, err = probe(t.Context(), slow, 50*time.Millisecond)
	if errors.As(err, &exited) || err == nil || !strings.Contains(err.Error(), "did not finish within 50ms") {
		t.Errorf("slow: %v", err)
	}
}

func TestWindowsSweepsEarlierAsides(t *testing.T) {
	r := newRelease(t, "v0.4.0")
	r.publish(t, "windows", map[string][]byte{"ca.exe": fake["0.4.0"], "cloudafrica.exe": fake["0.4.0"]})
	dir := t.TempDir()
	exe := install(t, dir, "ca.exe", "0.3.1")
	old := install(t, dir, "ca.exe"+asideSuffix+"1", "0.3.1")
	if _, err := Run(t.Context(), r.options(exe, "windows", "0.3.1")); err != nil {
		t.Fatal(err)
	}
	if exists(old) {
		t.Error("earlier aside kept")
	}
}

func TestReplaceRunningRestoresWhenTheMoveFails(t *testing.T) {
	dir := t.TempDir()
	dst := install(t, dir, "ca.exe", "0.3.1")
	err := replaceRunning(filepath.Join(dir, "missing"), dst, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "installing") {
		t.Errorf("err = %v", err)
	}
	if got, _ := os.ReadFile(dst); !bytes.Equal(got, fake["0.3.1"]) {
		t.Error("the original is not back in place")
	}
	if left, _ := filepath.Glob(dst + asideSuffix + "*"); len(left) != 0 {
		t.Errorf("aside left: %v", left)
	}
}

func TestSiblingFailureIsReportedNotRolledBack(t *testing.T) {
	dir, tmp := t.TempDir(), t.TempDir()
	exe := install(t, dir, "ca.exe", "0.4.0")
	sib := install(t, dir, "cloudafrica.exe", "0.3.1")
	o := Options{Exe: exe, GOOS: "windows", RetryFor: time.Millisecond}.withDefaults()
	err := replaceSibling(t.Context(), o, tmp) // tmp holds no cloudafrica.exe
	if err == nil || !strings.Contains(err.Error(), sib+" was not updated") {
		t.Errorf("err = %v", err)
	}
	if got := versionOf(t, sib); got != "ca 0.3.1 (fake)" {
		t.Errorf("sibling: %q", got)
	}
}

func TestRetryStopsAtItsDeadline(t *testing.T) {
	var n int
	start := time.Now()
	err := retry(120*time.Millisecond, func() error { n++; return errors.New("locked") })
	if err == nil || n < 2 || time.Since(start) > time.Second {
		t.Errorf("err %v after %d tries in %s", err, n, time.Since(start))
	}
	n = 0
	if err := retry(time.Second, func() error { n++; return nil }); err != nil || n != 1 {
		t.Errorf("success: %v after %d tries", err, n)
	}
}

func TestDigestOf(t *testing.T) {
	sums := "abc  a.zip\nDEF *b.tar.gz\n\nbad line here\n"
	for name, want := range map[string]string{"a.zip": "abc", "b.tar.gz": "def", "c": ""} {
		got, ok := digestOf(sums, name)
		if got != want || ok != (want != "") {
			t.Errorf("digestOf(%q) = %q, %v", name, got, ok)
		}
	}
}
