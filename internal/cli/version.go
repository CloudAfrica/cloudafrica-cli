package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/update"
)

const updateCheckEvery = 24 * time.Hour

func (a *App) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(a.Stdout, "ca %s (cloudafrica-go %s, %s/%s)\n", a.Version, cloudafrica.Version, runtime.GOOS, runtime.GOARCH)
			if a.TTY {
				if latest := a.latestRelease(cmd.Context()); latest != "" && update.Newer(latest, a.Version) {
					fmt.Fprintf(a.Stderr, "A newer release is available: %s. Run '%s'.\n", latest, a.updateCommand())
				}
			}
			return nil
		},
	}
}

type updateCache struct {
	CheckedAt time.Time `json:"checkedAt"`
	Latest    string    `json:"latest"`
}

// latestRelease returns the newest release tag, asking GitHub at most once a
// day: a failed ask is cached too, so an unreachable GitHub costs one try a
// day. With no CacheDir there is no way to keep that promise, so it does not
// ask. Any failure yields "": the notice is a courtesy, never an error. The
// cache lives under CacheDir, away from the profiles.
func (a *App) latestRelease(ctx context.Context) string {
	if a.Version == "" || a.Version == "dev" || a.CacheDir == "" {
		return ""
	}
	path := filepath.Join(a.CacheDir, "update-check.json")
	var c updateCache
	if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &c) == nil && time.Since(c.CheckedAt) < updateCheckEvery {
		return c.Latest
	}
	latest := a.fetchLatest(ctx)
	c = updateCache{CheckedAt: time.Now(), Latest: latest}
	if data, err := json.Marshal(c); err == nil && os.MkdirAll(a.CacheDir, 0o700) == nil {
		_ = os.WriteFile(path, data, 0o600)
	}
	return latest
}

func (a *App) fetchLatest(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	latest, err := update.Latest(ctx, a.releaseHTTP(), update.LatestReleaseURL, "ca/"+a.Version)
	if err != nil {
		return ""
	}
	return latest
}
