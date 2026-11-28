## ca vms public-ips detach

Detach a public IP from a virtual machine

### Synopsis

Detaches a public IP from the VM. It is refused while a release of that address is already running. Needs protected:network.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms public-ips detach PUBLIC_IP_ID [flags]
```

### Examples

```
  ca vms public-ips detach "$(ca vms public-ips list --vm web-01 -q)" --vm web-01
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

* [ca vms public-ips](ca_vms_public-ips.md)	 - Manage a VM's public IP addresses

