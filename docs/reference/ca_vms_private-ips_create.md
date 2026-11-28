## ca vms private-ips create

Add a private IP address to a virtual machine's network interface

### Synopsis

Adds the next free private address on one of the VM's interfaces; you pick the interface, not the address. Needs write:network.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms private-ips create [flags]
```

### Examples

```
  ca vms private-ips create --vm web-01 --nic-id "$(ca vms nics list --vm web-01 -q | head -1)"
```

### Options

```
  -d, --data string        the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help               help for create
      --nic-id string      the interface's id, from ca vms nics list (required)
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

