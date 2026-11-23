## ca k8s kubeconfig

Print a kubeconfig for a cluster that gets its token from `ca k8s token`

### Synopsis

Prints the cluster's kubeconfig as YAML, whatever --output says, with its user running `ca k8s token` in place of the downloaded shell and curl command, so it also works on Windows. Save it and point KUBECONFIG at it. kubectl then needs CLOUDAFRICA_TOKEN and CLOUDAFRICA_ELEVATION (see `ca k8s token --help`); the kubeconfig itself holds no credential. It pins CLOUDAFRICA_ENDPOINT to the API it came from.

```
ca k8s kubeconfig CLUSTER_ID [flags]
```

### Options

```
  -h, --help          help for kubeconfig
      --managed-vpn   the kubeconfig that reaches the cluster over the account's Managed VPN
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca k8s](ca_k8s.md)	 - Manage Kubernetes clusters

