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
	"github.com/CloudAfrica/cloudafrica-cli/internal/cmdspec"
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
	// CacheDir holds non-secret caches; "" disables them and the update check.
	CacheDir string
	// Transport is the base RoundTripper for API, OAuth and GitHub release calls (nil: the default).
	Transport http.RoundTripper
	// LoginTimeout bounds the wait for the browser sign-in (0: the default).
	LoginTimeout time.Duration
	// GOOS picks the default sign-in flow ("": runtime.GOOS).
	GOOS string
	// DeviceWait pauses between the device sign-in's polls (nil: a timer).
	DeviceWait func(ctx context.Context, d time.Duration) error
	// ElevatePoll and ElevateTimeout pace `ca auth elevate` (0: 2 s and 5 min).
	ElevatePoll    time.Duration
	ElevateTimeout time.Duration
	// PairGrace and PairGracePoll pace the wait for a device sign-in's pairing
	// to land (0: 20 s and 2 s).
	PairGrace     time.Duration
	PairGracePoll time.Duration
	// OperationPoll paces every operation wait (0: from 2 s, doubling to 10 s).
	OperationPoll time.Duration
	// Exe is the name the CLI was run as ("" means "ca"); a kubeconfig from
	// `ca k8s kubeconfig` runs it again.
	Exe string
	// Executable finds the running binary, for `ca update` and the update
	// notice's command (nil: os.Executable).
	Executable func() (string, error)

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
	// elevation is CLOUDAFRICA_ELEVATION for CLOUDAFRICA_TOKEN, else "".
	elevation string
	scope     string // the profile's granted scopes, space-separated; "" when unknown
	source    *auth.Source
	client    *cloudafrica.Client
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
// CLOUDAFRICA_ENDPOINT when that is set. CLOUDAFRICA_ELEVATION goes with
// CLOUDAFRICA_TOKEN only, and must be well formed.
func (a *App) session() (*session, error) {
	var s *session
	tok := strings.TrimSpace(a.Getenv("CLOUDAFRICA_TOKEN"))
	elevation := strings.TrimSpace(a.Getenv(elevationEnv))
	switch {
	case elevation != "" && tok == "":
		return nil, usageErr(errElevationWithoutToken)
	case elevation != "" && !wellFormedElevation(elevation):
		return nil, usageErr(errElevationMalformed)
	}
	if tok != "" {
		ep := a.envEndpoint()
		if ep == "" {
			ep = DefaultEndpoint
		}
		if err := auth.RequireTLS(ep); err != nil {
			return nil, usageErr(err)
		}
		s = &session{endpoint: ep, envToken: true, elevation: elevation}
		c, err := a.newClient(ep, cloudafrica.StaticToken(tok), elevation)
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
	src := &auth.Source{Store: a.Store, Name: name, HTTP: a.authHTTP(), Endpoint: p.Endpoint}
	c, err := a.newClient(p.Endpoint, src, "")
	if err != nil {
		return nil, err
	}
	return &session{name: name, endpoint: p.Endpoint, scope: p.Scope, source: src, client: c}, nil
}

func sameOrigin(a, b string) bool {
	oa, err1 := auth.Origin(a)
	ob, err2 := auth.Origin(b)
	return err1 == nil && err2 == nil && oa == ob
}

// newClient builds the API client. A non-empty elevation is sent on every
// request; it wraps this client's transport only, never a.Transport, which
// the update check also uses.
func (a *App) newClient(endpoint string, ts cloudafrica.TokenSource, elevation string) (*cloudafrica.Client, error) {
	base := a.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	var rt http.RoundTripper = &singleShotTransport{base: base}
	if elevation != "" {
		rt = &elevationTransport{base: rt, secret: elevation}
	}
	return cloudafrica.New(endpoint, ts,
		cloudafrica.WithUserAgent("ca/"+a.Version),
		cloudafrica.WithTransport(rt))
}

// pairGraceDefault is how long a device sign-in's pairing may trail its token:
// the portal approves the device request, then pairs the grant, and `ca` can
// be handed the token in between. It matches the portal's own wait on a
// failed pairing.
const (
	pairGraceDefault     = 20 * time.Second
	pairGracePollDefault = 2 * time.Second
)

// withSession runs fn against a session. On an interactive terminal with a
// stored profile, a step-up refusal offers its remedy and runs fn again: both
// not-paired and insufficient-scope offer `ca auth login` (when signing in
// can grant the scope), because approving a sign-in that holds a protected:
// scope pairs it. Signing in is offered at most once, so a refusal that
// persists after it ends the loop. After a device sign-in, not-paired is
// retried for a while first, since its pairing can trail the token.
func (a *App) withSession(ctx context.Context, fn func(s *session) error) error {
	s, err := a.session()
	if err != nil {
		return err
	}
	err = fn(s)
	if s.envToken || !a.TTY {
		return err
	}
	signedIn := false
	for {
		slug, ok := stepUpSlug(err)
		if !ok || signedIn || !loginGrants(refusedScope(err)) {
			return err
		}
		fmt.Fprintf(a.Stderr, "%v\n", err)
		if slug == "not-paired" {
			fmt.Fprintln(a.Stderr, "This is a protected action, and this sign-in is not paired for it. "+
				"Signing in again and approving it pairs it.")
		} else {
			fmt.Fprintln(a.Stderr, "This sign-in was not granted the scope this action needs.")
		}
		if !a.confirm("Sign in again now (`ca auth login`)?") {
			return err
		}
		used, lerr := a.login(ctx, s.name, "", flowAuto, refusedScope(err))
		if lerr != nil {
			return lerr
		}
		signedIn = true
		if s, err = a.session(); err != nil {
			return err
		}
		err = fn(s)
		if used == flowDevice {
			_, err = a.awaitPairing(ctx, s, err, fn)
		}
	}
}

// awaitPairing repeats fn while it is refused not-paired, up to the grace
// period, and returns the session and error of the last attempt.
func (a *App) awaitPairing(ctx context.Context, s *session, err error, fn func(s *session) error) (*session, error) {
	grace, poll := a.PairGrace, a.PairGracePoll
	if grace <= 0 {
		grace = pairGraceDefault
	}
	if poll <= 0 {
		poll = pairGracePollDefault
	}
	deadline := time.Now().Add(grace)
	announced := false
	for {
		if slug, _ := stepUpSlug(err); (slug != "not-paired" && !errors.Is(err, errStillNotPaired)) ||
			time.Now().After(deadline) {
			return s, err
		}
		if !announced {
			announced = true
			fmt.Fprintln(a.Stderr, "Waiting for the sign-in to be paired for protected actions...")
		}
		select {
		case <-ctx.Done():
			return s, ctx.Err()
		case <-time.After(poll):
		}
		next, serr := a.session()
		if serr != nil {
			return s, serr
		}
		s, err = next, fn(next)
	}
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
	// Reject a bad --output before a command creates, changes or deletes anything.
	root.PersistentPreRunE = func(*cobra.Command, []string) error {
		_, err := a.printer()
		return err
	}
	pf := root.PersistentFlags()
	pf.StringVar(&a.profileFlag, "profile", "", "profile to use (default $CLOUDAFRICA_PROFILE, then \"default\")")
	pf.StringVarP(&a.outputFlag, "output", "o", "", "output format: table, json or yaml (default table on a terminal, json otherwise)")
	pf.BoolVarP(&a.quiet, "quiet", "q", false, "print ids only")

	for _, s := range cmdspec.Sections {
		root.AddGroup(&cobra.Group{ID: s.ID, Title: s.Title})
	}
	root.SetHelpCommandGroupID("cli")
	root.SetCompletionCommandGroupID("cli")
	for _, c := range []*cobra.Command{a.authCmd(), a.updateCmd(), a.versionCmd()} {
		c.GroupID = "cli"
		root.AddCommand(c)
	}
	root.AddCommand(a.vmCmd(), a.k8sCmd(), a.operationCmd())
	a.addGenerated(root)
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
	envToken := strings.TrimSpace(a.Getenv("CLOUDAFRICA_TOKEN")) != ""
	if hint := hint(err, envToken, envToken && strings.TrimSpace(a.Getenv(elevationEnv)) != ""); hint != "" {
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
		Exe:         exeName(os.Args[0]),
	}, nil
}

// exeName is the command name the CLI was run as, without a Windows .exe;
// kubectl runs it again by PATH. An unusable argv0 is "".
func exeName(argv0 string) string {
	name := strings.TrimSuffix(filepath.Base(argv0), ".exe")
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	return name
}
