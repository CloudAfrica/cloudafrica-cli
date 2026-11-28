## ca vms backups set-protection

Protect or release one backup

### Synopsis

Protects a backup from rotation, or releases it. A protected backup is not deleted when retention tightens. STARTED_AT is the backup's start time as `ca vms backups list` shows it; one that matches no backup, or more than one, is refused. Needs protected:backup.

```
ca vms backups set-protection STARTED_AT [flags]
```

### Examples

```
  ca vms backups set-protection 2026-11-01T02:00:00Z --vm web-01 --protected
```

### Options

```
  -d, --data string   the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help          help for set-protection
      --protected     True to protect the backup, false to release it.
      --vm string     the vm id or name (required)
  -y, --yes           do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms backups](ca_vms_backups.md)	 - Manage a VM's backups

