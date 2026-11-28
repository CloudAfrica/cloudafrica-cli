## ca vms backups restore-to-new-vm

Restore a backup onto a new virtual machine

### Synopsis

Creates a new, billable VM from the backup and leaves the source VM as it is. Choose --datacenter-slug from `ca vms backups restore-sites list`. Without --vpc-resource-id the VM is public-only and needs --public-ip. --terms-accepted is required. Needs protected:backup and write:compute, and protected:network with --public-ip. STARTED_AT is the backup's start time as `ca vms backups list` shows it; one that matches no backup, or more than one, is refused.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms backups restore-to-new-vm STARTED_AT [flags]
```

### Examples

```
  ca vms backups restore-to-new-vm 2026-11-01T02:00:00Z --vm web-01 --name web-01-copy \
    --datacenter-slug <dc-slug> --vpc-resource-id "$(ca vpcs get main -q)" --terms-accepted
```

### Options

```
  -d, --data string              the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --datacenter-slug string   the datacenter for the new VM, from ca vms backups restore-sites list (required)
  -h, --help                     help for restore-to-new-vm
      --name string              DNS-style name of the new VM, unique in the account. (required)
      --no-wait                  print the operation and return without waiting for it
      --public-ip                Attach a public IP to the new VM.
      --terms-accepted           Acceptance of the terms of use. Must be true. (required)
      --timeout duration         how long to wait for the operation (default 10m0s)
      --vm string                the vm id or name (required)
      --vpc-resource-id string   the VPC the new VM joins; leave it out for a public-only VM, which needs --public-ip
  -y, --yes                      do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms backups](ca_vms_backups.md)	 - Manage a VM's backups

