## ca vms private-ips create

Add a private IP address to a virtual machine's network interface

### Synopsis

Leases the next free address from the interface's VPC; the caller picks the interface, never the address. Requires write:network. Must carry an Idempotency-Key header. Asynchronous: returns 202 with the operation to poll at Location. Refuses with 404 for an unknown VM or interface.

```
ca vms private-ips create [flags]
```

### Options

```
  -d, --data string        the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help               help for create
      --nic-id string      The interface's identifier, from GET /v1/vms/{id}/network-interfaces. (required)
      --no-wait            print the operation and return without waiting for it
      --timeout duration   how long to wait for the operation (default 10m0s)
      --vm string          the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms private-ips](ca_vms_private-ips.md)	 - Manage a VM's private IP addresses

