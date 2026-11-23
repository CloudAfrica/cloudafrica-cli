## ca k8s

Manage Kubernetes clusters

```
ca k8s [flags]
```

### Options

```
  -h, --help   help for k8s
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca](ca.md)	 - The CloudAfrica command-line interface
* [ca k8s get](ca_k8s_get.md)	 - Show a Kubernetes cluster, its upgrade status and its nodes
* [ca k8s kubeconfig](ca_k8s_kubeconfig.md)	 - Print a kubeconfig for a cluster that gets its token from `ca k8s token`
* [ca k8s list](ca_k8s_list.md)	 - List Kubernetes clusters
* [ca k8s token](ca_k8s_token.md)	 - Print a 15-minute cluster-admin token as a kubectl ExecCredential
* [ca k8s upgrade-workers](ca_k8s_upgrade-workers.md)	 - Upgrade a cluster's workers (a protected action: needs `ca auth elevate`)

