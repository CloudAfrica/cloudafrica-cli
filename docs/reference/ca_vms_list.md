## ca vms list

List virtual machines in the account

### Synopsis

Lists your VMs, each with its full details. --slug narrows it to the VM with that name.

```
ca vms list [flags]
```

### Examples

```
  ca vms list
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

* [ca vms](ca_vms.md)	 - Manage virtual machines

