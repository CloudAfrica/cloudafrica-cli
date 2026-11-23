## ca vm backup list

List a virtual machine's backups

### Synopsis

Newest first. Cursor-paginated: follow nextCursor via `?cursor=`.

```
ca vm backup list [flags]
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for list
      --limit int       fetch one page of at most this many rows (the route's cap applies) and print it as returned
      --vm string       the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vm backup](ca_vm_backup.md)	 - Manage a VM's backups

