## ca vms reboot

Restart a virtual machine

### Synopsis

Shuts the VM down and starts it again.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms reboot VM [flags]
```

### Examples

```
  ca vms reboot web-01
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

