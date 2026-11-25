## ca vms nics detach

Detach a network interface from a virtual machine

### Synopsis

Requires protected:network.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms nics detach NIC_ID [flags]
```

### Options

```
  -h, --help               help for detach
      --no-wait            print the operation and return without waiting for it
      --timeout duration   how long to wait for the operation (default 10m0s)
      --vm string          the vm id or name (required)
  -y, --yes                do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms nics](ca_vms_nics.md)	 - Manage a VM's network interfaces

