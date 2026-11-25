## ca vms add-disk

Attach a data disk to a virtual machine

### Synopsis

Billing-affecting: the new volume raises the VM's disk charge. Requires write:compute. Must carry an Idempotency-Key header: a retry under the same key replays the first call's stored response rather than starting a second, 409 while that first call is still in flight, and 422 if the key is reused for a different request. That matters most here: a blind retry without one adds a SECOND billable volume. Asynchronous: returns 202 with the operation to poll at Location.

```
ca vms add-disk VM [flags]
```

### Options

```
  -d, --data string        the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help               help for add-disk
      --no-wait            print the operation and return without waiting for it
      --size-gib int       Size in GiB, at least 1. A size above the disk ceiling is refused with a 400.
      --timeout duration   how long to wait for the operation (default 10m0s)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms](ca_vms.md)	 - Manage virtual machines

