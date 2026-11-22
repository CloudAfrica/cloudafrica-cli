package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

// pairedToken sets CLOUDAFRICA_TOKEN and CLOUDAFRICA_ELEVATION to a person's
// API token and the secret it is paired with.
func pairedToken(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	h.env["CLOUDAFRICA_ELEVATION"] = testSecret
	h.fake.PairToken("cak_protected", testSecret)
	return h
}

func TestK8sTokenPrintsTheExecCredential(t *testing.T) {
	h := pairedToken(t)
	h.mustRun("k8s", "token", "k-1", "-o", "table")
	var got struct {
		APIVersion string `json:"apiVersion"`
		Kind       string `json:"kind"`
		Status     struct {
			Token               string `json:"token"`
			ExpirationTimestamp string `json:"expirationTimestamp"`
		} `json:"status"`
	}
	if err := json.Unmarshal(h.stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout %q: %v", h.stdout, err)
	}
	if got.APIVersion != execAPIVersion || got.Kind != "ExecCredential" ||
		got.Status.Token != fakeapi.ClusterToken("k-1") || got.Status.ExpirationTimestamp != "2026-11-22T10:15:00Z" {
		t.Errorf("credential = %+v", got)
	}
	if strings.Contains(h.stderr.String(), fakeapi.ClusterToken("k-1")) {
		t.Errorf("stderr carries the token: %s", h.stderr)
	}
}

// A stored sign-in cannot mint, and the API would answer an unexplained 404,
// so the CLI refuses before calling it.
func TestK8sTokenRefusesASignInWithoutCalling(t *testing.T) {
	h := elevated(t)
	if code := h.run("k8s", "token", "k-1"); code != ExitUsage {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if !strings.Contains(h.stderr.String(), "CLOUDAFRICA_TOKEN") || h.stdout.Len() != 0 || len(h.fake.TokenIdemKeys) != 0 {
		t.Errorf("stderr %q stdout %q calls %d", h.stderr, h.stdout, len(h.fake.TokenIdemKeys))
	}
}

func TestK8sTokenUnpairedTokenGetsThePairingHint(t *testing.T) {
	h := newHarness(t)
	h.env["CLOUDAFRICA_TOKEN"] = "cak_protected"
	if code := h.run("k8s", "token", "k-1"); code != ExitStepUp {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if !strings.Contains(h.stderr.String(), "CLOUDAFRICA_TOKEN is not paired") || h.stdout.Len() != 0 {
		t.Errorf("stderr %q stdout %q", h.stderr, h.stdout)
	}
}

func TestK8sTokenNotFoundNamesWhoCanMint(t *testing.T) {
	h := newHarness(t)
	access, _ := h.fake.IssueRefresh(&fakeapi.Grant{Scopes: append([]string{}, defaultScopes...), Elevated: true})
	h.env["CLOUDAFRICA_TOKEN"] = access
	if code := h.run("k8s", "token", "k-1"); code != ExitAPI {
		t.Fatalf("exit %d, stderr %s", code, h.stderr)
	}
	if !strings.Contains(h.stderr.String(), tokenNotFoundHint) || h.stdout.Len() != 0 {
		t.Errorf("stderr %q stdout %q", h.stderr, h.stdout)
	}
}

func TestK8sTokenAnswerWithoutATokenIsAnError(t *testing.T) {
	for name, body := range map[string]string{
		"empty":        "",
		"blank token":  `{"status":{"token":"","expirationTimestamp":"2026-11-22T10:15:00Z"}}`,
		"no expiry":    `{"status":{"token":"t"}}`,
		"blank expiry": `{"status":{"token":"t","expirationTimestamp":""}}`,
	} {
		t.Run(name, func(t *testing.T) {
			h := pairedToken(t)
			h.fake.EmptyClusterToken = body == ""
			h.fake.ClusterTokenBody = body
			if code := h.run("k8s", "token", "k-1"); code == ExitOK {
				t.Fatal("exit 0")
			}
			if h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), "without a cluster token") {
				t.Errorf("stdout %q stderr %q", h.stdout, h.stderr)
			}
		})
	}
}

type kubeconfigDoc struct {
	Clusters []struct {
		Cluster struct {
			Server string `json:"server"`
		} `json:"cluster"`
	} `json:"clusters"`
	Users []struct {
		User struct {
			Exec struct {
				APIVersion string              `json:"apiVersion"`
				Command    string              `json:"command"`
				Args       []string            `json:"args"`
				Env        []map[string]string `json:"env"`
				Mode       string              `json:"interactiveMode"`
				Hint       string              `json:"installHint"`
			} `json:"exec"`
		} `json:"user"`
	} `json:"users"`
}

func parseKubeconfig(t *testing.T, b []byte) kubeconfigDoc {
	t.Helper()
	var d kubeconfigDoc
	if err := yaml.Unmarshal(b, &d); err != nil || len(d.Users) != 1 || len(d.Clusters) != 1 {
		t.Fatalf("kubeconfig %s: %v", b, err)
	}
	return d
}

// The kubeconfig's exec entry, run as kubectl would run it, prints the
// credential: the args it writes are ones `ca k8s token` takes.
func TestK8sKubeconfigRunsCaK8sToken(t *testing.T) {
	h := signedIn(t)
	h.mustRun("k8s", "kubeconfig", "k-1", "-o", "json")
	d := parseKubeconfig(t, h.stdout.Bytes())
	e := d.Users[0].User.Exec
	if e.Command != "ca" || strings.Join(e.Args, " ") != "k8s token k-1" || e.APIVersion != execAPIVersion ||
		e.Mode != "Never" || !strings.Contains(e.Hint, "CLOUDAFRICA_ELEVATION") {
		t.Errorf("exec = %+v", e)
	}
	if len(e.Env) != 1 || e.Env[0]["name"] != "CLOUDAFRICA_ENDPOINT" || e.Env[0]["value"] != h.fake.URL {
		t.Errorf("env = %v", e.Env)
	}
	if d.Clusters[0].Cluster.Server != "https://prod.k8s.example:6443" {
		t.Errorf("server = %q", d.Clusters[0].Cluster.Server)
	}

	k := pairedToken(t)
	k.env[e.Env[0]["name"]] = e.Env[0]["value"]
	k.fake = h.fake
	h.fake.PairToken("cak_protected", testSecret)
	k.mustRun(e.Args...)
	if !strings.Contains(k.stdout.String(), fakeapi.ClusterToken("k-1")) {
		t.Errorf("exec stdout = %s", k.stdout)
	}
}

func TestK8sKubeconfigManagedVPN(t *testing.T) {
	h := signedIn(t)
	h.mustRun("k8s", "kubeconfig", "k-1", "--managed-vpn")
	if d := parseKubeconfig(t, h.stdout.Bytes()); d.Clusters[0].Cluster.Server != "https://10.9.0.4:6443" ||
		d.Users[0].User.Exec.Command != "ca" {
		t.Errorf("kubeconfig %s", h.stdout)
	}
}

func TestRewriteKubeconfigUsesTheRunName(t *testing.T) {
	out, err := rewriteKubeconfig([]byte(fakeapi.Kubeconfig("k-1", "https://x:6443")), "cloudafrica", "k-1", DefaultEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	if e := parseKubeconfig(t, out).Users[0].User.Exec; e.Command != "cloudafrica" || !strings.Contains(e.Hint, "`cloudafrica`") {
		t.Errorf("exec = %+v", e)
	}
}

func TestRewriteKubeconfigRefusesOneWithNoExecUser(t *testing.T) {
	in := "apiVersion: v1\nkind: Config\nusers:\n- name: u\n  user:\n    token: abc\n"
	if _, err := rewriteKubeconfig([]byte(in), "ca", "k-1", DefaultEndpoint); err == nil {
		t.Fatal("no error")
	}
}
