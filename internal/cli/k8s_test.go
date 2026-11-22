package cli

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

func elevated(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.seed("default", &fakeapi.Grant{Scopes: append([]string{}, defaultScopes...), Elevated: true}, time.Now().Add(time.Hour))
	return h
}

func cluster(id string, upgrade map[string]any) map[string]any {
	c := map[string]any{"id": id, "name": "prod", "state": "ACTIVE", "kubernetesVersion": "v1.33.1", "workers": 3}
	if upgrade != nil {
		c["upgrade"] = upgrade
	}
	return c
}

func targets(versions ...string) []any {
	out := []any{}
	for _, v := range versions {
		out = append(out, map[string]any{"kubernetes": v, "talos": "v1.10.0"})
	}
	return out
}

func TestK8sListShowsTheWorkersUpgradeState(t *testing.T) {
	h := signedIn(t)
	h.fake.Clusters = []map[string]any{
		cluster("k-1", map[string]any{"workers": map[string]any{"state": "UPGRADE_AVAILABLE"}}),
		cluster("k-2", nil),
		cluster("k-3", map[string]any{"workers": map[string]any{"state": "CURRENT"}}),
	}
	h.mustRun("k8s", "list", "-o", "table")
	lines := strings.Split(strings.TrimSpace(h.stdout.String()), "\n")
	if len(lines) != 4 || !strings.Contains(lines[0], "UPGRADE") {
		t.Fatalf("stdout:\n%s", h.stdout)
	}
	for i, want := range []string{"UPGRADE_AVAILABLE", "-", "CURRENT"} {
		if f := strings.Fields(lines[i+1]); f[len(f)-1] != want {
			t.Errorf("row %d = %q, want UPGRADE %s", i+1, lines[i+1], want)
		}
	}
	h.mustRun("kubernetes", "list", "-q")
	if got := strings.Fields(h.stdout.String()); strings.Join(got, ",") != "k-1,k-2,k-3" {
		t.Errorf("alias ids = %v", got)
	}
}

func TestK8sGetPrintsTheDetailObject(t *testing.T) {
	h := signedIn(t)
	c := cluster("k-1", map[string]any{
		"controlPlane":            map[string]any{"state": "DEADLINE", "deadline": "2026-12-31"},
		"workers":                 map[string]any{"state": "UPGRADE_AVAILABLE"},
		"workerKubernetesVersion": "v1.32.4", "workerTalosVersion": "v1.9.5", "workerTargets": targets("v1.33.1"),
	})
	c["nodes"] = []any{
		map[string]any{"name": "w-1", "status": "running", "ready": true, "privateIp": "10.0.0.5"},
		map[string]any{"name": "w-2", "status": "running", "ready": false, "privateIp": "10.0.0.6"},
	}
	h.fake.Clusters = []map[string]any{c}
	h.mustRun("k8s", "get", "k-1", "-o", "json")
	var got map[string]any
	if err := json.Unmarshal(h.stdout.Bytes(), &got); err != nil || got["cluster"] == nil || got["nodes"] == nil {
		t.Fatalf("json = %s (%v)", h.stdout, err)
	}
	h.mustRun("k8s", "get", "k-1", "-o", "table")
	for _, want := range []string{"v1.33.1 DEADLINE (deadline 2026-12-31)", "v1.32.4 (Talos v1.9.5) UPGRADE_AVAILABLE",
		"v1.33.1 (Talos v1.10.0)", "running  ready  10.0.0.5", "running  not ready  10.0.0.6"} {
		if !strings.Contains(h.stdout.String(), want) {
			t.Errorf("table lacks %q:\n%s", want, h.stdout)
		}
	}
}

func TestK8sGetWithoutUpgradeOrTargets(t *testing.T) {
	h := signedIn(t)
	h.fake.Clusters = []map[string]any{cluster("k-1", nil), cluster("k-2", map[string]any{
		"workerTargets":      []any{},
		"controlPlaneWindow": map[string]any{"start": "2027-01-01", "end": "2027-01-15", "kubernetes": "v1.34.0"},
	})}
	h.mustRun("k8s", "get", "k-1", "-o", "table")
	if !strings.Contains(h.stdout.String(), "status unavailable") || strings.Contains(h.stdout.String(), "Worker targets") {
		t.Errorf("k-1:\n%s", h.stdout)
	}
	h.mustRun("k8s", "get", "k-2", "-o", "table")
	if !regexp.MustCompile(`(?m)^Worker targets +-$`).MatchString(h.stdout.String()) ||
		!strings.Contains(h.stdout.String(), "2027-01-01 to 2027-01-15, v1.34.0") {
		t.Errorf("k-2:\n%s", h.stdout)
	}
}

func TestUpgradeWorkersTakesTheNewestTarget(t *testing.T) {
	h := elevated(t)
	h.fake.Clusters = []map[string]any{cluster("k-1", map[string]any{"workerTargets": targets("v1.32.5", "v1.33.1")})}
	h.mustRun("k8s", "upgrade-workers", "k-1", "-q")
	if len(h.fake.WriteURLs) != 1 || h.fake.WriteURLs[0] != "POST /v1/kubernetes/clusters/k-1/actions/upgrade-workers" ||
		h.fake.Bodies[0]["targetKubernetes"] != "v1.33.1" {
		t.Fatalf("writes %v bodies %v", h.fake.WriteURLs, h.fake.Bodies)
	}
	if got := strings.TrimSpace(h.stdout.String()); got != "op/upgrade-k-1" {
		t.Errorf("stdout = %q", got)
	}
}

func TestUpgradeWorkersSkipsATargetWithNoVersion(t *testing.T) {
	h := elevated(t)
	h.fake.Clusters = []map[string]any{cluster("k-1", map[string]any{
		"workerTargets": append(targets("v1.32.5"), map[string]any{"talos": "v1.10.0"})})}
	h.mustRun("k8s", "upgrade-workers", "k-1", "-q")
	if len(h.fake.Bodies) != 1 || h.fake.Bodies[0]["targetKubernetes"] != "v1.32.5" {
		t.Errorf("bodies %v", h.fake.Bodies)
	}
}

func TestUpgradeWorkersWithNoTarget(t *testing.T) {
	for name, c := range map[string]struct {
		upgrade map[string]any
		code    int
		stderr  string
	}{
		"current": {map[string]any{"workers": map[string]any{"state": "CURRENT"}, "workerTargets": []any{}}, ExitOK,
			"workers are current"},
		"deadline, no target": {map[string]any{"workers": map[string]any{"state": "DEADLINE", "deadline": "2027-01-15"}},
			ExitOK, "workers: DEADLINE (deadline 2027-01-15); there is no worker target yet"},
		"upgrade available, no target": {map[string]any{"workerKubernetesVersion": "v1.32.4",
			"workers": map[string]any{"state": "UPGRADE_AVAILABLE"}}, ExitOK,
			"workers: v1.32.4 UPGRADE_AVAILABLE; there is no worker target yet"},
		"unknown": {map[string]any{"workers": map[string]any{"state": "UNKNOWN"}}, ExitAPI,
			"workers: UNKNOWN; there is no worker target; `ca k8s get k-1`"},
		"unsupported": {map[string]any{"workers": map[string]any{"state": "UNSUPPORTED"}}, ExitAPI,
			"workers: UNSUPPORTED; there is no worker target; `ca k8s get k-1`"},
		"no workers state": {map[string]any{"workers": map[string]any{}}, ExitAPI,
			"workers: -; there is no worker target; `ca k8s get k-1`"},
		"no workers":  {map[string]any{}, ExitAPI, "workers: -; there is no worker target; `ca k8s get k-1`"},
		"unavailable": {nil, ExitAPI, "upgrade status is unavailable; try again"},
	} {
		t.Run(name, func(t *testing.T) {
			h := elevated(t)
			h.fake.Clusters = []map[string]any{cluster("k-1", c.upgrade)}
			code := h.run("k8s", "upgrade-workers", "k-1")
			if code != c.code || len(h.fake.Writes) != 0 || !strings.Contains(h.stderr.String(), c.stderr) {
				t.Errorf("exit %d (want %d), writes %d, stderr %q", code, c.code, len(h.fake.Writes), h.stderr)
			}
			if h.stdout.Len() != 0 || strings.Contains(h.stderr.String(), "platform") {
				t.Errorf("stdout = %q, stderr %q", h.stdout, h.stderr)
			}
		})
	}
}

func TestUpgradeWorkersWaitWithNoTarget(t *testing.T) {
	for state, code := range map[string]int{"DEADLINE": ExitAPI, "UPGRADE_AVAILABLE": ExitAPI, "CURRENT": ExitOK} {
		h := elevated(t)
		h.fake.Clusters = []map[string]any{cluster("k-1", map[string]any{"workers": map[string]any{"state": state}})}
		if got := h.run("k8s", "upgrade-workers", "k-1", "--wait"); got != code || len(h.fake.Writes) != 0 ||
			(code == ExitAPI) != strings.Contains(h.stderr.String(), "nothing to wait for") {
			t.Errorf("%s: exit %d (want %d), writes %d, stderr %q", state, got, code, len(h.fake.Writes), h.stderr)
		}
	}
}

func TestUpgradeWorkersWaitsUpTo30Minutes(t *testing.T) {
	a := &App{}
	if got := a.k8sUpgradeWorkers().Flags().Lookup("timeout").DefValue; got != "30m0s" {
		t.Errorf("--timeout default = %s", got)
	}
}

func TestUpgradeWorkersVersionGetsItsV(t *testing.T) {
	for _, v := range []string{"1.37.2", "v1.37.2"} {
		h := elevated(t)
		h.mustRun("k8s", "upgrade-workers", "k-1", "--version", v, "-q")
		if len(h.fake.Bodies) != 1 || h.fake.Bodies[0]["targetKubernetes"] != "v1.37.2" {
			t.Errorf("--version %s: bodies %v", v, h.fake.Bodies)
		}
	}
}

func TestUpgradeWorkersPrintsTheConflictReason(t *testing.T) {
	h := elevated(t)
	h.fake.UpgradeConflict = "Another change is running on this cluster."
	if code := h.run("k8s", "upgrade-workers", "k-1", "--version", "v1.33.1"); code != ExitAPI ||
		!strings.Contains(h.stderr.String(), "Another change is running on this cluster.") {
		t.Errorf("exit %d, stderr %q", code, h.stderr)
	}
}

func TestUpgradeWorkersNotPairedExits5(t *testing.T) {
	h := newHarness(t)
	h.seed("default", protected(), time.Now().Add(time.Hour))
	if code := h.run("k8s", "upgrade-workers", "k-1", "--version", "v1.33.1"); code != ExitStepUp ||
		!strings.Contains(h.stderr.String(), "ca auth elevate") {
		t.Errorf("exit %d, stderr %q", code, h.stderr)
	}
}

func TestUpgradeWorkersWithNoOperationToFollow(t *testing.T) {
	for _, c := range []struct {
		wait bool
		code int
		want string
	}{
		{false, ExitOK, "There is no operation to follow; `ca k8s get k-1`"},
		{true, ExitAPI, "there is no operation to wait for; `ca k8s get k-1`"},
	} {
		h := elevated(t)
		h.fake.UpgradeNoOperation = true
		args := []string{"k8s", "upgrade-workers", "k-1", "--version", "v1.33.1"}
		if c.wait {
			args = append(args, "--wait")
		}
		if code := h.run(args...); code != c.code || !strings.Contains(h.stderr.String(), c.want) ||
			len(h.fake.Writes) != 1 {
			t.Errorf("--wait=%v: exit %d (want %d), writes %d, stderr %q", c.wait, code, c.code, len(h.fake.Writes), h.stderr)
		}
	}
}

func TestUpgradeWorkersReadsTheOperationFromLocation(t *testing.T) {
	h := elevated(t)
	h.fake.OmitOperationID = true
	h.mustRun("k8s", "upgrade-workers", "k-1", "--version", "v1.33.1", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "op/upgrade-k-1" {
		t.Errorf("stdout = %q", got)
	}
}

func TestUpgradeWorkersWaits(t *testing.T) {
	h := elevated(t)
	h.fake.OpStates["op/upgrade-k-1"] = []string{"running", "complete"}
	h.mustRun("k8s", "upgrade-workers", "k-1", "--version", "v1.33.1", "--wait", "-o", "json")
	if !strings.Contains(h.stdout.String(), `"complete"`) {
		t.Errorf("stdout = %s", h.stdout)
	}
}
