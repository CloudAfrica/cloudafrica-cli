## ca ip-address list

List every IP address the account holds

### Synopsis

Attached or not, public and private alike. Ordered public-before-private, then by owner name, then by address. Cursor-paginated: follow nextCursor via `?cursor=`.

```
ca ip-address list [flags]
```

### Examples

```
  ca ip-address list
  ca ip-address list -o json
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

* [ca ip-address](ca_ip-address.md)	 - List IP addresses

