## ca backup-policy list

List the account's backup policies

### Synopsis

Each live virtual machine's backup policy, tagged with the VM's id. Only live VMs are listed: a deleted VM's policy is not, though /v1/vms/{id}/backup-policy still answers it. Cursor-paginated: follow nextCursor via `?cursor=`.

```
ca backup-policy list [flags]
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

* [ca backup-policy](ca_backup-policy.md)	 - List the backup policy of every VM

