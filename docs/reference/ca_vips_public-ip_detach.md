## ca vips public-ip detach

Take a VIP's public IP off

### Synopsis

Returns the virtual IP's public address to the pool; a VIP with none is left as it is. A VIP that still has firewall rules is refused: delete them with `ca vips firewall-rules delete` first. A VIP the platform manages is refused too. Needs protected:network.

```
ca vips public-ip detach [flags]
```

### Options

```
  -h, --help         help for detach
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

