## ca vm list

List virtual machines in the account

### Synopsis

Cursor-paginated: pass `cursor` from the previous response's nextCursor to fetch the next page. Use this to find a VM when you do not have its id: filter by current display name with `slug`, or read every VM in the account. Each row is the full VM, so no second call is needed for its details. When you already hold the VM's id, call getVm instead of listing.

```
ca vm list [flags]
```

### Examples

```
  ca vm list
  ca vm list -o json
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for list
      --limit int       fetch one page of at most this many rows (the route's cap applies) and print it as returned
      --slug string     only the VM with this slug
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vm](ca_vm.md)	 - Manage virtual machines

