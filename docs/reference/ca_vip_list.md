## ca vip list

List the account's virtual IPs

### Synopsis

One entry per VIP, with its private address, its public overlay address if it has one, and the VMs declared to serve it. In the order /v1/ip-addresses lists their first row. Cursor-paginated: follow nextCursor via `?cursor=`.

```
ca vip list [flags]
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

* [ca vip](ca_vip.md)	 - List virtual IPs

