## ca vms create

Create a VM

```
ca vms create [flags]
```

### Options

```
      --accept-terms                     accept the terms of use (required)
      --boot-disk-gib int                boot disk size in GiB (required)
      --datacenter ca datacenters list   datacenter slug, from ca datacenters list (required)
      --extra-disk-gib int64Slice        extra data disk in GiB (repeatable, max 8) (default [])
  -h, --help                             help for create
      --image string                     OS image id (required)
      --name string                      DNS-style name, unique in the account (required)
      --no-wait                          print the operation and return without waiting for it
      --placement-group string           placement group id or name
      --public-ip                        attach a public IP
      --ram-gib int                      RAM in GiB, even (required)
      --reserved-ip string               build onto this reserved public IP id
      --ssh-key ca ssh-keys list         SSH key id or name to install, from ca ssh-keys list (repeatable)
      --timeout duration                 how long to wait for the operation (default 10m0s)
      --vcpu int32                       vCPU count, even (required)
      --vpc ca vpcs list                 VPC id or name the VM joins, from ca vpcs list (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms](ca_vms.md)	 - Manage virtual machines

