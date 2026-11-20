// Package cli holds the `ca` commands.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/auth"
	"github.com/CloudAfrica/cloudafrica-cli/internal/output"
	"github.com/CloudAfrica/cloudafrica-cli/internal/profile"
)

// DefaultEndpoint is the only endpoint compiled in.
const DefaultEndpoint = "https://api.cloudafrica.net"

// App is the CLI's environment; tests replace any of it.
type App struct {
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader
	Getenv  func(string) string
	// TTY reports whether stdout and stdin are an interactive terminal.
	TTY         bool
	OpenBrowser func(url string) error
	Store       profile.Store
	// CacheDir holds non-secret caches (the update check); "" disables them.
	CacheDir string
	// Transport is the base RoundTripper for API and OAuth calls (nil: the default).
	Transport http.RoundTripper
	// LoginTimeout bounds the wait for the browser sign-in (0: the default).
	LoginTimeout time.Duration

	stdin *bufio.Reader // shared by every prompt, so none swallows another's answer

	// Flags.
	profileFlag string
	outputFlag  string
	quiet       bool
}

// session is a resolved credential for one command run.
type session struct {
	name     string // profile name; "" for CLOUDAFRICA_TOKEN
	endpoint string
	envToken bool
	source   *auth.Source
	client   *cloudafrica.Client
}

func (a *App) authHTTP() *http.Client {
	hc := auth.HTTPClient()
	if a.Transport != nil {
		hc.Transport = a.Transport
	}
	return hc
}

func (a *App) printer() (output.Printer, error) {
	f, err := output.Resolve(a.outputFlag, a.TTY)
	if err != nil {
		return output.Printer{}, usageErr(err)
	}
	return output.Printer{W: a.Stdout, Format: f, Quiet: a.quiet}, nil
}

func (a *App) profileName() (string, error) {
	name, err := profile.Resolve(a.profileFlag, a.Getenv)
	if err != nil {
		return "", usageErr(err)
	}
	return name, nil
}

// envEndpoint is CLOUDAFRICA_ENDPOINT without a trailing slash, or "".
func (a *App) envEndpoint() string {
	return strings.TrimRight(strings.TrimSpace(a.Getenv("CLOUDAFRICA_ENDPOINT")), "/")
}

// session resolves the credential: CLOUDAFRICA_TOKEN overrides any profile;
// otherwise the selected profile, whose endpoint must agree with
// CLOUDAFRICA_ENDPOINT when that is set.
func (a *App) session() (*session, error) {
	var s *session
	if tok := strings.TrimSpace(a.Getenv("CLOUDAFRICA_TOKEN")); tok != "" {
		ep := a.envEndpoint()
		if ep == "" {
			ep = DefaultEndpoint
		}
		s = &session{endpoint: ep, envToken: true}
		c, err := a.newClient(ep, cloudafrica.StaticToken(tok))
		if err != nil {
			return nil, err
		}
		s.client = c
		return s, nil
	}
	name, err := a.profileName()
	if err != nil {
		return nil, err
	}
	p, err := a.Store.Load(name)
	if errors.Is(err, profile.ErrNotFound) {
		return nil, &auth.LoginRequiredError{Profile: name, Reason: fmt.Sprintf("profile %q is not signed in", name)}
	}
	if err != nil {
		return nil, err
	}
	if ep := a.envEndpoint(); ep != "" && !sameOrigin(ep, p.Endpoint) {
		return nil, usageErr(fmt.Errorf("profile %q is signed in to %s, but CLOUDAFRICA_ENDPOINT is %s; use a profile signed in there",
			name, p.Endpoint, ep))
	}
	src := &auth.Source{Store: a.Store, Name: name, HTTP: a.authHTTP()}
	c, err := a.newClient(p.Endpoint, src)
	if err != nil {
		return nil, err
	}
	return &session{name: name, endpoint: p.Endpoint, source: src, client: c}, nil
}

func sameOrigin(a, b string) bool {
	oa, err1 := auth.Origin(a)
	ob, err2 := auth.Origin(b)
	return err1 == nil && err2 == nil && oa == ob
}

func (a *App) newClient(endpoint string, ts cloudafrica.TokenSource) (*cloudafrica.Client, error) {
	return cloudafrica.New(endpoint, ts,
		cloudafrica.WithUserAgent("ca/"+a.Version),
		cloudafrica.WithTransport(a.Transport))
}

// withSession runs fn against a session. A step-up refusal on an
// interactive terminal with a stored profile offers to re-authorize with
// protected scopes, then runs fn once more.
func (a *App) withSession(ctx context.Context, fn func(s *session) error) error {
	s, err := a.session()
	if err != nil {
		return err
	}
	err = fn(s)
	slug, ok := stepUpSlug(err)
	if !ok || s.envToken || !a.TTY {
		return err
	}
	fmt.Fprintf(a.Stderr, "%v\n", err)
	if slug == "not-paired" {
		fmt.Fprintln(a.Stderr, "This action needs protected access, and the elevation window has lapsed.")
	} else {
		fmt.Fprintln(a.Stderr, "This action needs a scope this sign-in was not granted.")
	}
	if !a.confirm("Re-authorize in the browser with protected access now?") {
		return err
	}
	if lerr := a.login(ctx, s.name, true); lerr != nil {
		return lerr
	}
	s, serr := a.session()
	if serr != nil {
		return serr
	}
	return fn(s)
}

// confirm asks a yes/no question on the terminal; anything but y/yes is no.
func (a *App) confirm(q string) bool {
	fmt.Fprintf(a.Stderr, "%s [y/N] ", q)
	if a.stdin == nil {
		a.stdin = bufio.NewReader(a.Stdin)
	}
	line, _ := a.stdin.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}

// Root builds the command tree.
func (a *App) Root() *cobra.Command {
	root := group("ca", "The CloudAfrica command-line interface")
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetOut(a.Stdout)
	root.SetErr(a.Stderr)
	root.SetIn(a.Stdin)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageErr(err) })
	pf := root.PersistentFlags()
	pf.StringVar(&a.profileFlag, "profile", "", "profile to use (default $CLOUDAFRICA_PROFILE, then \"default\")")
	pf.StringVarP(&a.outputFlag, "output", "o", "", "output format: table, json or yaml (default table on a terminal, json otherwise)")
	pf.BoolVarP(&a.quiet, "quiet", "q", false, "print ids only")

	root.AddCommand(a.authCmd(), a.vmCmd(), a.operationCmd(), a.apiCmd(), a.versionCmd())
	return root
}

// Run executes args and returns the process exit code.
func (a *App) Run(ctx context.Context, args []string) int {
	root := a.Root()
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return ExitOK
	}
	code := ExitCode(err)
	fmt.Fprintf(a.Stderr, "Error: %v\n", err)
	if hint := hint(err); hint != "" {
		fmt.Fprintln(a.Stderr, hint)
	}
	return code
}

// NewFromOS builds an App on the real process environment.
func NewFromOS(version string, tty bool, open func(string) error) (*App, error) {
	dir, err := profile.DefaultDir()
	if err != nil {
		return nil, err
	}
	cache, err := os.UserCacheDir()
	if err == nil {
		cache = filepath.Join(cache, "cloudafrica")
	} else {
		cache = ""
	}
	return &App{
		CacheDir:    cache,
		Version:     version,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Stdin:       os.Stdin,
		Getenv:      os.Getenv,
		TTY:         tty,
		OpenBrowser: open,
		Store:       profile.Store{Dir: dir},
	}, nil
}
