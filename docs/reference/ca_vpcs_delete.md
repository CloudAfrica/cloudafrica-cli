## ca vpcs delete

Delete a VPC

### Synopsis

Deletes the VPC and returns its address range to the pool. It cannot be undone. A VPC that a VM still uses is refused, without naming the VM: `ca vms list` and `ca vms nics list` find it. Needs protected:network.

```
ca vpcs delete VPC [flags]
```

### Examples

```
  ca vpcs delete web
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

