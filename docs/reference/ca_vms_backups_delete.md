## ca vms backups delete

Forget one backup of a virtual machine

### Synopsis

Permanently deletes one backup. STARTED_AT is the backup's start time as `ca vms backups list` shows it; one that matches no backup, or more than one, is refused. Needs protected:backup.

```
ca vms backups delete STARTED_AT [flags]
```

### Examples

```
  ca vms backups delete 2026-11-01T02:00:00Z --vm web-01
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

* [ca vms backups](ca_vms_backups.md)	 - Manage a VM's backups

