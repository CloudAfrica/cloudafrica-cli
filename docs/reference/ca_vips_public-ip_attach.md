## ca vips public-ip attach

Give a VIP a public IP

### Synopsis

Gives the virtual IP a free public address from the pool. A VIP that already has one is refused, and so is an account at its public-IP limit or a pool with no address free. If ca fails without an answer, run `ca vips list`: a VIP that now shows a public address was attached. Needs protected:network.

```
ca vips public-ip attach [flags]
```

### Examples

```
  ca vips public-ip attach --vpc web --vip lb1
```

### Options

```
  -h, --help         help for attach
      --vip string   the vip id or name (required)
      --vpc string   the vpc id or name (required)
  -y, --yes          do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vips public-ip](ca_vips_public-ip.md)	 - Attach and detach a virtual IP's public address

