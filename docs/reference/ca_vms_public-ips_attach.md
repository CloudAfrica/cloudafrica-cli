## ca vms public-ips attach

Attach a public IP to a virtual machine

### Synopsis

Requires protected:network. Refuses with 422 limit-reached at the VM's public-IP cap (the PUBLIC_IPS_PER_VM key) or the account's (the PUBLIC_IP key), and with a transitional 409 while THIS VM already has an attach in flight — the guard is keyed on the VM, so a concurrent attach on another of the account's VMs is not refused. privateLeaseId optionally names the VM's private address (a leaseId from GET /v1/vms/{id}/private-ips) the public IP is NATed to; absent keeps the primary interface's address. The operation fails if the named private address already has a public IP, or if the guest's firewall would not pass traffic to it. reservedIpId optionally names one of the account's unattached Reserved IPs (GET /v1/vpcs/{id}/reserved-ips) to attach instead of drawing one from the pool: 404 if the account does not hold it, 409 if it is already attached. It is exempt from the account's PUBLIC_IP cap, which already counts it, but not from the VM's. Detaching it later keeps the reservation.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms public-ips attach [flags]
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

