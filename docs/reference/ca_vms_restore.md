## ca vms restore

Restore a virtual machine from one of its backups

### Synopsis

Restores the VM from one of its backups, overwriting everything written since. --started-at is the backup's start time in epoch seconds; `ca vms backups list` shows it in RFC 3339, so convert it first. One that matches no backup, or more than one, is refused. Needs protected:backup.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms restore VM [flags]
```

### Examples

```
  # --started-at is the backup's start time in epoch seconds (2026-11-01T02:00:00Z)
  ca vms restore web-01 --started-at 1793498400
```

### Options

```
  -d, --data string        the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help               help for restore
      --no-wait            print the operation and return without waiting for it
      --started-at int     the backup's start time in epoch seconds (ca vms backups list shows it in RFC 3339) (required)
      --timeout duration   how long to wait for the operation (default 10m0s)
  -y, --yes                do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms](ca_vms.md)	 - Manage virtual machines

