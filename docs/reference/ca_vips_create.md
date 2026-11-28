## ca vips create

Reserve a VIP in a VPC

### Synopsis

Reserves a virtual IP: a free private address in the VPC, picked for you. Needs write:network.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vips create [flags]
```

### Examples

```
  ca vips create --vpc web
```

### Options

```
  -h, --help               help for create
      --no-wait            print the operation and return without waiting for it
      --timeout duration   how long to wait for the operation (default 10m0s)
      --vpc string         the vpc id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vips](ca_vips.md)	 - Manage virtual IPs

