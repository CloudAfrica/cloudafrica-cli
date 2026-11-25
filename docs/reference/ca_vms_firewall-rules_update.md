## ca vms firewall-rules update

Update a firewall rule on a virtual machine

### Synopsis

Asynchronous: returns 202 with the operation to poll at Location.

```
ca vms firewall-rules update RULE_ID [flags]
```

### Options

```
  -d, --data string          the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --description string   Description / note, if set.
      --from-port int        First port in the range, if any.
  -h, --help                 help for update
      --no-wait              print the operation and return without waiting for it
      --protocol string      tcp, udp or icmp. (required)
      --source string        An IPv4 address, a CIDR network, or "any". (required)
      --timeout duration     how long to wait for the operation (default 10m0s)
      --to-port int          Last port in the range, if any.
      --vm string            the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms firewall-rules](ca_vms_firewall-rules.md)	 - Manage a VM's firewall rules

