## ca vms public-ips attach

Attach a public IP to a virtual machine

### Synopsis

Attaches a public IP to the VM, drawn from the pool or, with --reserved-ip-id, one of the account's unattached reserved IPs; detaching it later keeps the reservation. --private-lease-id picks the private address it maps to (from `ca vms private-ips list`); the default is the primary interface's. It is refused at the VM's or the account's public-IP limit (a reserved IP counts against the VM's only), while another attach on this VM is running, and for a reserved IP that is attached already. The operation fails if the private address already has a public IP, or the guest's firewall would block traffic to it. Needs protected:network.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms public-ips attach [flags]
```

### Examples

```
  ca vms public-ips attach --vm web-01
```

### Options

```
  -h, --help                      help for attach
      --no-wait                   print the operation and return without waiting for it
      --private-lease-id string   The private address to pair the public IP with.
      --reserved-ip-id string     An unattached Reserved IP of the account to attach, instead of one drawn from the pool.
      --timeout duration          how long to wait for the operation (default 10m0s)
      --vm string                 the vm id or name (required)
  -y, --yes                       do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms public-ips](ca_vms_public-ips.md)	 - Manage a VM's public IP addresses

