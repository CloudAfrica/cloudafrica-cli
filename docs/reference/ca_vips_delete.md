## ca vips delete

Release a VIP

### Synopsis

Requires protected:network. Returns the private address to the VPC's network; it may be given to someone else. Refuses with 409 while the VIP has a public IP: detach it first.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vips delete VIP [flags]
```

### Options

```
  -h, --help               help for delete
      --no-wait            print the operation and return without waiting for it
      --timeout duration   how long to wait for the operation (default 10m0s)
      --vpc string         the vpc id or name (required)
  -y, --yes                do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vips](ca_vips.md)	 - Manage virtual IPs

