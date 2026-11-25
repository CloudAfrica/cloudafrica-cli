## ca vms rename

Rename a virtual machine

### Synopsis

Requires write:compute. The new name is a DNS name.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms rename VM [flags]
```

### Examples

```
  ca vms rename web-01 --name db-01
```

### Options

```
  -d, --data string        the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help               help for rename
      --name string        The new DNS-style name. (required)
      --no-wait            print the operation and return without waiting for it
      --timeout duration   how long to wait for the operation (default 10m0s)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms](ca_vms.md)	 - Manage virtual machines

