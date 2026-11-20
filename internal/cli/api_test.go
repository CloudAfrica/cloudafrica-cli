package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRawAPIDataForms(t *testing.T) {
	file := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(file, []byte(`{"name":"from-file"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		data, stdin, want string
	}{
		{`{"name":"literal"}`, "", "literal"},
		{"@" + file, "", "from-file"},
		{"@-", `{"name":"from-stdin"}`, "from-stdin"},
	} {
		h := signedIn(t)
		h.stdin = c.stdin
		h.mustRun("api", "POST", "/v1/vms/vm-1/actions/rename", "--data", c.data)
		if got := h.fake.Bodies[0]["name"]; got != c.want {
			t.Errorf("--data %s: body name = %v", c.data, got)
		}
		if ct := h.fake.Writes[0].Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("--data %s: Content-Type = %q", c.data, ct)
		}
	}
}

func TestRawAPIProblemGoesToStdoutAndExits1(t *testing.T) {
	h := signedIn(t)
	if code := h.run("api", "GET", "/v1/vms/missing"); code != ExitAPI {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(h.stdout.String(), "not-found") {
		t.Errorf("stdout = %s", h.stdout)
	}
}

func TestRawAPIUsageErrors(t *testing.T) {
	h := signedIn(t)
	for _, args := range [][]string{
		{"api", "GET", "v1/vms"},
		{"api", "POST", "/v1/vms", "--data", "  "},
		{"api", "POST", "/v1/vms", "--data", "@/does/not/exist"},
	} {
		if code := h.run(args...); code != ExitUsage {
			t.Errorf("%v: exit %d, want %d", args, code, ExitUsage)
		}
	}
	if len(h.fake.Writes) != 0 {
		t.Errorf("writes = %d", len(h.fake.Writes))
	}
}

func TestRawAPIBodilessAcceptedPrintsLocation(t *testing.T) {
	h := signedIn(t)
	h.fake.OmitOperationID = true
	h.mustRun("api", "POST", "/v1/vms/vm-1/actions/reboot")
	if got := strings.TrimSpace(h.stdout.String()); got != `{"location":"/v1/operations/op/reboot-vm-1"}` {
		t.Errorf("stdout = %s", got)
	}
}
