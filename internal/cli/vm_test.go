package cli

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

func signedIn(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.seed("default", compute(), time.Now().Add(time.Hour))
	return h
}

var createArgs = []string{"vm", "create", "--name", "web", "--datacenter", "jb1", "--image", "img", "--vpc", "vpc-1",
	"--vcpu", "2", "--ram-gib", "4", "--boot-disk-gib", "20", "--accept-terms"}

func TestCreateSendsEveryFlagInTheBody(t *testing.T) {
	h := signedIn(t)
	h.mustRun(append(createArgs, "--public-ip=false", "--ssh-key", "k1", "--ssh-key", "k2",
		"--extra-disk-gib", "10", "--extra-disk-gib", "20", "--placement-group", "pg-1", "--reserved-ip", "rip-1")...)
	want := map[string]any{
		"name": "web", "datacenterSlug": "jb1", "osImageId": "img", "vpcResourceId": "vpc-1",
		"vcpu": 2.0, "ramMib": 4096.0, "bootDiskGib": 20.0, "termsAccepted": true,
		"publicIp": false, "sshKeyIds": []any{"k1", "k2"}, "extraDisksGib": []any{10.0, 20.0},
		"placementGroupId": "pg-1", "reservedPublicIpId": "rip-1",
	}
	if got := h.fake.Bodies[0]; !reflect.DeepEqual(got, want) {
		t.Errorf("body = %v\nwant   %v", got, want)
	}
}

func TestCreateOmitsUnsetOptionals(t *testing.T) {
	h := signedIn(t)
	h.mustRun(createArgs...)
	body := h.fake.Bodies[0]
	for _, k := range []string{"publicIp", "sshKeyIds", "extraDisksGib", "placementGroupId", "reservedPublicIpId"} {
		if _, ok := body[k]; ok {
			t.Errorf("%s sent though its flag was not given: %v", k, body)
		}
	}
}

func TestResizeSendsOnlyTheGivenFields(t *testing.T) {
	for _, c := range []struct {
		args []string
		want map[string]any
	}{
		{[]string{"--vcpu", "4"}, map[string]any{"cpus": 4.0}},
		{[]string{"--ram-gib", "8"}, map[string]any{"ramMib": 8192.0}},
		{[]string{"--vcpu", "4", "--ram-gib", "8"}, map[string]any{"cpus": 4.0, "ramMib": 8192.0}},
	} {
		h := signedIn(t)
		h.mustRun(append([]string{"vm", "resize", "vm-1"}, c.args...)...)
		if h.fake.WriteURLs[0] != "POST /v1/vms/vm-1/actions/resize" || !reflect.DeepEqual(h.fake.Bodies[0], c.want) {
			t.Errorf("%v: %s %v, want %v", c.args, h.fake.WriteURLs[0], h.fake.Bodies[0], c.want)
		}
	}
	h := signedIn(t)
	if code := h.run("vm", "resize", "vm-1"); code != ExitUsage || len(h.fake.Writes) != 0 {
		t.Errorf("resize with neither flag: exit %d, writes %d", code, len(h.fake.Writes))
	}
}

func TestActionsHitTheirEndpoints(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"start", "vm-1"}, "POST /v1/vms/vm-1/actions/start"},
		{[]string{"shutdown", "vm-1"}, "POST /v1/vms/vm-1/actions/shutdown"},
		{[]string{"shutdown", "vm-1", "--guest-timeout", "30"}, "POST /v1/vms/vm-1/actions/shutdown?timeout=30"},
		{[]string{"reboot", "vm-1"}, "POST /v1/vms/vm-1/actions/reboot"},
		{[]string{"force-stop", "vm-1"}, "POST /v1/vms/vm-1/actions/force-stop"},
		{[]string{"rename", "vm-1", "db"}, "POST /v1/vms/vm-1/actions/rename"},
	} {
		h := signedIn(t)
		h.mustRun(append(append([]string{"vm"}, c.args...), "-q")...)
		if len(h.fake.WriteURLs) != 1 || h.fake.WriteURLs[0] != c.want {
			t.Errorf("%v: writes %v, want %s", c.args, h.fake.WriteURLs, c.want)
		}
		if strings.TrimSpace(h.stdout.String()) == "" {
			t.Errorf("%v: no operation id printed", c.args)
		}
		if c.args[0] == "rename" && h.fake.Bodies[0]["name"] != "db" {
			t.Errorf("rename body = %v", h.fake.Bodies[0])
		}
	}
}

func TestGetPrintsTheVM(t *testing.T) {
	h := signedIn(t)
	h.fake.VMs = []map[string]any{{"id": "vm-1", "name": "web"}, {"id": "vm-2", "name": "db"}}
	h.mustRun("vm", "get", "vm-2", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "vm-2" {
		t.Errorf("stdout = %q", got)
	}
}

func TestDeleteOnTerminalAsksFirst(t *testing.T) {
	h := newHarness(t)
	h.seed("default", &fakeapi.Grant{Scopes: []string{"read:compute", "write:compute", "protected:compute"}, Elevated: true},
		time.Now().Add(time.Hour))
	h.tty = true
	h.stdin = "n\n"
	if code := h.run("vm", "delete", "vm-1"); code == ExitOK || h.fake.DeleteCalls != 0 {
		t.Fatalf("declined: exit %d, delete calls %d", code, h.fake.DeleteCalls)
	}
	if !strings.Contains(h.stderr.String(), "Delete VM vm-1?") {
		t.Errorf("no prompt: %s", h.stderr)
	}
	h.stdin = "y\n"
	h.mustRun("vm", "delete", "vm-1")
	if h.fake.DeleteCalls != 1 {
		t.Errorf("confirmed: delete calls %d", h.fake.DeleteCalls)
	}
}

// A 202 whose body lacks the operation id falls back to the Location header,
// which carries a slashed id.
func TestAcceptedWithoutBodyUsesLocation(t *testing.T) {
	h := signedIn(t)
	h.fake.OmitOperationID = true
	h.mustRun("vm", "start", "vm-1", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "op/start-vm-1" {
		t.Errorf("operation id = %q", got)
	}
	h.mustRun("vm", "start", "vm-1", "--wait", "-o", "json")
	if !strings.Contains(h.stdout.String(), `"op/start-vm-1"`) || !strings.Contains(h.stdout.String(), "complete") {
		t.Errorf("wait output = %s", h.stdout)
	}
}

func TestOperationIDFromLocation(t *testing.T) {
	for loc, want := range map[string]string{
		"/v1/operations/op/x":                            "op/x",
		"https://api.cloudafrica.net/v1/operations/op-1": "op-1",
		"/v1/operations/":                                "",
		"/v1/vms/vm-1":                                   "",
		"":                                               "",
	} {
		got, ok := operationIDFromLocation(loc)
		if got != want || ok != (want != "") {
			t.Errorf("operationIDFromLocation(%q) = %q, %v", loc, got, ok)
		}
	}
}

func TestOperationGetEmptyBodyIsAnError(t *testing.T) {
	h := signedIn(t)
	h.fake.EmptyOperationBody = true
	if code := h.run("operation", "get", "op/x"); code != ExitAPI || !strings.Contains(h.stderr.String(), "empty body") {
		t.Errorf("exit %d, stderr %s", code, h.stderr)
	}
}

func TestBadOutputIsRejectedBeforeTheWrite(t *testing.T) {
	h := newHarness(t)
	h.seed("default", &fakeapi.Grant{Scopes: []string{"read:compute", "write:compute", "protected:compute"}, Elevated: true},
		time.Now().Add(time.Hour))
	if code := h.run("vm", "delete", "vm-1", "--yes", "-o", "xml"); code != ExitUsage || h.fake.DeleteCalls != 0 {
		t.Errorf("exit %d, delete calls %d", code, h.fake.DeleteCalls)
	}
}

func TestRAMOutOfRangeIsRejected(t *testing.T) {
	for _, args := range [][]string{
		append(append([]string{}, createArgs...), "--ram-gib", "18014398509481988"),
		{"vm", "resize", "vm-1", "--ram-gib", "4194308"},
		{"vm", "resize", "vm-1", "--ram-gib", "0"},
	} {
		h := signedIn(t)
		if code := h.run(args...); code != ExitUsage || len(h.fake.Writes) != 0 {
			t.Errorf("%v: exit %d, writes %d", args[len(args)-1], code, len(h.fake.Writes))
		}
	}
}
