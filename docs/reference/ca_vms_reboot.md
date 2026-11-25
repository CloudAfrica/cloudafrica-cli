## ca vms reboot

Restart a virtual machine

### Synopsis

Shuts the virtual machine down and starts it again. Asynchronous: returns 202 with the operation to poll.

```
ca vms reboot VM [flags]
```

### Options

```
  -h, --help               help for reboot
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

