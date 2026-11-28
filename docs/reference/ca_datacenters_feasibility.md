## ca datacenters feasibility

Check whether a datacenter can take a VM shape

### Synopsis

Shows how many of --count VMs of this shape the datacenter could place, optionally in a placement group. Nothing is reserved. --count defaults to 1 and the answer never exceeds it; the platform may lower a large count, and the output's `requested` shows the count it used. --extra-disks-gib is the total of all extra disks, not one disk.

```
ca datacenters feasibility [flags]
```

### Examples

```
  ca datacenters feasibility --datacenter jnb --vcpu 4 --ram-gib 8 --boot-disk-gib 40
```

### Options

```
      --boot-disk-gib int           Boot disk size in GiB. (required)
      --count int                   How many VMs of this shape; defaults to 1, at most 1000.
      --datacenter string           A datacenter slug. (required)
      --extra-disks-gib int         The TOTAL size in GiB of all extra disks a create would request; defaults to 0.
  -h, --help                        help for feasibility
      --placement-group-id string   A placement group the VMs would join.
      --ram-gib int                 the VM's RAM in GiB (required)
      --vcpu int                    vCPU count. (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca datacenters](ca_datacenters.md)	 - List datacenters and check placement feasibility

