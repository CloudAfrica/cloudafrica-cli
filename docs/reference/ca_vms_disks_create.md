## ca vms disks create

Attach a data disk to a virtual machine

### Synopsis

Attaches a new data disk, which adds to the VM's disk charge. If ca fails without an answer, run `ca vms disks list` before trying again: a second run adds a second billable disk. Needs write:compute.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms disks create [flags]
```

### Examples

```
  ca vms disks create --vm web-01 --size-gib 100
```

### Options

```
  -d, --data string        the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help               help for create
      --no-wait            print the operation and return without waiting for it
      --size-gib int       size in GiB, at least 1, up to the disk limit
      --timeout duration   how long to wait for the operation (default 10m0s)
      --vm string          the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms disks](ca_vms_disks.md)	 - Manage a VM's disks

