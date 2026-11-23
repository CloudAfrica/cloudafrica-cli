## ca vm firewall-rule create

Add a firewall rule to a virtual machine

### Synopsis

Must carry an Idempotency-Key header. Asynchronous: returns 202 with the operation to poll at Location. Refuses with 422 limit-reached (the FIREWALL_RULES_PER_VM key) at the VM's rule cap.

```
ca vm firewall-rule create [flags]
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
      --vm string            the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vm firewall-rule](ca_vm_firewall-rule.md)	 - Manage a VM's firewall rules

