package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/auth"
	"github.com/CloudAfrica/cloudafrica-cli/internal/output"
	"github.com/CloudAfrica/cloudafrica-cli/internal/profile"
)

// defaultScopes is what every sign-in requests: the read: and write: scopes
// of every API operation the CLI covers or has pending for a later slice
// (generated), so a sign-in made now also covers commands added later; and the protected:
// scopes of its commonest protected actions. Holding a protected: scope is
// not enough for a protected action: the sign-in must also be paired, which
// is `ca auth elevate`.
var defaultScopes = append(slices.Clone(genScopes),
	"protected:compute", "protected:network", "protected:backup", "protected:s3")

const (
	elevatePollDefault    = 2 * time.Second
	elevateTimeoutDefault = 5 * time.Minute
	elevateLabel          = "CloudAfrica CLI (ca)"
	elevateTokenLabel     = "CloudAfrica CLI (ca) — API token"
)

func (a *App) authCmd() *cobra.Command {
	cmd := group("auth", "Sign in, sign out and list profiles")
	var extra []string
	var accountID string
	login := &cobra.Command{
		Use:   "login",
		Short: "Sign in through the browser",
		Long: "Sign in through the browser and store the session in a profile. One profile holds one account: " +
			"to use several accounts, sign in once per account with --profile and name the account with " +
			"--account. Signing in again to a profile stays in the account it is signed in to. Signing in " +
			"never allows protected actions such as vm delete; `ca auth elevate` does.",
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			name, err := a.profileName()
			if err != nil {
				return err
			}
			return a.login(cmd.Context(), name, accountID, extra...)
		},
	}
	login.Flags().StringArrayVar(&extra, "scope", nil, "also request this scope, e.g. protected:billing "+
		"(repeatable; kept for every later sign-in of this profile)")
	login.Flags().StringVar(&accountID, "account", "", "sign in to this account id (default: the account this "+
		"profile is already signed in to)")
	elevate := &cobra.Command{
		Use:   "elevate",
		Short: "Allow protected actions (such as vm delete) for this sign-in or API token",
		Long: "Pair the selected profile's sign-in for protected actions. Opens a page in the portal where you " +
			"step up and confirm the code shown here, then waits up to 5 minutes for the confirmation. " +
			"The pairing lasts for the account's elevation window.\n\n" +
			"With CLOUDAFRICA_TOKEN set, it pairs that API token instead and prints its secret, alone, on " +
			"stdout; that pairing also ends with the confirming person's portal session. A script sets it with\n\n" +
			"  CLOUDAFRICA_ELEVATION=$(ca auth elevate) || exit; export CLOUDAFRICA_ELEVATION\n\n" +
			"An existing CLOUDAFRICA_ELEVATION is paired again rather than replaced.",
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := a.session()
			if err != nil {
				return err
			}
			return a.elevate(cmd.Context(), s)
		},
	}

	status := &cobra.Command{
		Use:   "status",
		Short: "Show the selected profile's sign-in",
		Args:  noArgs,
		RunE:  func(*cobra.Command, []string) error { return a.authStatus() },
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		Args:  noArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return a.authList() },
	}
	logout := &cobra.Command{
		Use:   "logout",
		Short: "Revoke the selected profile's session and delete the profile",
		Args:  noArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return a.logout(cmd.Context()) },
	}
	cmd.AddCommand(login, elevate, status, list, logout)
	return cmd
}

// login signs name in. The endpoint is CLOUDAFRICA_ENDPOINT, else the
// profile's existing endpoint, else the default. A successful login into an
// existing profile revokes that profile's previous grant: every authorize
// makes a new one, and the old would otherwise stay live in Connected apps.
// A corrupt profile file is replaced, since login is how it gets repaired.
func (a *App) login(ctx context.Context, name, accountID string, extra ...string) error {
	old, err := a.loadForReplace(name)
	if err != nil {
		return err
	}
	endpoint := a.envEndpoint()
	if endpoint == "" && old != nil {
		endpoint = old.Endpoint
	}
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	hc := a.authHTTP()
	meta, err := auth.Discover(ctx, hc, endpoint)
	if err != nil {
		return err
	}
	var extras []string
	if old != nil {
		extras = append(extras, old.ExtraScopes...)
	}
	for _, sc := range extra {
		sc = strings.TrimSpace(sc)
		if sc == "" {
			continue
		}
		if !strings.Contains(sc, ":") {
			return usageErr(fmt.Errorf("--scope %q is not a scope: use the form read:compute or protected:billing", sc))
		}
		if !slices.Contains(defaultScopes, sc) && !slices.Contains(extras, sc) {
			extras = append(extras, sc)
		}
	}
	scopes := append(slices.Clone(defaultScopes), extras...)
	accountID = strings.TrimSpace(accountID)
	reused := false
	if accountID == "" && old != nil && old.Endpoint == endpoint && old.AccountID != "" {
		accountID, reused = old.AccountID, true
	}
	tok, err := auth.Login(ctx, hc, meta, auth.LoginOptions{
		Scopes:  scopes,
		Account: accountID,
		Open:    a.OpenBrowser,
		Out:     a.Stderr,
		Timeout: a.LoginTimeout,
	})
	if err != nil {
		if reused && strings.Contains(err.Error(), "access_denied") {
			return &hintedError{
				err: fmt.Errorf("%w: sign-in was declined for account %s (the page refused it, or you pressed Deny)",
					auth.ErrLoginRequired, accountID),
				hint: "That is the account this profile is signed in to. Sign in to another with " +
					"`ca auth login --account <id>`; `ca auth logout` removes this profile.",
			}
		}
		return &auth.LoginRequiredError{Profile: name, Reason: err.Error()}
	}
	p := &profile.Profile{
		Endpoint:     endpoint,
		Scope:        tok.Scope,
		ExtraScopes:  extras,
		AccessToken:  tok.AccessToken,
		Expiry:       tok.Expiry(time.Now()),
		RefreshToken: tok.RefreshToken,
	}
	acct, aerr := a.fetchAccount(ctx, endpoint, tok.AccessToken)
	switch {
	case aerr != nil && accountID != "":
		return a.rejectSession(ctx, p, fmt.Errorf("%w: could not confirm the session is for account %s (%v); "+
			"nothing was saved", auth.ErrLoginRequired, accountID, aerr), "Sign in again once the API is reachable.")
	case aerr != nil:
		fmt.Fprintf(a.Stderr, "Warning: could not read the account name: %v\n", aerr)
	case accountID != "" && acct.id != accountID:
		return a.rejectSession(ctx, p, fmt.Errorf("%w: asked to sign in to account %s but the session is for %s; "+
			"nothing was saved", auth.ErrLoginRequired, accountID, orUnknown(acct.id)),
			"One profile holds one account: use `ca auth login --profile <name> --account <id>` "+
				"for each account. An older gateway ignores --account.")
	default:
		p.AccountID, p.AccountName = acct.id, acct.name
	}
	// Under the profile lock, so a refresh in another process cannot
	// overwrite this save; re-read, because that refresh may have rotated the
	// grant being replaced.
	release, err := a.Store.Lock(ctx, name)
	if err != nil {
		return err
	}
	if cur, err := a.Store.Load(name); err == nil {
		old = cur
	}
	err = a.Store.Save(name, p)
	release()
	if err != nil {
		return err
	}
	if missing := missingScopes(scopes, tok.Scope); len(missing) > 0 {
		fmt.Fprintf(a.Stderr, "Warning: not granted: %s. Commands that need them will be refused; "+
			"sign in again and tick them on the consent page.\n", strings.Join(missing, ", "))
	}
	if old != nil && old.RefreshToken != "" && old.RefreshToken != p.RefreshToken {
		if err := a.revokeOld(ctx, old); err != nil {
			fmt.Fprintf(a.Stderr, "Warning: the previous session for this profile could not be revoked (%v); "+
				"remove it under Connected apps in the portal.\n", err)
		}
	}
	who := p.AccountName
	if who == "" {
		who = p.AccountID
	}
	fmt.Fprintf(a.Stderr, "Signed in to %s as profile %q.\n", orUnknown(who), name)
	return nil
}

// rejectSession revokes the grant just issued for p, which login will not
// save, and returns err with hint.
func (a *App) rejectSession(ctx context.Context, p *profile.Profile, err error, hint string) error {
	if p.RefreshToken != "" {
		if rerr := a.revokeOld(ctx, p); rerr != nil {
			fmt.Fprintf(a.Stderr, "Warning: the new session could not be revoked (%v); "+
				"remove it under Connected apps in the portal.\n", rerr)
		}
	}
	return &hintedError{err: err, hint: hint}
}

// elevate pairs s's grant for protected actions: it asks for a pairing,
// opens the confirm page, and repeats the request until the answer is
// paired or the wait runs out. The gateway derives the page and code from
// the grant, so the page is opened once. A sign-in known to lack
// any protected: scope is refused before any request, since pairing it would
// not allow protected actions. For CLOUDAFRICA_TOKEN it pairs a secret
// instead (CLOUDAFRICA_ELEVATION, else a new one) and prints it, alone, on
// stdout once paired, whatever -o says: a script captures it with $(...).
func (a *App) elevate(ctx context.Context, s *session) error {
	if s.scope != "" && !slices.ContainsFunc(strings.Fields(s.scope), func(sc string) bool {
		return strings.HasPrefix(sc, "protected:")
	}) {
		return &hintedError{err: errNoProtectedScope,
			hint: "Run `ca auth login` and tick the protected: scopes on the consent page, then `ca auth elevate`."}
	}
	poll, limit := a.ElevatePoll, a.ElevateTimeout
	if poll <= 0 {
		poll = elevatePollDefault
	}
	if limit <= 0 {
		limit = elevateTimeoutDefault
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	timedOut := func() error {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("the pairing was not confirmed within %s: %w", limit, context.DeadlineExceeded)
		}
		return ctx.Err()
	}
	label, who := elevateLabel, "this sign-in"
	params := &cloudafrica.CreateElevationRequestParams{}
	secret := ""
	if s.envToken {
		label, who = elevateTokenLabel, "this API token"
		secret = s.elevation
		if secret == "" {
			var err error
			if secret, err = newElevationSecret(); err != nil {
				return fmt.Errorf("generating the elevation secret: %w", err)
			}
		}
		params.XCloudAfricaElevation = &secret
	}
	paired := func() error {
		if secret == "" {
			return nil
		}
		if _, err := fmt.Fprintln(a.Stdout, secret); err != nil {
			return fmt.Errorf("writing the elevation secret to stdout: %w", err)
		}
		return nil
	}
	shown := ""
	for {
		r, err := s.client.API.CreateElevationRequestWithResponse(ctx, params,
			cloudafrica.PublicElevationRequestBody{Label: &label})
		if err != nil {
			if ctx.Err() != nil {
				return timedOut()
			}
			return err
		}
		if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
			return err
		}
		if r.JSON200 == nil {
			return errors.New("empty elevation response")
		}
		if d := r.JSON200; d.Paired != nil && *d.Paired {
			if shown == "" {
				fmt.Fprintf(a.Stderr, "%s is already allowed protected actions.\n", capitalise(who))
			} else {
				fmt.Fprintf(a.Stderr, "Paired: %s may use protected actions for the elevation window.\n", who)
			}
			return paired()
		}
		page, code := output.S(r.JSON200.Url), output.S(r.JSON200.Code)
		if page == "" {
			return errors.New("the elevation response has neither paired nor a page to confirm on")
		}
		if code == "" {
			return errors.New("the elevation response carries no confirmation code")
		}
		if shown == "" {
			if err := auth.RequireTLS(page); err != nil {
				return fmt.Errorf("refusing the pairing page: %w", err)
			}
			fmt.Fprintf(a.Stderr, "Opening your browser to allow protected actions. If it does not open, visit:\n\n"+
				"  %s\n\nCheck that the page shows the code %s, then confirm it.\n", page, code)
			if a.OpenBrowser != nil {
				if err := a.OpenBrowser(page); err != nil {
					fmt.Fprintf(a.Stderr, "(could not open a browser: %v)\n", err)
				}
			}
			fmt.Fprintln(a.Stderr, "Waiting for confirmation...")
			shown = page
		}
		select {
		case <-ctx.Done():
			return timedOut()
		case <-time.After(poll):
		}
	}
}

// loadForReplace loads a profile about to be replaced: nil when
// it has no file, and nil with a warning when its file is corrupt.
func (a *App) loadForReplace(name string) (*profile.Profile, error) {
	p, err := a.Store.Load(name)
	switch {
	case errors.Is(err, profile.ErrNotFound):
		return nil, nil
	case errors.Is(err, profile.ErrCorrupt):
		fmt.Fprintf(a.Stderr, "Warning: %v; replacing it. Any session it held stays listed under Connected apps "+
			"in the portal.\n", err)
		return nil, nil
	}
	return p, err
}

func (a *App) revokeOld(ctx context.Context, old *profile.Profile) error {
	hc := a.authHTTP()
	meta, err := auth.Discover(ctx, hc, old.Endpoint)
	if err != nil {
		return err
	}
	return auth.Revoke(ctx, hc, meta, old.RefreshToken)
}

// missingScopes lists the requested scopes the token response did not grant.
func missingScopes(requested []string, granted string) []string {
	have := map[string]bool{}
	for _, s := range strings.Fields(granted) {
		have[s] = true
	}
	var out []string
	for _, s := range requested {
		if !have[s] {
			out = append(out, s)
		}
	}
	return out
}

type account struct{ id, name string }

func (a *App) fetchAccount(ctx context.Context, endpoint, token string) (*account, error) {
	c, err := a.newClient(endpoint, cloudafrica.StaticToken(token), "")
	if err != nil {
		return nil, err
	}
	r, err := c.API.GetAccountWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
		return nil, err
	}
	if r.JSON200 == nil {
		return nil, errors.New("empty account response")
	}
	return &account{id: output.S(r.JSON200.AccountId), name: output.S(r.JSON200.AccountName)}, nil
}

func orUnknown(s string) string {
	if s == "" {
		return "(unknown account)"
	}
	return s
}

func (a *App) authStatus() error {
	pr, err := a.printer()
	if err != nil {
		return err
	}
	if strings.TrimSpace(a.Getenv("CLOUDAFRICA_TOKEN")) != "" {
		ep := a.envEndpoint()
		if ep == "" {
			ep = DefaultEndpoint
		}
		elevation := strings.TrimSpace(a.Getenv(elevationEnv)) != ""
		v := map[string]any{"credential": "CLOUDAFRICA_TOKEN", "endpoint": ep, "elevation": elevation}
		rows := [][]string{{"Credential", "CLOUDAFRICA_TOKEN (overrides every profile)"}}
		if ep != DefaultEndpoint {
			rows = append(rows, []string{"Endpoint", ep})
		}
		set := "not set"
		if elevation {
			set = elevationEnv + " set"
		}
		rows = append(rows, []string{"Elevation", set})
		return pr.Print(output.Result{Value: v, Rows: rows})
	}
	name, err := a.profileName()
	if err != nil {
		return err
	}
	p, err := a.Store.Load(name)
	if errors.Is(err, profile.ErrNotFound) {
		return &auth.LoginRequiredError{Profile: name, Reason: fmt.Sprintf("profile %q is not signed in", name)}
	}
	if err != nil {
		return err
	}
	v := map[string]any{
		"profile": name, "endpoint": p.Endpoint, "accountId": p.AccountID, "accountName": p.AccountName,
		"scope": p.Scope, "accessTokenExpiry": p.Expiry.UTC().Format(time.RFC3339),
		"refreshToken": p.RefreshToken != "",
	}
	rows := [][]string{
		{"Profile", name},
		{"Account", fmt.Sprintf("%s (%s)", orUnknown(p.AccountName), p.AccountID)},
		{"Scopes", p.Scope},
		{"Access token expires", p.Expiry.Local().Format(time.RFC3339)},
	}
	if p.Endpoint != DefaultEndpoint {
		rows = append(rows, []string{"Endpoint", p.Endpoint})
	}
	if p.RefreshToken == "" {
		rows = append(rows, []string{"Session", "no refresh token: sign in again when the access token expires"})
	}
	return pr.Print(output.Result{Value: v, Rows: rows, IDs: []string{name}})
}

func (a *App) authList() error {
	pr, err := a.printer()
	if err != nil {
		return err
	}
	selected, err := a.profileName()
	if err != nil {
		return err
	}
	names, err := a.Store.Names()
	if err != nil {
		return err
	}
	type row struct {
		Profile      string `json:"profile"`
		Selected     bool   `json:"selected"`
		AccountID    string `json:"accountId,omitempty"`
		AccountName  string `json:"accountName,omitempty"`
		Endpoint     string `json:"endpoint,omitempty"`
		RefreshToken bool   `json:"refreshToken"`
		Error        string `json:"error,omitempty"`
	}
	res := output.Result{Headers: []string{"", "PROFILE", "ACCOUNT", "ENDPOINT", "REFRESH TOKEN"}}
	var rows []row
	for _, n := range names {
		r := row{Profile: n, Selected: n == selected}
		p, err := a.Store.Load(n)
		if err != nil {
			r.Error = err.Error()
		} else {
			r.AccountID, r.AccountName, r.Endpoint, r.RefreshToken = p.AccountID, p.AccountName, p.Endpoint, p.RefreshToken != ""
		}
		rows = append(rows, r)
		mark, acct, refresh := "", r.AccountName, "yes"
		if r.Selected {
			mark = "*"
		}
		if acct == "" {
			acct = r.AccountID
		}
		if !r.RefreshToken {
			refresh = "no"
		}
		if r.Error != "" {
			acct, refresh = "(unreadable: "+r.Error+")", ""
		}
		res.Rows = append(res.Rows, []string{mark, n, acct, r.Endpoint, refresh})
		res.IDs = append(res.IDs, n)
	}
	if rows == nil {
		rows = []row{}
	}
	res.Value = rows
	if len(names) == 0 && pr.Format == output.Table && !pr.Quiet {
		fmt.Fprintln(a.Stderr, "No profiles. Run `ca auth login`.")
		return nil
	}
	return pr.Print(res)
}

func (a *App) logout(ctx context.Context) error {
	name, err := a.profileName()
	if err != nil {
		return err
	}
	// Under the profile lock, so a refresh in another process cannot save
	// the profile back after it is deleted.
	release, err := a.Store.Lock(ctx, name)
	if err != nil {
		return err
	}
	defer release()
	p, err := a.Store.Load(name)
	switch {
	case errors.Is(err, profile.ErrNotFound):
		fmt.Fprintf(a.Stderr, "Profile %q is not signed in.\n", name)
		return nil
	case errors.Is(err, profile.ErrCorrupt):
		fmt.Fprintf(a.Stderr, "Warning: %v; deleting it. Any session it held stays listed under Connected apps "+
			"in the portal.\n", err)
		p = nil
	case err != nil:
		return err
	}
	if p != nil && p.RefreshToken != "" {
		if err := a.revokeOld(ctx, p); err != nil {
			return fmt.Errorf("revoking the session failed, so the profile was kept: run logout again, remove it "+
				"under Connected apps in the portal, or delete %s: %w", a.Store.Path(name), err)
		}
	}
	if err := a.Store.Delete(name); err != nil {
		return err
	}
	fmt.Fprintf(a.Stderr, "Signed out of profile %q.\n", name)
	return nil
}

var noArgs = func(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return usageErr(fmt.Errorf("%s takes no arguments", cmd.CommandPath()))
	}
	return nil
}

// group is a command that only holds subcommands; an unknown subcommand is a usage error.
func group(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageErr(fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath()))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
}

// requireFlags is MarkFlagRequired with a usage error (exit 2).
func requireFlags(cmd *cobra.Command, names ...string) error {
	var missing []string
	for _, n := range names {
		if !cmd.Flags().Changed(n) {
			missing = append(missing, "--"+n)
		}
	}
	if len(missing) > 0 {
		return usageErr(fmt.Errorf("missing required flags: %s", strings.Join(missing, ", ")))
	}
	return nil
}

func exactArgs(n int, names string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return usageErr(fmt.Errorf("usage: %s %s", cmd.CommandPath(), names))
		}
		return nil
	}
}

// decodeJSON reads an API response body into v after checking its status.
func decodeJSON(resp *http.Response, v any) error {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := cloudafrica.CheckResponse(resp, data); err != nil {
		return err
	}
	if v == nil {
		return nil
	}
	if len(data) == 0 {
		return fmt.Errorf("HTTP %d with an empty body", resp.StatusCode)
	}
	return json.Unmarshal(data, v)
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
