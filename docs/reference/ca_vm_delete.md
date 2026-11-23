## ca vm delete

Delete a virtual machine

### Synopsis

Destroys the VM and its disks. Irreversible. Requires protected:compute. The VM must be stopped: a running VM is refused with 409. Asynchronous: returns 202 with the operation to poll.

```
ca vm delete VM [flags]
```

### Options

```
  -h, --help               help for delete
      --no-wait            print the operation and return without waiting for it
      --timeout duration   how long to wait for the operation (default 10m0s)
  -y, --yes                do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vm](ca_vm.md)	 - Manage virtual machines

