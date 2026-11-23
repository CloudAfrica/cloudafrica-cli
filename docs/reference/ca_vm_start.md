## ca vm start

Start a virtual machine

### Synopsis

Asynchronous: returns 202 with the operation to poll at Location.

```
ca vm start VM [flags]
```

### Examples

```
  ca vm start web-01
```

### Options

```
  -h, --help               help for start
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

* [ca vm](ca_vm.md)	 - Manage virtual machines

