## ca vms backups restore-sites list

List the datacenters a backup can be restored to as a new VM

### Synopsis

The datacenters that can take this backup as a new virtual machine, each with whether it has room. An empty list means the backup cannot be restored to a new VM. `startedAt` is the whole-second value the backup list returns; a timestamp that matches no backup is 404, and one that matches more than one is 409.

```
ca vms backups restore-sites list [flags]
```

### Options

```
      --backup string   the backup's start time: RFC 3339 or epoch seconds (required)
  -h, --help            help for list
      --vm string       the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms backups restore-sites](ca_vms_backups_restore-sites.md)	 - Manage restore-sites

