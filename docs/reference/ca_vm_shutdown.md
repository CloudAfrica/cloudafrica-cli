## ca vm shutdown

Shut down a virtual machine gracefully

### Synopsis

Asks the guest OS to shut down. Asynchronous: returns 202 with the operation to poll.

```
ca vm shutdown VM [flags]
```

### Options

```
      --guest-timeout int   Seconds to wait for the guest before giving up. Default 60.
  -h, --help                help for shutdown
      --no-wait             print the operation and return without waiting for it
      --timeout duration    how long to wait for the operation (default 10m0s)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vm](ca_vm.md)	 - Manage virtual machines

