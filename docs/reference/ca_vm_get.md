## ca vm get

Get a virtual machine by id

### Synopsis

Get one virtual machine by its id, with its current state. Use this when you already hold the id, from listVms or an earlier call. It cannot look a VM up by name: to find an id from a name, call listVms with `slug`. A 404 means no VM of that id in this account.

```
ca vm get VM [flags]
```

### Examples

```
  ca vm get web-01
```

### Options

```
  -h, --help   help for get
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vm](ca_vm.md)	 - Manage virtual machines

