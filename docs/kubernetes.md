# Kubernetes

> **Not yet available in production.** These commands are in `ca` but hidden from the command list and the
> reference until managed Kubernetes is offered in production.

```sh
ca k8s list               # UPGRADE: the workers' upgrade state
ca k8s get <id>           # versions, upgrade state and deadlines, worker targets, nodes
ca k8s upgrade-workers <id> [--version v1.33.1] [--no-wait] [--timeout 30m]
ca k8s kubeconfig <id> [--managed-vpn] > prod.kubeconfig
ca k8s token <id>         # the 15-minute credential kubectl reads; the kubeconfig runs it
```

The group also answers to `kubernetes`. Its commands take flags only, not `--data`. `upgrade-workers` is a
protected action and does not ask for confirmation. Each worker drains and
restarts in turn, so a workload with one replica is down while its node restarts. Without `--version`
it takes the newest worker target. If there is none, it says so on stderr. With `--no-wait` it then
exits 0 when the workers are CURRENT, DEADLINE or UPGRADE_AVAILABLE (worker targets never go past the
control plane's version), so a scheduled `--no-wait` run stays quiet until a target is offered. Any other
state with no target exits 1, and so does waiting (the default) unless the workers are CURRENT.
Running it again after an upgrade finishes starts the next one, if there is a newer target.

`ca k8s kubeconfig` prints the cluster's kubeconfig with its user set to run `ca k8s token`, in place of
the downloaded `sh -c curl` command, so it also works on native Windows. It holds no credential. kubectl
runs `ca k8s token` whenever it needs a token: that mints a 15-minute cluster-admin token and needs
`CLOUDAFRICA_TOKEN` (your own API token with `protected:compute`)
and `CLOUDAFRICA_ELEVATION` paired with it, as for a script (see [Scripts and CI](auth.md#scripts-and-ci)). A `ca auth login` sign-in
cannot mint a cluster token. The kubeconfig pins `CLOUDAFRICA_ENDPOINT` to the API it came from.

```sh
export CLOUDAFRICA_TOKEN=cak_...
CLOUDAFRICA_ELEVATION=$(ca auth elevate) || exit
export CLOUDAFRICA_ELEVATION
ca k8s kubeconfig <id> > prod.kubeconfig
KUBECONFIG=prod.kubeconfig kubectl get nodes
```

Every flag is in `ca k8s <command> --help`.
