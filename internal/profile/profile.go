// Package profile stores one signed-in account per file under the user's
// config directory. A file holds secrets, so it is written 0600 and replaced
// atomically.
package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Default is the profile used when neither --profile nor CLOUDAFRICA_PROFILE names one.
const Default = "default"

// Profile is one account's sign-in. One profile is one account: a grant is
// bound to the single account picked on the consent page.
type Profile struct {
	Endpoint     string    `json:"endpoint"`
	AccountID    string    `json:"accountId,omitempty"`
	AccountName  string    `json:"accountName,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	AccessToken  string    `json:"accessToken"`
	Expiry       time.Time `json:"expiry"`
	RefreshToken string    `json:"refreshToken,omitempty"`
}

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// ErrNotFound is returned by Load for a profile that has no file.
var ErrNotFound = errors.New("profile not found")

// Store is a directory of profile files.
type Store struct {
	Dir string
}

// DefaultDir is $XDG_CONFIG_HOME/cloudafrica, ~/.config/cloudafrica, or
// %AppData%\cloudafrica on Windows (os.UserConfigDir).
func DefaultDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cloudafrica"), nil
}

// ValidateName refuses a name that is not a plain file stem.
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: use letters, digits, '_' or '-', starting with a letter or digit", name)
	}
	return nil
}

func (s Store) path(name string) string { return filepath.Join(s.Dir, name+".json") }

// Load reads a profile; ErrNotFound when it has no file.
func (s Store) Load(name string) (*Profile, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.path(name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("profile %s is corrupt: %w", name, err)
	}
	return &p, nil
}

// Save writes p atomically: a 0600 temp file in the same directory, synced,
// then renamed over the old file. A crash leaves the old file or the new one,
// never a torn one.
func (s Store) Save(name string, p *Profile) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	// CreateTemp opens the file 0600.
	f, err := os.CreateTemp(s.Dir, name+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path(name)); err != nil {
		return err
	}
	ok = true
	return nil
}

// Delete removes a profile; a missing one is not an error.
func (s Store) Delete(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	err := os.Remove(s.path(name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// Names lists the stored profiles, sorted.
func (s Store) Names() ([]string, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".json") {
			continue
		}
		stem := strings.TrimSuffix(n, ".json")
		if ValidateName(stem) == nil {
			names = append(names, stem)
		}
	}
	sort.Strings(names)
	return names, nil
}

// Resolve picks the profile name: the flag, then CLOUDAFRICA_PROFILE, then Default.
func Resolve(flag string, getenv func(string) string) (string, error) {
	name := flag
	if name == "" {
		name = getenv("CLOUDAFRICA_PROFILE")
	}
	if name == "" {
		name = Default
	}
	if err := ValidateName(name); err != nil {
		return "", err
	}
	return name, nil
}
