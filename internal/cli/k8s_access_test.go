package cli

import (
	"encoding/json"
	"fmt"
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

// Without CLOUDAFRICA_TOKEN the CLI refuses before calling the API, which
// would answer an unexplained 404, and never says to sign in: a sign-in
// cannot mint.
func TestK8sTokenWithoutAnEnvTokenRefusesWithoutCalling(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T) *harness{
		"signed in":   elevated,
		"no sign-in":  newHarness,
		"blank token": func(t *testing.T) *harness { h := newHarness(t); h.env["CLOUDAFRICA_TOKEN"] = " "; return h },
		"signed in elsewhere": func(t *testing.T) *harness {
			h := elevated(t)
			h.env["CLOUDAFRICA_ENDPOINT"] = "https://elsewhere.example"
			return h
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := setup(t)
			if code := h.run("k8s", "token", "k-1"); code != ExitUsage {
				t.Fatalf("exit %d, stderr %s", code, h.stderr)
			}
			if !strings.Contains(h.stderr.String(), errTokenNeedsPAT.Error()) || strings.Contains(h.stderr.String(), "run `ca auth login`") ||
				h.stdout.Len() != 0 || len(h.fake.TokenIdemKeys) != 0 {
				t.Errorf("stderr %q stdout %q calls %d", h.stderr, h.stdout, len(h.fake.TokenIdemKeys))
			}
		})
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

// The credential is the command's whole output: failing to write it fails.
func TestK8sTokenFailsWhenTheCredentialCannotBeWritten(t *testing.T) {
	h := pairedToken(t)
	h.failStdout = true
	if code := h.run("k8s", "token", "k-1"); code == ExitOK {
		t.Errorf("exit 0, stderr %s", h.stderr)
	}
}

type kubeconfigDoc struct {
	CurrentContext string `json:"current-context"`
	Contexts       []struct {
		Name string `json:"name"`
	} `json:"contexts"`
	Clusters []struct {
		Cluster struct {
			Server string `json:"server"`
			CA     string `json:"certificate-authority-data"`
		} `json:"cluster"`
	} `json:"clusters"`
	Users []struct {
		Name string `json:"name"`
		User struct {
			Token string `json:"token"`
			Exec  struct {
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
	d := parseKubeconfigUsers(t, b)
	if len(d.Users) != 1 {
		t.Fatalf("kubeconfig %s", b)
	}
	return d
}

func parseKubeconfigUsers(t *testing.T, b []byte) kubeconfigDoc {
	t.Helper()
	var d kubeconfigDoc
	if err := yaml.Unmarshal(b, &d); err != nil || len(d.Clusters) != 1 {
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
	if d.Clusters[0].Cluster.Server != "https://prod.k8s.example:6443" || d.Clusters[0].Cluster.CA != "Q0E=" ||
		d.CurrentContext != "prod" || len(d.Contexts) != 1 || d.Contexts[0].Name != "prod" {
		t.Errorf("rest of the kubeconfig = %+v", d)
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

// A failed fetch writes nothing, so `> prod.kubeconfig` never saves a broken one.
func TestK8sKubeconfigFailedFetchWritesNothing(t *testing.T) {
	for name, c := range map[string]struct {
		status int
		body   string
		want   string
	}{
		"not found":    {404, "", "404"},
		"not a 200":    {203, fakeapi.Kubeconfig("k-1", "https://x:6443"), "HTTP 203 with no kubeconfig"},
		"empty":        {200, "", "HTTP 200 with no kubeconfig"},
		"not yaml":     {200, "users: [", "reading the kubeconfig"},
		"no exec user": {200, "users:\n- name: u\n  user:\n    token: abc\n", "no exec user"},
	} {
		for _, vpn := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s vpn=%v", name, vpn), func(t *testing.T) {
				h := signedIn(t)
				h.fake.KubeconfigStatus, h.fake.KubeconfigBody = c.status, c.body
				args := []string{"k8s", "kubeconfig", "k-1"}
				if vpn {
					args = append(args, "--managed-vpn")
				}
				if code := h.run(args...); code == ExitOK {
					t.Fatal("exit 0")
				}
				if h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), c.want) {
					t.Errorf("stdout %q stderr %q", h.stdout, h.stderr)
				}
			})
		}
	}
}

func TestK8sKubeconfigFailsWhenItCannotBeWritten(t *testing.T) {
	h := signedIn(t)
	h.failStdout = true
	if code := h.run("k8s", "kubeconfig", "k-1"); code == ExitOK {
		t.Errorf("exit 0, stderr %s", h.stderr)
	}
}

// Every exec user is rewritten; a user with a static credential is left as is.
func TestRewriteKubeconfigRewritesEveryExecUser(t *testing.T) {
	in := `clusters:
- name: prod
  cluster:
    server: https://x:6443
users:
- name: a
  user:
    exec:
      command: sh
- name: static
  user:
    token: abc
- name: b
  user:
    exec:
      command: sh
`
	out, err := rewriteKubeconfig([]byte(in), "ca", "k-1", DefaultEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	d := parseKubeconfigUsers(t, out)
	if len(d.Users) != 3 {
		t.Fatalf("users %s", out)
	}
	for _, u := range d.Users {
		switch u.Name {
		case "static":
			if u.User.Token != "abc" || u.User.Exec.Command != "" {
				t.Errorf("static user = %+v", u.User)
			}
		default:
			if u.User.Exec.Command != "ca" || strings.Join(u.User.Exec.Args, " ") != "k8s token k-1" {
				t.Errorf("user %s = %+v", u.Name, u.User.Exec)
			}
		}
	}
}

func TestExeName(t *testing.T) {
	for argv0, want := range map[string]string{
		"ca":                "ca",
		"ca.exe":            "ca",
		"/usr/local/bin/ca": "ca",
		"cloudafrica.exe":   "cloudafrica",
		"":                  "",
		"/":                 "",
	} {
		if got := exeName(argv0); got != want {
			t.Errorf("exeName(%q) = %q, want %q", argv0, got, want)
		}
	}
}
