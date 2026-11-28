## ca vms backups restore-sites list

List the datacenters a backup can be restored to as a new VM

### Synopsis

Lists the datacenters that can take this backup as a new VM, each with whether it has room. An empty list means it cannot be restored to a new VM. --backup is the backup's start time as `ca vms backups list` shows it; one that matches no backup, or more than one, is refused.

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

