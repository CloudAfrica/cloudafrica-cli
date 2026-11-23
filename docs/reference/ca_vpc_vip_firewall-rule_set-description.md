## ca vpc vip firewall-rule set-description

Set the description of a VIP firewall rule

### Synopsis

Requires write:network. Asynchronous: returns 202 with the operation to poll at Location.

```
ca vpc vip firewall-rule set-description RULE_ID [flags]
```

### Options

```
  -d, --data string          the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --description string   At most 255 characters; null or blank clears it.
  -h, --help                 help for set-description
      --no-wait              print the operation and return without waiting for it
      --timeout duration     how long to wait for the operation (default 10m0s)
      --vip string           the vip id or name (required)
      --vpc string           the vpc id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpc vip firewall-rule](ca_vpc_vip_firewall-rule.md)	 - Manage a virtual IP's firewall rules

