## ca backup-policies list

List the account's backup policies

### Synopsis

Lists the backup policy of each live VM, with the VM's id. A deleted VM's policy is not listed; `ca vms backup-policy get` still shows it.

```
ca backup-policies list [flags]
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for list
      --limit int       fetch one page of at most this many rows (the route's cap applies) and print it as returned
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca backup-policies](ca_backup-policies.md)	 - List the backup policy of every live VM

