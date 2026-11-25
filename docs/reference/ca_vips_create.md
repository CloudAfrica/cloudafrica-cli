## ca vips create

Reserve a VIP in a VPC

### Synopsis

Requires write:network. Takes a free private address from the VPC's network; the address is not a parameter. Asynchronous: returns 202 with the operation to poll at Location; a succeeded operation's createdResource names the new VIP, whose id the other /v1/vpcs/{id}/vips routes take.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vips create [flags]
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

