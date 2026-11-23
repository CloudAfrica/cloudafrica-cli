## ca vm nic detach

Detach a network interface from a virtual machine

### Synopsis

Requires protected:network. Asynchronous: returns 202 with the operation to poll at Location.

```
ca vm nic detach NIC_ID [flags]
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

* [ca vm nic](ca_vm_nic.md)	 - Manage a VM's network interfaces

