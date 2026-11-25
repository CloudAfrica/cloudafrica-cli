## ca vms backup-policy delete

Remove a virtual machine's backup policy

### Synopsis

Stops future scheduled backups. Nothing is deleted: every existing backup stays where it is. 404 when the VM had no policy.

```
ca vms backup-policy delete [flags]
```

### Options

```
  -h, --help        help for delete
      --vm string   the vm id or name (required)
  -y, --yes         do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms backup-policy](ca_vms_backup-policy.md)	 - Manage a VM's backup policy

