## ca vm vip remove

Take a VM out of a VIP's members

### Synopsis

Requires protected:network, and manage access to both the VM and the VIP's VPC. Withdraws the VM's declaration; the permits it no longer needs come off shortly after the operation completes, and traffic to the VIP stops reaching the VM. Refuses as bind does. Asynchronous: returns 202 with the operation to poll at Location.

```
ca vm vip remove VIP [flags]
```

### Options

```
  -h, --help               help for remove
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

* [ca vm vip](ca_vm_vip.md)	 - Manage a VM's virtual IP membership

