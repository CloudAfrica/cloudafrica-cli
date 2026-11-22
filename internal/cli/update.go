package cli

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/CloudAfrica/cloudafrica-cli/internal/update"
)

func (a *App) updateCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update ca to the latest release",
		Long: "Update ca to the latest release on GitHub, after checking the download against the release's SHA256SUMS.\n" +
			"A ca installed with Homebrew or Scoop is updated by that package manager; ca names the command.",
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			exe, err := a.executable()
			if err != nil {
				return err
			}
			res, err := update.Run(cmd.Context(), update.Options{
				Exe:          exe,
				GOOS:         runtime.GOOS,
				GOARCH:       runtime.GOARCH,
				Current:      a.Version,
				Check:        check,
				APIURL:       update.LatestReleaseURL,
				DownloadBase: update.DownloadBase,
				Client:       a.releaseHTTP(),
				UserAgent:    "ca/" + a.Version,
				Getenv:       a.Getenv,
			})
			var refused *update.RefusedError
			if errors.As(err, &refused) {
				return usageErr(err)
			}
			if err != nil {
				return err
			}
			a.reportUpdate(res)
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "only report whether a newer release exists")
	return cmd
}

func (a *App) reportUpdate(res update.Result) {
	switch res.State {
	case update.UpToDate:
		fmt.Fprintf(a.Stdout, "ca %s is the latest release\n", res.Current)
	case update.Ahead:
		fmt.Fprintf(a.Stdout, "ca %s is newer than the latest release %s\n", res.Current, res.Latest)
	case update.Available:
		fmt.Fprintf(a.Stdout, "ca %s is available (this is %s)\n", res.Latest, res.Current)
	case update.Updated:
		fmt.Fprintf(a.Stdout, "Updated ca %s -> %s\n", res.Current, res.Latest)
	}
	if res.Managed != "" && res.State == update.Available {
		fmt.Fprintf(a.Stdout, "ca was installed by a package manager; run '%s'\n", res.Managed)
	}
	if res.SiblingErr != nil {
		fmt.Fprintf(a.Stderr, "Warning: %v\n", res.SiblingErr)
	}
}

// updateCommand is what the version notice tells the user to run.
func (a *App) updateCommand() string {
	if exe, err := a.executable(); err == nil {
		if m := update.Managed(exe, runtime.GOOS, a.Getenv); m != "" {
			return m
		}
	}
	return "ca update"
}

// executable is the running binary with symlinks resolved, so a Homebrew
// link resolves into the Cellar.
func (a *App) executable() (string, error) {
	find := a.Executable
	if find == nil {
		find = os.Executable
	}
	exe, err := find()
	if err != nil {
		return "", fmt.Errorf("finding the running ca: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", exe, err)
	}
	return resolved, nil
}

// releaseHTTP reaches GitHub: a.Transport for tests, and never the
// elevation transport, so no credential leaves for GitHub.
func (a *App) releaseHTTP() *http.Client {
	return &http.Client{Transport: a.Transport}
}
