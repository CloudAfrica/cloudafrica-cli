## ca vms backups set-protection

Protect or release one backup

### Synopsis

A protected backup is skipped by rotation, which is what stops a retention tightening from deleting it. `startedAt` is the whole-second value the backup list returns; a timestamp that matches no backup is 404, and one that matches more than one is 409.

```
ca vms backups set-protection STARTED_AT [flags]
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

