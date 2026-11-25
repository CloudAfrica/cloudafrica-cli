## ca vms firewall-rules delete

Remove a firewall rule from a virtual machine

### Synopsis

Asynchronous: returns 202 with the operation to poll at Location.

```
ca vms firewall-rules delete RULE_ID [flags]
```

### Options

```
  -h, --help               help for delete
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

* [ca vms firewall-rules](ca_vms_firewall-rules.md)	 - Manage a VM's firewall rules

