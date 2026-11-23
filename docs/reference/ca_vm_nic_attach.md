## ca vm nic attach

Attach a network interface to a virtual machine

### Synopsis

Adds an interface in one of the account's VPCs; the platform picks its address. Requires protected:network. Must carry an Idempotency-Key header. Asynchronous: returns 202 with the operation to poll at Location. 202 does not mean the interface is up inside the guest: it is hot-plugged, and the guest's OS must configure it. Refuses with 404 for an unknown VM or VPC.

```
ca vm nic attach [flags]
```

### Options

```
  -d, --data string              the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                     help for attach
      --no-wait                  print the operation and return without waiting for it
      --timeout duration         how long to wait for the operation (default 10m0s)
      --vm string                the vm id or name (required)
      --vpc-resource-id string   The VPC's identifier, as /v1/vpcs uses it. (required)
  -y, --yes                      do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vm nic](ca_vm_nic.md)	 - Manage a VM's network interfaces

