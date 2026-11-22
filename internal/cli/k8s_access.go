package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"
)

// execAPIVersion is the client-go exec credential version the token route
// answers in, and the one the rewritten kubeconfig asks kubectl for.
const execAPIVersion = "client.authentication.k8s.io/v1beta1"

// errTokenNeedsPAT refuses a cluster token for a stored sign-in: the API
// mints one only for a person's own API token, and answers anything else 404.
var errTokenNeedsPAT = errors.New("a cluster token needs CLOUDAFRICA_TOKEN set to your own API token with " +
	"protected:compute, and CLOUDAFRICA_ELEVATION set to the secret paired with it " +
	"(`CLOUDAFRICA_ELEVATION=$(ca auth elevate)`); a `ca auth login` sign-in cannot get one")

const tokenNotFoundHint = "404 also means CLOUDAFRICA_TOKEN is not a person's own API token, or that person is not an " +
	"owner or admin of the cluster's account."

func execInstallHint(exe string) string {
	return fmt.Sprintf("Needs the CloudAfrica CLI as `%s` on PATH (https://github.com/CloudAfrica/cloudafrica-cli). "+
		"Set CLOUDAFRICA_TOKEN to your own API token with protected:compute and CLOUDAFRICA_ELEVATION to the "+
		"secret paired with it (`CLOUDAFRICA_ELEVATION=$(ca auth elevate)`), then run kubectl again.", exe)
}

// exe is the name the CLI was run as, which kubectl runs again by PATH.
func (a *App) exe() string {
	if a.Exe == "" {
		return "ca"
	}
	return a.Exe
}

func (a *App) k8sToken() *cobra.Command {
	return &cobra.Command{
		Use:   "token CLUSTER_ID",
		Short: "Print a 15-minute cluster-admin token as a kubectl ExecCredential",
		Long: "Prints the ExecCredential JSON kubectl's exec plugin reads; `ca k8s kubeconfig` writes a kubeconfig " +
			"that runs this. It always prints JSON, whatever --output says. Needs CLOUDAFRICA_TOKEN (your own API " +
			"token with protected:compute, as an account owner or admin) and CLOUDAFRICA_ELEVATION (the secret " +
			"paired with it); a `ca auth login` sign-in cannot get a cluster token.",
		Args: exactArgs(1, "CLUSTER_ID"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.withSession(cmd.Context(), func(s *session) error {
				if !s.envToken {
					return usageErr(errTokenNeedsPAT)
				}
				cred, err := mintClusterToken(cmd.Context(), s, args[0])
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(a.Stdout, "%s\n", cred)
				return err
			})
		},
	}
}

// mintClusterToken is the ExecCredential JSON for cluster id. An answer
// without a token and expiry (a replayed request answers an empty 200) is
// an error, so kubectl never reads an empty credential.
func mintClusterToken(ctx context.Context, s *session, id string) ([]byte, error) {
	r, err := s.client.API.CreateKubernetesClusterTokenWithResponse(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
		var ae *cloudafrica.APIError
		if errors.As(err, &ae) && ae.Status == http.StatusNotFound {
			return nil, &hintedError{err: err, hint: tokenNotFoundHint}
		}
		return nil, err
	}
	c := r.JSON200
	if c == nil || c.Status == nil || c.Status.Token == nil || *c.Status.Token == "" ||
		c.Status.ExpirationTimestamp == nil || *c.Status.ExpirationTimestamp == "" {
		return nil, errors.New("the API answered without a cluster token; run the command again")
	}
	return json.Marshal(map[string]any{
		"apiVersion": execAPIVersion,
		"kind":       "ExecCredential",
		"status":     map[string]string{"token": *c.Status.Token, "expirationTimestamp": *c.Status.ExpirationTimestamp},
	})
}

func (a *App) k8sKubeconfig() *cobra.Command {
	var managedVPN bool
	cmd := &cobra.Command{
		Use:   "kubeconfig CLUSTER_ID",
		Short: "Print a kubeconfig for a cluster that gets its token from `ca k8s token`",
		Long: "Prints the cluster's kubeconfig as YAML, whatever --output says, with its user running `ca k8s token` " +
			"in place of the downloaded shell and curl command, so it also works on Windows. Save it and point " +
			"KUBECONFIG at it. kubectl then needs CLOUDAFRICA_TOKEN and CLOUDAFRICA_ELEVATION (see `ca k8s token " +
			"--help`); the kubeconfig itself holds no credential. It pins CLOUDAFRICA_ENDPOINT to the API it came from.",
		Args: exactArgs(1, "CLUSTER_ID"),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			return a.withSession(cmd.Context(), func(s *session) error {
				var body []byte
				var hr *http.Response
				if managedVPN {
					r, err := s.client.API.GetKubernetesClusterManagedVpnKubeconfigWithResponse(cmd.Context(), id)
					if err != nil {
						return err
					}
					hr, body = r.HTTPResponse, r.Body
				} else {
					r, err := s.client.API.GetKubernetesClusterKubeconfigWithResponse(cmd.Context(), id)
					if err != nil {
						return err
					}
					hr, body = r.HTTPResponse, r.Body
				}
				if err := cloudafrica.CheckResponse(hr, body); err != nil {
					return err
				}
				if hr.StatusCode != http.StatusOK || len(body) == 0 {
					return fmt.Errorf("HTTP %d with no kubeconfig", hr.StatusCode)
				}
				out, err := rewriteKubeconfig(body, a.exe(), id, s.endpoint)
				if err != nil {
					return err
				}
				_, err = a.Stdout.Write(out)
				return err
			})
		},
	}
	cmd.Flags().BoolVar(&managedVPN, "managed-vpn", false, "the kubeconfig that reaches the cluster over the account's Managed VPN")
	return cmd
}

// rewriteKubeconfig replaces the exec command of every user in kubeconfig
// with `<exe> k8s token <id>`, run against endpoint. A kubeconfig with no
// exec user is refused: there would be nothing to rewrite, and kubectl
// would sign in some other way than this command says.
func rewriteKubeconfig(kubeconfig []byte, exe, id, endpoint string) ([]byte, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(kubeconfig, &doc); err != nil {
		return nil, fmt.Errorf("reading the kubeconfig: %w", err)
	}
	users, _ := doc["users"].([]any)
	rewritten := 0
	for _, u := range users {
		named, _ := u.(map[string]any)
		user, _ := named["user"].(map[string]any)
		if user == nil || user["exec"] == nil {
			continue
		}
		user["exec"] = map[string]any{
			"apiVersion":      execAPIVersion,
			"command":         exe,
			"args":            []string{"k8s", "token", id},
			"env":             []map[string]string{{"name": "CLOUDAFRICA_ENDPOINT", "value": endpoint}},
			"interactiveMode": "Never",
			"installHint":     execInstallHint(exe),
		}
		rewritten++
	}
	if rewritten == 0 {
		return nil, errors.New("the kubeconfig has no exec user to point at `ca k8s token`")
	}
	return yaml.Marshal(doc)
}
