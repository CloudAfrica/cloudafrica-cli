## ca vips delete

Release a VIP

### Synopsis

Releases the virtual IP's private address back to the VPC, where it may go to something else. A VIP with a public IP is refused: detach it with `ca vips public-ip detach` first. Needs protected:network.

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

