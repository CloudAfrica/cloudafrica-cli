## ca account backup-retention

How long a deleted virtual machine's newest backup is kept

### Synopsis

A live platform value, not a constant: read it here rather than hardcoding it. 0 means the newest backup is removed on the next sweep after the VM is deleted.

```
ca account backup-retention [flags]
```

### Options

```
  -h, --help   help for backup-retention
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca account](ca_account.md)	 - Show the account and its limits

