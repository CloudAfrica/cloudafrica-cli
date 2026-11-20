package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"
)

// LatestReleaseURL is the GitHub API document naming the newest release.
var LatestReleaseURL = "https://api.github.com/repos/CloudAfrica/cloudafrica-cli/releases/latest"

const updateCheckEvery = 24 * time.Hour

func (a *App) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(a.Stdout, "ca %s (cloudafrica-go %s, %s/%s)\n", a.Version, cloudafrica.Version, runtime.GOOS, runtime.GOARCH)
			if a.TTY {
				if latest := a.latestRelease(cmd.Context()); latest != "" && newer(latest, a.Version) {
					fmt.Fprintf(a.Stderr, "A newer release is available: %s (https://github.com/CloudAfrica/cloudafrica-cli/releases)\n", latest)
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
// day. Any failure yields "": the notice is a courtesy, never an error.
func (a *App) latestRelease(ctx context.Context) string {
	if a.Version == "" || a.Version == "dev" {
		return ""
	}
	path := filepath.Join(a.Store.Dir, "update-check.json")
	var c updateCache
	if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &c) == nil && time.Since(c.CheckedAt) < updateCheckEvery {
		return c.Latest
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestReleaseURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ca/"+a.Version)
	hc := &http.Client{Transport: a.Transport}
	resp, err := hc.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel) != nil {
		return ""
	}
	c = updateCache{CheckedAt: time.Now(), Latest: rel.TagName}
	if data, err := json.Marshal(c); err == nil && os.MkdirAll(a.Store.Dir, 0o700) == nil {
		_ = os.WriteFile(path, data, 0o600)
	}
	return rel.TagName
}

// newer reports whether release version a is newer than b. Both are
// vMAJOR.MINOR.PATCH with an optional -prerelease; a prerelease sorts before
// its release. Anything unparsable is never newer.
func newer(a, b string) bool {
	pa, oka := parseVersion(a)
	pb, okb := parseVersion(b)
	if !oka || !okb {
		return false
	}
	for i := range 3 {
		if pa.n[i] != pb.n[i] {
			return pa.n[i] > pb.n[i]
		}
	}
	switch {
	case pa.pre == pb.pre:
		return false
	case pa.pre == "":
		return true
	case pb.pre == "":
		return false
	}
	return pa.pre > pb.pre
}

type version struct {
	n   [3]int
	pre string
}

func parseVersion(s string) (version, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	var v version
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, v.pre = s[:i], s[i+1:]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v.n[i] = n
	}
	return v, true
}
