package profile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Lock takes the profile's cross-process lock and returns its release. The
// server burns a refresh token on use and refuses (or, outside a short grace,
// revokes the whole grant) when a used one is presented again, so two `ca`
// processes must never refresh the same profile at once: the holder
// refreshes, the waiter then re-reads the profile. Login and logout take it
// too, so a refresh in flight cannot resurrect or overwrite their write.
//
// It is an OS advisory lock (flock, LockFileEx) on <name>.lock, so the kernel
// drops it when the holder exits, however it exits; there is no staleness to
// guess. The file itself is left in place.
func (s Store) Lock(ctx context.Context, name string) (func(), error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(s.Dir, name+".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	delay := 20 * time.Millisecond
	for {
		ok, err := tryLock(f)
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("locking profile %s (%s): %w", name, path, err)
		}
		if ok {
			return func() {
				_ = unlock(f)
				_ = f.Close()
			}, nil
		}
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			_ = f.Close()
			return nil, fmt.Errorf("waiting for the lock on profile %s (%s): %w", name, path, ctx.Err())
		case <-t.C:
		}
		delay = min(delay*2, 250*time.Millisecond)
	}
}
