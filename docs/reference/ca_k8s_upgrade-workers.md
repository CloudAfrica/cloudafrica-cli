## ca k8s upgrade-workers

Upgrade a cluster's workers (a protected action: needs `ca auth elevate`)

### Synopsis

Upgrades every worker to a Kubernetes version and its Talos version, one node at a time. Each node drains, so a workload with one replica is down while its node restarts. Without --version it takes the newest of the cluster's worker targets (`ca k8s get`). When there is none it says so and exits 0 if the workers are CURRENT, DEADLINE or UPGRADE_AVAILABLE, else 1; when waiting (the default, unlike --no-wait) only CURRENT exits 0.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca k8s upgrade-workers CLUSTER_ID [flags]
```

### Options

```
  -h, --help                 help for upgrade-workers
      --no-wait              print the operation and return without waiting for it
      --timeout duration     how long to wait for the operation (default 30m0s)
      --version ca k8s get   Kubernetes version to upgrade to, from ca k8s get (default: the newest target)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca k8s](ca_k8s.md)	 - Manage Kubernetes clusters

