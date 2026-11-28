## ca vpcs reserved-ips create

Reserve a public IP

### Synopsis

Reserves a free public address from the pool for the account until you release it. It counts against the account's public-IP limit whether or not it is attached. It is refused when no address is free. If ca fails without an answer, run `ca vpcs reserved-ips list` before trying again: a second run can reserve a second address. Needs write:network.

```
ca vpcs reserved-ips create [flags]
```

### Examples

```
  ca vpcs reserved-ips create --vpc web
```

### Options

```
  -h, --help         help for create
      --vpc string   the vpc id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpcs reserved-ips](ca_vpcs_reserved-ips.md)	 - Manage a VPC's reserved public IP addresses

