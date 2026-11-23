## ca vm backup delete

Forget one backup of a virtual machine

### Synopsis

Requires protected:backup. Permanently deletes the named backup's archive. `startedAt` is the whole-second value the backup list returns; a timestamp that matches no backup is 404, and one that matches more than one is 409.

```
ca vm backup delete STARTED_AT [flags]
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

* [ca vm backup](ca_vm_backup.md)	 - Manage a VM's backups

