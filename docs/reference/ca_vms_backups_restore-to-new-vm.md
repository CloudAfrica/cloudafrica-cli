## ca vms backups restore-to-new-vm

Restore a backup onto a new virtual machine

### Synopsis

Creates a new, billable VM from the named backup and leaves the source VM untouched. Requires protected:backup, and the token must also hold write:compute, plus protected:network when `publicIp` is set; a token without them is refused with 403. `termsAccepted` must be true. Choose `datacenterSlug` from GET …/restore-sites. A VM with no `vpcResourceId` is public-only and needs `publicIp`. `startedAt` is the whole-second value the backup list returns; a timestamp that matches no backup is 404, and one that matches more than one is 409.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms backups restore-to-new-vm STARTED_AT [flags]
```

### Options

```
  -d, --data string              the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --datacenter-slug string   The datacenter slug for the new VM; see GET …/restore-sites. (required)
  -h, --help                     help for restore-to-new-vm
      --name string              DNS-style name of the new VM, unique in the account. (required)
      --no-wait                  print the operation and return without waiting for it
      --public-ip                Attach a public IP to the new VM.
      --terms-accepted           Acceptance of the terms of use. Must be true. (required)
      --timeout duration         how long to wait for the operation (default 10m0s)
      --vm string                the vm id or name (required)
      --vpc-resource-id string   The VPC the new VM joins. Omit for a public-only VM, which needs publicIp.
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

