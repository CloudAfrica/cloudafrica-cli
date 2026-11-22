package update

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Options is everything Run touches, so tests never use the real executable.
type Options struct {
	// Exe is the running binary, already passed through filepath.EvalSymlinks.
	Exe          string
	GOOS, GOARCH string
	// Current is the running version as built ("0.3.1", or "dev").
	Current string
	// Check reports whether a newer release exists and changes nothing.
	Check        bool
	APIURL       string // LatestReleaseURL in production
	DownloadBase string // DownloadBase in production
	Client       *http.Client
	UserAgent    string
	Getenv       func(string) string

	// Zero means the default: 10 s, 5 min, 10 s and, on Windows, 2 s.
	LatestTimeout   time.Duration
	DownloadTimeout time.Duration
	ProbeTimeout    time.Duration
	RetryFor        time.Duration
}

// State is what Run found or did.
type State int

const (
	// UpToDate: the running version is the latest release. It starts at 1 so
	// the zero State, which every failed Run returns, names no outcome.
	UpToDate State = iota + 1
	// Ahead: the running version is newer than the latest release (an rc).
	Ahead
	// Available: a newer release exists and nothing was changed (--check, or
	// a package manager owns the binary).
	Available
	// Updated: the binary was replaced.
	Updated
)

// Result reports a Run.
type Result struct {
	State State
	// Current and Latest are versions without the tag's v.
	Current, Latest string
	// Managed is the package-manager command that owns the binary, or "".
	Managed string
	// SiblingErr is set when the Windows binary was replaced but the other
	// name (ca.exe or cloudafrica.exe) beside it could not be.
	SiblingErr error
}

// RefusedError is a request Run will not carry out (a dev build, a directory
// it cannot write to): the caller's to fix, not a failure of GitHub or the
// network.
type RefusedError struct{ msg string }

func (e *RefusedError) Error() string { return e.msg }

const (
	tempPrefix  = ".ca-update-"
	asideSuffix = ".old-"
)

// Run checks for a newer release and, unless Check is set or a package
// manager owns the binary, replaces Exe with it.
func Run(ctx context.Context, o Options) (Result, error) {
	o = o.withDefaults()
	if _, ok := parseVersion(o.Current); !ok {
		return Result{}, &RefusedError{"a development build cannot update itself; install a release instead"}
	}
	res := Result{Current: strings.TrimPrefix(o.Current, "v"), Managed: Managed(o.Exe, o.GOOS, o.Getenv)}

	lctx, cancel := context.WithTimeout(ctx, o.LatestTimeout)
	tag, err := Latest(lctx, o.Client, o.APIURL, o.UserAgent)
	cancel()
	if err != nil {
		return res, err
	}
	res.Latest = strings.TrimPrefix(tag, "v")
	switch {
	case Newer(o.Current, tag):
		res.State = Ahead
		return res, nil
	case !Newer(tag, o.Current):
		res.State = UpToDate
		return res, nil
	case o.Check || res.Managed != "":
		res.State = Available
		return res, nil
	}

	dir := filepath.Dir(o.Exe)
	sweepStale(dir)
	tmp, err := os.MkdirTemp(dir, tempPrefix)
	if err != nil && !errors.Is(err, fs.ErrPermission) {
		return res, fmt.Errorf("creating a temporary directory beside ca: %w", err)
	}
	if err != nil {
		msg := fmt.Sprintf("cannot write to %s, where ca is installed", dir)
		if o.GOOS != "windows" {
			msg += "; run `sudo ca update`"
		}
		return res, &RefusedError{msg}
	}
	defer os.RemoveAll(tmp)

	names := binaries(o.GOOS)
	dctx, cancel := context.WithTimeout(ctx, o.DownloadTimeout)
	err = fetch(dctx, o, tag, res.Latest, tmp, names)
	cancel()
	if err != nil {
		return res, err
	}

	fresh := filepath.Join(tmp, source(o.GOOS, o.Exe))
	if err := os.Chmod(fresh, 0o755); err != nil {
		return res, err
	}
	line, err := probe(ctx, fresh, o.ProbeTimeout)
	if err != nil {
		return res, fmt.Errorf("the downloaded ca does not run (%w); nothing was replaced", err)
	}
	if !strings.HasPrefix(line, "ca "+res.Latest+" (") {
		return res, fmt.Errorf("the downloaded ca reports %q, not version %s; nothing was replaced", line, res.Latest)
	}

	if o.GOOS != "windows" {
		if err := os.Rename(fresh, o.Exe); err != nil {
			return res, fmt.Errorf("replacing %s: %w", o.Exe, err)
		}
		res.State = Updated
		return res, nil
	}
	sweepAsides(dir)
	if err := replaceRunning(fresh, o.Exe, o.RetryFor); err != nil {
		return res, err
	}
	res.State = Updated
	res.SiblingErr = replaceSibling(ctx, o, tmp)
	return res, nil
}

func (o Options) withDefaults() Options {
	if o.LatestTimeout == 0 {
		o.LatestTimeout = 10 * time.Second
	}
	if o.DownloadTimeout == 0 {
		o.DownloadTimeout = 5 * time.Minute
	}
	if o.ProbeTimeout == 0 {
		o.ProbeTimeout = 10 * time.Second
	}
	if o.RetryFor == 0 && o.GOOS == "windows" {
		o.RetryFor = 2 * time.Second
	}
	if o.Getenv == nil {
		o.Getenv = func(string) string { return "" }
	}
	return o
}

// binaries are the files taken from the release archive.
func binaries(goos string) []string {
	if goos == "windows" {
		return []string{"ca.exe", "cloudafrica.exe"}
	}
	return []string{"ca"}
}

// source is the archive member that replaces exe. On Unix the archive holds
// only ca, which also replaces a cloudafrica that install.sh wrote as a file
// beside a foreign ca. On Windows both names are in the zip.
func source(goos, exe string) string {
	if goos == "windows" && strings.EqualFold(filepath.Base(exe), "cloudafrica.exe") {
		return "cloudafrica.exe"
	}
	return binaries(goos)[0]
}

// sibling is the other Windows name beside exe, or "".
func sibling(exe string) string {
	switch strings.ToLower(filepath.Base(exe)) {
	case "ca.exe":
		return "cloudafrica.exe"
	case "cloudafrica.exe":
		return "ca.exe"
	}
	return ""
}

// replaceSibling swaps the other Windows name when it is this CLI, as
// install.ps1 decides it: one that runs and exits non-zero or prints another
// line is a foreign ca.exe and stays. One that cannot be started, or does not
// finish, is left too, with an error so the user hears of it.
func replaceSibling(ctx context.Context, o Options, tmp string) error {
	name := sibling(o.Exe)
	if name == "" {
		return nil
	}
	dst := filepath.Join(filepath.Dir(o.Exe), name)
	if !exists(dst) {
		return nil
	}
	line, err := probe(ctx, dst, o.ProbeTimeout)
	var exited *exec.ExitError
	if errors.As(err, &exited) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s was not updated: running it to check it is ca: %w", dst, err)
	}
	if !ours(line) {
		return nil
	}
	if err := replaceRunning(filepath.Join(tmp, name), dst, o.RetryFor); err != nil {
		return fmt.Errorf("%s was not updated: %w", dst, err)
	}
	return nil
}

// ours reports whether a `version` first line is this CLI's: "ca <v> (".
func ours(line string) bool {
	rest, ok := strings.CutPrefix(line, "ca ")
	return ok && strings.Contains(rest, " (")
}

// probe runs `path version` and returns its first line. stdout is a pipe, so
// the binary does no terminal-only update check. A non-zero exit is an
// *exec.ExitError carrying the start of stderr; a timeout is not one.
func probe(ctx context.Context, path string, timeout time.Duration) (string, error) {
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(pctx, path, "version")
	cmd.Stdin = nil
	out, err := cmd.Output()
	if err != nil && pctx.Err() != nil && ctx.Err() == nil {
		return "", fmt.Errorf("%s version did not finish within %s", filepath.Base(path), timeout)
	}
	var exited *exec.ExitError
	if errors.As(err, &exited) {
		if msg := strings.TrimSpace(string(exited.Stderr)); msg != "" {
			if len(msg) > 200 {
				msg = msg[:200] + "..."
			}
			return "", fmt.Errorf("%w: %s", err, msg)
		}
	}
	if err != nil {
		return "", err
	}
	line, _, _ := bufio.NewReader(bytes.NewReader(out)).ReadLine()
	return string(line), nil
}

// replaceRunning moves src to dst where dst may be running: Windows will not
// overwrite a running exe but lets it be renamed, so dst goes aside under a
// unique name first and comes back if the move fails.
func replaceRunning(src, dst string, retryFor time.Duration) error {
	aside := fmt.Sprintf("%s%s%d", dst, asideSuffix, time.Now().UnixNano())
	if err := retry(retryFor, func() error { return os.Rename(dst, aside) }); err != nil {
		return fmt.Errorf("moving %s aside: %w", dst, err)
	}
	if err := retry(retryFor, func() error { return os.Rename(src, dst) }); err != nil {
		if rerr := retry(retryFor, func() error { return os.Rename(aside, dst) }); rerr != nil {
			return fmt.Errorf("installing %s: %w; the previous version is at %s (rename it back)", dst, err, aside)
		}
		return fmt.Errorf("installing %s: %w", dst, err)
	}
	_ = os.Remove(aside) // fails while that copy is still running; the next update sweeps it
	return nil
}

// retry calls f until it succeeds or d has passed, backing off from 25 ms:
// antivirus scanning a new exe briefly locks it.
func retry(d time.Duration, f func() error) error {
	deadline := time.Now().Add(d)
	wait := 25 * time.Millisecond
	for {
		err := f()
		if err == nil || !time.Now().Add(wait).Before(deadline) {
			return err
		}
		time.Sleep(wait)
		wait *= 2
	}
}

// sweepAsides deletes earlier Windows asides; one still running stays.
func sweepAsides(dir string) {
	for _, name := range []string{"ca.exe", "cloudafrica.exe"} {
		old, _ := filepath.Glob(filepath.Join(dir, name+asideSuffix+"*"))
		for _, p := range old {
			_ = os.Remove(p)
		}
	}
}

// sweepStale deletes temp directories a killed update left behind; one
// younger than an hour may belong to an update still running.
func sweepStale(dir string) {
	old, _ := filepath.Glob(filepath.Join(dir, tempPrefix+"*"))
	for _, p := range old {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() && time.Since(fi.ModTime()) > time.Hour {
			_ = os.RemoveAll(p)
		}
	}
}
