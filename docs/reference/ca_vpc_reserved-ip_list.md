## ca vpc reserved-ip list

List the account's reserved public IPs

### Synopsis

Requires read:network. The account's reservations in address order, attached or idle; the VPC in the path is only the object the request is authorised on. Cursor-paginated: follow nextCursor via `?cursor=`.

```
ca vpc reserved-ip list [flags]
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for list
      --limit int       fetch one page of at most this many rows (the route's cap applies) and print it as returned
      --vpc string      the vpc id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpc reserved-ip](ca_vpc_reserved-ip.md)	 - Manage a VPC's reserved public IP addresses

