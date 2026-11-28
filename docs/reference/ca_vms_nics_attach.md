## ca vms nics attach

Attach a network interface to a virtual machine

### Synopsis

Adds a network interface in one of the account's VPCs; the platform picks its address. The interface is hot-plugged, so the guest's OS must still configure it. Needs protected:network.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms nics attach [flags]
```

### Examples

```
  ca vms nics attach --vm web-01 --vpc-resource-id "$(ca vpcs get db -q)"
```

### Options

```
  -d, --data string              the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                     help for attach
      --no-wait                  print the operation and return without waiting for it
      --timeout duration         how long to wait for the operation (default 10m0s)
      --vm string                the vm id or name (required)
      --vpc-resource-id string   the VPC's id (ca vpcs get NAME -q prints it) (required)
  -y, --yes                      do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms nics](ca_vms_nics.md)	 - Manage a VM's network interfaces

