## ca vpc vip firewall-rule list

List firewall rules that apply to a VIP

### Synopsis

The rules addressed to the VIP's public IP that name no VM.

```
ca vpc vip firewall-rule list [flags]
```

### Options

```
  -h, --help         help for list
      --vip string   the vip id or name (required)
      --vpc string   the vpc id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpc vip firewall-rule](ca_vpc_vip_firewall-rule.md)	 - Manage a virtual IP's firewall rules

