## ca vms resize

Resize a virtual machine

### Synopsis

Only the fields you send change. A disk is named by the id the disk list returns, and disks only grow. Billing-affecting.

Some fields of the request body have no flag; set them with --data.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms resize VM [flags]
```

### Examples

```
  ca vms resize web-01 --vcpu 4 --ram-gib 8
```

### Options

```
  -d, --data string        the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help               help for resize
      --no-wait            print the operation and return without waiting for it
      --ram-gib int        the new RAM in GiB
      --timeout duration   how long to wait for the operation (default 10m0s)
      --vcpu int           the new vCPU count
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms](ca_vms.md)	 - Manage virtual machines

