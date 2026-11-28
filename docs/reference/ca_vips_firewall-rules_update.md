## ca vips firewall-rules update

Set the description of a VIP firewall rule

### Synopsis

Requires write:network.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vips firewall-rules update RULE_ID [flags]
```

### Options

```
  -d, --data string          the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --description string   at most 255 characters; an empty value clears it
  -h, --help                 help for update
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

* [ca vips firewall-rules](ca_vips_firewall-rules.md)	 - Manage a virtual IP's firewall rules

