## ca vm public-ip detach

Detach a public IP from a virtual machine

### Synopsis

Requires protected:network. Asynchronous: returns 202 with the operation to poll at Location. Refuses with a transitional 409 while THAT ADDRESS already has a release in flight — keyed on the address, not the VM.

```
ca vm public-ip detach PUBLIC_IP_ID [flags]
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

* [ca vm public-ip](ca_vm_public-ip.md)	 - Manage a VM's public IP addresses

