package profile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// staleLock is how old a lock file must be before another process may break
// it. A refresh is one HTTP round trip, so a holder this old has died.
const staleLock = 30 * time.Second

// Lock takes the profile's cross-process lock and returns its release. The
// server burns a refresh token on use and refuses (or, outside a short grace,
// revokes the whole grant) when a used one is presented again, so two `ca`
// processes must never refresh the same profile at once: the holder
// refreshes, the waiter then re-reads the profile.
//
// The lock is a file created with O_EXCL, which behaves the same on every OS;
// one older than staleLock is taken to be abandoned and removed.
func (s Store) Lock(ctx context.Context, name string) (func(), error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(s.Dir, name+".lock")
	delay := 20 * time.Millisecond
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if st, serr := os.Stat(path); serr == nil && time.Since(st.ModTime()) > staleLock {
			_ = os.Remove(path)
			continue
		}
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, fmt.Errorf("waiting for the lock on profile %s (%s): %w", name, path, ctx.Err())
		case <-t.C:
		}
		delay = min(delay*2, 250*time.Millisecond)
	}
}
