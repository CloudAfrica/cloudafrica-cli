## ca vms vips add

Make a VM a member of a VIP

### Synopsis

Makes the VM a server for the virtual IP's address and lets the VIP's traffic reach it. The platform does not choose a holder, so the guest must configure the address itself. Adding a member again changes nothing. A VIP owned by a Kubernetes LoadBalancer Service, a VM the platform manages, or a VIP with no VPC is refused. Needs protected:network and access to both the VM and the VIP's VPC.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms vips add VIP [flags]
```

### Examples

```
  ca vms vips add lb1 --vm web-01
```

### Options

```
  -h, --help               help for add
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

* [ca vms vips](ca_vms_vips.md)	 - Manage a VM's virtual IP membership

