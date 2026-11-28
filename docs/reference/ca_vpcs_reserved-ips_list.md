## ca vpcs reserved-ips list

List the account's reserved public IPs

### Synopsis

Lists the account's reserved public IPs in address order, attached or idle. They belong to the account, not to the VPC --vpc names, so any of its VPCs lists the same ones. Needs read:network.

```
ca vpcs reserved-ips list [flags]
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

* [ca vpcs reserved-ips](ca_vpcs_reserved-ips.md)	 - Manage a VPC's reserved public IP addresses

