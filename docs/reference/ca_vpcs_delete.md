## ca vpcs delete

Delete a VPC

### Synopsis

Retracts the VPC and returns its range to the pool. Irreversible. Requires protected:network. Synchronous: returns 204. Refuses with 409 while a VM still holds a lease on its private network. The backend's typed blocker list is not forwarded, and its body carries no detail for the gateway to read, so the 409 names neither the VMs nor how many: the detail is this API's generic conflict sentence.

```
ca vpcs delete VPC [flags]
```

### Options

```
  -h, --help   help for delete
  -y, --yes    do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpcs](ca_vpcs.md)	 - Manage VPCs

