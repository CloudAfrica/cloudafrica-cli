package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/cmdspec"
	"github.com/CloudAfrica/cloudafrica-cli/internal/output"
)

// upgradeWaitTimeout is upgrade-workers' --timeout default: each worker
// drains and restarts in turn.
const upgradeWaitTimeout = 30 * time.Minute

var k8sHeaders = []string{"ID", "NAME", "STATE", "VERSION", "WORKERS", "UPGRADE"}

func k8sRow(c cloudafrica.PublicKubernetesClusterDto) []string {
	upgrade := "-"
	if c.Upgrade != nil && c.Upgrade.Workers != nil && c.Upgrade.Workers.State != nil {
		upgrade = string(*c.Upgrade.Workers.State)
	}
	return []string{output.S(c.Id), output.S(c.Name), output.S(c.State), output.S(c.KubernetesVersion),
		output.N(c.Workers), upgrade}
}

// upgradeStatus is "<version> <state> (deadline <date>)", leaving out what
// is absent.
func upgradeStatus(version *string, st *cloudafrica.PublicKubernetesUpgradeStatusDto) string {
	parts := []string{}
	if v := output.S(version); v != "" {
		parts = append(parts, v)
	}
	if st != nil && st.State != nil {
		parts = append(parts, string(*st.State))
	}
	if st != nil && st.Deadline != nil {
		parts = append(parts, "(deadline "+st.Deadline.Format("2006-01-02")+")")
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " ")
}

func (a *App) k8sCmd() *cobra.Command {
	cmd := group("k8s", "Manage Kubernetes clusters")
	cmd.Aliases = []string{"kubernetes"}
	// Hidden from help and the reference until managed Kubernetes is offered in production; it still runs.
	cmd.Hidden = true
	cmd.AddCommand(a.k8sList(), a.k8sGet(), a.k8sUpgradeWorkers(), a.k8sToken(), a.k8sKubeconfig())
	return cmd
}

func (a *App) k8sList() *cobra.Command {
	return listCmd(a, "List Kubernetes clusters", k8sHeaders, k8sRow,
		func(c cloudafrica.PublicKubernetesClusterDto) string { return output.S(c.Id) },
		func(ctx context.Context, c *cloudafrica.ClientWithResponses) ([]cloudafrica.PublicKubernetesClusterDto, error) {
			return listAll(func(cursor *string) (*[]cloudafrica.PublicKubernetesClusterDto, *string, error) {
				limit := pageLimit
				r, err := c.ListKubernetesClustersWithResponse(ctx, &cloudafrica.ListKubernetesClustersParams{Cursor: cursor, Limit: &limit})
				if err != nil {
					return nil, nil, err
				}
				if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
					return nil, nil, err
				}
				if r.JSON200 == nil {
					return nil, nil, errors.New("empty Kubernetes cluster list response")
				}
				return r.JSON200.Data, r.JSON200.NextCursor, nil
			})
		})
}

func getCluster(ctx context.Context, s *session, id string) (*cloudafrica.PublicKubernetesClusterDetailDto, error) {
	r, err := s.client.API.GetKubernetesClusterWithResponse(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
		return nil, err
	}
	if r.JSON200 == nil || r.JSON200.Cluster == nil {
		return nil, errors.New("empty Kubernetes cluster response")
	}
	return r.JSON200, nil
}

func clusterRows(d *cloudafrica.PublicKubernetesClusterDetailDto) [][]string {
	c := d.Cluster
	rows := [][]string{
		{"ID", output.S(c.Id)},
		{"Name", output.S(c.Name)},
		{"State", output.S(c.State)},
		{"Workers", output.N(c.Workers)},
		{"API server", output.S(c.ApiServerEndpoint)},
	}
	u := c.Upgrade
	if u == nil {
		rows = append(rows, []string{"Control plane", upgradeStatus(c.KubernetesVersion, nil)},
			[]string{"Upgrade", "status unavailable"})
	} else {
		workerVersion := output.S(u.WorkerKubernetesVersion)
		if t := output.S(u.WorkerTalosVersion); t != "" {
			workerVersion += " (Talos " + t + ")"
		}
		rows = append(rows, []string{"Control plane", upgradeStatus(c.KubernetesVersion, u.ControlPlane)},
			[]string{"Workers version", upgradeStatus(&workerVersion, u.Workers)})
		targets := []string{}
		if u.WorkerTargets != nil {
			for _, t := range *u.WorkerTargets {
				s := output.S(t.Kubernetes)
				if talos := output.S(t.Talos); talos != "" {
					s += " (Talos " + talos + ")"
				}
				targets = append(targets, s)
			}
		}
		if len(targets) == 0 {
			targets = append(targets, "-")
		}
		rows = append(rows, []string{"Worker targets", strings.Join(targets, ", ")})
		if w := u.ControlPlaneWindow; w != nil && w.Start != nil && w.End != nil {
			rows = append(rows, []string{"Control-plane window", fmt.Sprintf("%s to %s, %s",
				w.Start.Format("2006-01-02"), w.End.Format("2006-01-02"), output.S(w.Kubernetes))})
		}
	}
	if d.Nodes != nil {
		for _, n := range *d.Nodes {
			ready := "not ready"
			if n.Ready != nil && *n.Ready {
				ready = "ready"
			}
			rows = append(rows, []string{"Node " + output.S(n.Name),
				strings.Join([]string{output.S(n.Status), ready, output.S(n.PrivateIp)}, "  ")})
		}
	}
	return rows
}

func (a *App) k8sGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get CLUSTER_ID",
		Short: "Show a Kubernetes cluster, its upgrade status and its nodes",
		Args:  exactArgs(1, "CLUSTER_ID"),
		RunE: func(cmd *cobra.Command, args []string) error {
			pr, err := a.printer()
			if err != nil {
				return err
			}
			return a.withSession(cmd.Context(), func(s *session) error {
				d, err := getCluster(cmd.Context(), s, args[0])
				if err != nil {
					return err
				}
				return pr.Print(output.Result{Value: d, Rows: clusterRows(d), IDs: []string{output.S(d.Cluster.Id)}})
			})
		},
	}
}

// errUpgradeUnavailable is a cluster read that carried no upgrade status,
// which the API omits while it cannot read it.
const workersCurrent = "workers are current"

var errUpgradeUnavailable = errors.New("upgrade status is unavailable; try again")

// newestWorkerTarget is the version to upgrade the workers to: the last
// (newest) entry of workerTargets. "" with a nil error means there is
// nothing to do yet, and note says why; a state that leaves the workers with
// no target and no stated way forward is an error.
func newestWorkerTarget(c *cloudafrica.PublicKubernetesClusterDto) (target, note string, err error) {
	u := c.Upgrade
	if u == nil {
		return "", "", errUpgradeUnavailable
	}
	if u.WorkerTargets != nil {
		for i := len(*u.WorkerTargets) - 1; i >= 0; i-- {
			if v := output.S((*u.WorkerTargets)[i].Kubernetes); v != "" {
				return v, "", nil
			}
		}
	}
	status := "workers: " + upgradeStatus(u.WorkerKubernetesVersion, u.Workers) + "; there is no worker target"
	state := cloudafrica.PublicKubernetesUpgradeStatusDtoState("")
	if u.Workers != nil && u.Workers.State != nil {
		state = *u.Workers.State
	}
	switch state {
	case cloudafrica.CURRENT:
		return "", workersCurrent, nil
	case cloudafrica.DEADLINE, cloudafrica.UPGRADEAVAILABLE:
		return "", status + " yet", nil
	default:
		return "", "", fmt.Errorf("%s; `ca k8s get %s` shows the cluster", status, output.S(c.Id))
	}
}

func (a *App) k8sUpgradeWorkers() *cobra.Command {
	var (
		version string
		wf      waitFlags
	)
	cmd := &cobra.Command{
		Use:   "upgrade-workers CLUSTER_ID",
		Short: "Upgrade a cluster's workers (a protected action)",
		Long: "Upgrades every worker to a Kubernetes version and its Talos version, one node at a time. Each node " +
			"drains, so a workload with one replica is down while its node restarts. Without --version it takes the " +
			"newest of the cluster's worker targets (`ca k8s get`). When there is none it says so and exits 0 if the " +
			"workers are CURRENT, DEADLINE or UPGRADE_AVAILABLE, else 1; when waiting (the default, unlike --no-wait) " +
			"only CURRENT exits 0.\n\n" + cmdspec.WaitLine,
		Args: exactArgs(1, "CLUSTER_ID"),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			return a.withSession(cmd.Context(), func(s *session) error {
				target := version
				if target != "" && !strings.HasPrefix(target, "v") {
					target = "v" + target
				}
				if target == "" {
					d, err := getCluster(cmd.Context(), s, id)
					if err != nil {
						return err
					}
					t, note, err := newestWorkerTarget(d.Cluster)
					if err != nil {
						return err
					}
					if t == "" {
						if !wf.noWait && note != workersCurrent {
							return errors.New(note + "; there is nothing to wait for")
						}
						fmt.Fprintln(a.Stderr, note)
						return nil
					}
					target = t
				}
				r, err := s.client.API.UpgradeKubernetesClusterWorkersWithResponse(cmd.Context(), id,
					cloudafrica.PublicKubernetesClusterUpgradeWorkersRequest{TargetKubernetes: target})
				if err != nil {
					return err
				}
				if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
					return err
				}
				opID, ok := "", false
				if r.JSON202 != nil && r.JSON202.OperationId != nil && *r.JSON202.OperationId != "" {
					opID, ok = *r.JSON202.OperationId, true
				} else {
					opID, ok = operationIDFromLocation(r.HTTPResponse.Header.Get("Location"))
				}
				if !ok && r.HTTPResponse.StatusCode != http.StatusAccepted && r.HTTPResponse.StatusCode != http.StatusOK {
					return fmt.Errorf("HTTP %d with no operation id", r.HTTPResponse.StatusCode)
				}
				fmt.Fprintf(a.Stderr, "Upgrading the workers of %s to %s\n", id, target)
				if !ok {
					// Accepted with no operation to follow.
					if !wf.noWait {
						return fmt.Errorf("there is no operation to wait for; `ca k8s get %s` shows the upgrade", id)
					}
					fmt.Fprintf(a.Stderr, "There is no operation to follow; `ca k8s get %s` shows the upgrade.\n", id)
					return nil
				}
				return a.accepted(cmd.Context(), s, opID, wf)
			})
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "Kubernetes version to upgrade to, from `ca k8s get` (default: the newest target)")
	wf.bind(cmd, true)
	t := cmd.Flags().Lookup("timeout")
	_ = t.Value.Set(upgradeWaitTimeout.String())
	t.DefValue = t.Value.String()
	return cmd
}
