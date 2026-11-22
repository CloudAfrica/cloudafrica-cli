// Package update finds the latest release of this CLI and replaces the
// running binary with it (`ca update`).
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// LatestReleaseURL is the GitHub API document naming the newest release;
// GitHub leaves prereleases out of it.
var LatestReleaseURL = "https://api.github.com/repos/CloudAfrica/cloudafrica-cli/releases/latest"

// DownloadBase is where a release's assets are, under /<tag>/.
var DownloadBase = "https://github.com/CloudAfrica/cloudafrica-cli/releases/download"

// ErrNoRelease means GitHub names no usable release.
var ErrNoRelease = errors.New("no release found")

// Latest returns the newest release tag (such as v0.4.0). The request carries
// only Accept and User-Agent, so give it a client whose transport adds no
// credentials.
func Latest(ctx context.Context, client *http.Client, url, userAgent string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("asking GitHub for the latest release: %w", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return "", ErrNoRelease
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("asking GitHub for the latest release: %s", resp.Status)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return "", fmt.Errorf("reading GitHub's latest release: %w", err)
	}
	if _, ok := parseVersion(rel.TagName); !ok {
		return "", fmt.Errorf("%w (got %q)", ErrNoRelease, rel.TagName)
	}
	return rel.TagName, nil
}
