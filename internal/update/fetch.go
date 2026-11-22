package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const maxDownload = 512 << 20

var errNotListed = errors.New("not listed in")

// fetch downloads the release archive for o.GOOS/o.GOARCH and its SHA256SUMS,
// checks the digest, and extracts names into dir.
func fetch(ctx context.Context, o Options, tag, version, dir string, names []string) error {
	ext := "tar.gz"
	if o.GOOS == "windows" {
		ext = "zip"
	}
	archive := fmt.Sprintf("cloudafrica-cli_%s_%s_%s.%s", version, o.GOOS, o.GOARCH, ext)
	sumsName := fmt.Sprintf("cloudafrica-cli_%s_SHA256SUMS", version)
	base := strings.TrimRight(o.DownloadBase, "/") + "/" + tag + "/"

	var sums strings.Builder
	if _, err := download(ctx, o, base+sumsName, &sums); err != nil {
		return err
	}
	want, ok := digestOf(sums.String(), archive)
	if !ok {
		return fmt.Errorf("%s is %w %s", archive, errNotListed, sumsName)
	}

	path := filepath.Join(dir, archive)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	got, err := download(ctx, o, base+archive, f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("checksum mismatch for %s", archive)
	}
	if o.GOOS == "windows" {
		return unzip(path, dir, names)
	}
	return untar(path, dir, names)
}

// download writes url's body to w and returns its hex SHA-256.
func download(ctx context.Context, o Options, url string, w io.Writer) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", o.UserAgent)
	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading %s: %s", url, resp.Status)
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(w, h), io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", url, err)
	}
	if n > maxDownload {
		return "", fmt.Errorf("downloading %s: larger than %d bytes", url, maxDownload)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// digestOf finds name in a SHA256SUMS body ("<hex>  <name>" per line).
func digestOf(sums, name string) (string, bool) {
	sc := bufio.NewScanner(strings.NewReader(sums))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			return strings.ToLower(f[0]), true
		}
	}
	return "", false
}

// member checks an archive entry name: the release archives are flat, so a
// path is refused rather than resolved.
func member(name string) error {
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("unexpected archive member %q", name)
	}
	return nil
}

func untar(path, dir string, names []string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}
	tr := tar.NewReader(gz)
	found := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading %s: %w", filepath.Base(path), err)
		}
		if err := member(h.Name); err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg || !slices.Contains(names, h.Name) {
			continue
		}
		if err := write(filepath.Join(dir, h.Name), tr); err != nil {
			return err
		}
		found++
	}
	return complete(path, found, names)
}

func unzip(path, dir string, names []string) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}
	defer zr.Close()
	found := 0
	for _, zf := range zr.File {
		if err := member(zf.Name); err != nil {
			return err
		}
		if zf.FileInfo().IsDir() || !slices.Contains(names, zf.Name) {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		err = write(filepath.Join(dir, zf.Name), rc)
		rc.Close()
		if err != nil {
			return err
		}
		found++
	}
	return complete(path, found, names)
}

func complete(path string, found int, names []string) error {
	if found != len(names) {
		return fmt.Errorf("%s does not hold %s", filepath.Base(path), strings.Join(names, " and "))
	}
	return nil
}

func write(path string, r io.Reader) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(r, maxDownload+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxDownload {
		err = fmt.Errorf("%s is larger than %d bytes", filepath.Base(path), maxDownload)
	}
	return err
}
