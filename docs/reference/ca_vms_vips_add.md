## ca vms vips add

Make a VM a member of a VIP

### Synopsis

Requires protected:network, and manage access to both the VM and the VIP's VPC. Declares the VM as a server that answers for the VIP's address and permits the VIP's traffic to it; the platform does not elect a holder, so the guest must configure the address itself. Declaring a VM that is already a member changes nothing. Refuses with 409 for a VIP that a Kubernetes cluster's LoadBalancer Service owns, a VM the platform manages, and a VIP with no VPC on record. Asynchronous: returns 202 with the operation to poll at Location.

```
ca vms vips add VIP [flags]
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

