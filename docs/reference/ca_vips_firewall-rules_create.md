## ca vips firewall-rules create

Add a firewall rule to a VIP

### Synopsis

Requires write:network.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vips firewall-rules create [flags]
```

### Examples

```
  ca vips firewall-rules create --vpc web --vip lb1 --protocol tcp --from-port 443 --to-port 443 --source any
```

### Options

```
  -d, --data string          the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --description string   Description / note, if set.
      --from-port int        First port in the range, if any.
  -h, --help                 help for create
      --no-wait              print the operation and return without waiting for it
      --protocol string      tcp, udp or icmp. (required)
      --source string        An IPv4 address, a CIDR network, or "any". (required)
      --timeout duration     how long to wait for the operation (default 10m0s)
      --to-port int          Last port in the range, if any.
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

