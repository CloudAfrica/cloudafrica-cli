package profile

import (
	"context"
	"os"
	"path/filepath"
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
		if err := ValidateName(bad); err == nil && bad != "" {
			t.Errorf("ValidateName(%q) accepted", bad)
		}
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

func TestLockExcludesAndBreaksStale(t *testing.T) {
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

	// An abandoned lock (its holder died) is broken once it is stale.
	path := filepath.Join(s.Dir, "default.lock")
	if err := os.WriteFile(path, []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * staleLock)
	_ = os.Chtimes(path, old, old)
	r3, err := s.Lock(context.Background(), "default")
	if err != nil {
		t.Fatalf("stale lock not broken: %v", err)
	}
	r3()
}
