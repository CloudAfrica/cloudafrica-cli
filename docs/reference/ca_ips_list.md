## ca ips list

List every IP address the account holds

### Synopsis

Attached or not, public and private alike. Ordered public-before-private, then by owner name, then by address.

```
ca ips list [flags]
```

### Examples

```
  ca ips list
  ca ips list -o json
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

* [ca ips](ca_ips.md)	 - List IP addresses

