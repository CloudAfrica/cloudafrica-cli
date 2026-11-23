## ca k8s token

Print a 15-minute cluster-admin token as a kubectl ExecCredential

### Synopsis

Prints the ExecCredential JSON kubectl's exec plugin reads; `ca k8s kubeconfig` writes a kubeconfig that runs this. It always prints JSON, whatever --output says. Needs CLOUDAFRICA_TOKEN (your own API token with protected:compute) and CLOUDAFRICA_ELEVATION (the secret paired with it); a `ca auth login` sign-in cannot get a cluster token.

```
ca k8s token CLUSTER_ID [flags]
```

### Options

```
  -h, --help   help for token
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca k8s](ca_k8s.md)	 - Manage Kubernetes clusters

