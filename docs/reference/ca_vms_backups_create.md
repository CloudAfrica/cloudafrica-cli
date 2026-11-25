## ca vms backups create

Back a virtual machine up now

### Synopsis

An extra backup alongside the VM's policy, not a replacement for it. Must carry an Idempotency-Key header. Asynchronous: returns 202 with the operation to poll at Location.

```
ca vms backups create [flags]
```

### Options

```
      --compress string    Compression: zstd, lzo, gzip or none.
  -d, --data string        the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help               help for create
      --mode string        What the guest does while it is backed up: snapshot (no downtime), suspend, or stop.
      --no-wait            print the operation and return without waiting for it
      --notes string       The backup's description: at most 120 characters, one line. Returned as description by the backup list.
      --timeout duration   how long to wait for the operation (default 10m0s)
      --vm string          the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms backups](ca_vms_backups.md)	 - Manage a VM's backups

