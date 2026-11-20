package profile

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestResolvePrecedence(t *testing.T) {
	env := func(v string) func(string) string { return func(string) string { return v } }
	for _, c := range []struct{ flag, env, want string }{
		{"", "", Default},
		{"", "work", "work"},
		{"home", "work", "home"},
	} {
		got, err := Resolve(c.flag, env(c.env))
		if err != nil || got != c.want {
			t.Errorf("Resolve(%q, %q) = %q, %v; want %q", c.flag, c.env, got, err, c.want)
		}
	}
	for _, bad := range []string{"../x", "a/b", ".hidden", "", "x.json"} {
		if err := ValidateName(bad); err == nil {
			t.Errorf("ValidateName(%q) accepted", bad)
		}
	}
	if _, err := Resolve("", func(string) string { return "../etc" }); err == nil {
		t.Error("CLOUDAFRICA_PROFILE traversal accepted")
	}
}

func TestSaveReplacesAtomicallyAndLeavesNoTemp(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	for _, tok := range []string{"a", "b"} {
		if err := s.Save("default", &Profile{Endpoint: "https://x", AccessToken: tok}); err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.Load("default")
	if err != nil || p.AccessToken != "b" {
		t.Fatalf("load = %+v, %v", p, err)
	}
	entries, _ := os.ReadDir(s.Dir)
	if len(entries) != 1 {
		t.Errorf("dir holds %d entries, want only default.json", len(entries))
	}
	names, _ := s.Names()
	if len(names) != 1 || names[0] != "default" {
		t.Errorf("names = %v", names)
	}
}

func TestLockExcludes(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	release, err := s.Lock(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := s.Lock(ctx, "default"); err == nil {
		t.Fatal("second lock taken while the first is held")
	}
	release()
	r2, err := s.Lock(context.Background(), "default")
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	r2()
}

// Many contenders: never more than one inside at a time, and all get in.
func TestLockSerialisesContenders(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	var inside, maxInside, done atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := s.Lock(context.Background(), "default")
			if err != nil {
				t.Error(err)
				return
			}
			n := inside.Add(1)
			for {
				m := maxInside.Load()
				if n <= m || maxInside.CompareAndSwap(m, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			inside.Add(-1)
			done.Add(1)
			release()
		}()
	}
	wg.Wait()
	if maxInside.Load() != 1 || done.Load() != 8 {
		t.Errorf("max inside = %d, done = %d", maxInside.Load(), done.Load())
	}
}

// A holder that exits without releasing leaves no lock behind: the child
// process takes the lock and exits, and the parent then gets it at once.
func TestLockFreedWhenHolderDies(t *testing.T) {
	if dir := os.Getenv("CA_LOCK_HOLDER_DIR"); dir != "" {
		if _, err := (Store{Dir: dir}).Lock(context.Background(), "default"); err != nil {
			os.Exit(3)
		}
		os.Exit(0)
	}
	s := Store{Dir: t.TempDir()}
	cmd := exec.Command(os.Args[0], "-test.run=^TestLockFreedWhenHolderDies$")
	cmd.Env = append(os.Environ(), "CA_LOCK_HOLDER_DIR="+s.Dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("holder: %v %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "default.lock")); err != nil {
		t.Fatalf("holder left no lock file: %v", err)
	}
	f, err := os.OpenFile(filepath.Join(s.Dir, "default.lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if ok, err := tryLock(f); !ok {
		t.Fatalf("lock still held after the holder exited: %v", err)
	}
}

func TestLoadCorruptIsErrCorruptNamingThePath(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if err := os.WriteFile(s.Path("default"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.Load("default")
	if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), s.Path("default")) {
		t.Errorf("err = %v", err)
	}
}
