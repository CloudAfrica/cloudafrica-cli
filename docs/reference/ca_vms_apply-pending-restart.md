## ca vms apply-pending-restart

Apply a pending configuration change by restarting the virtual machine

### Synopsis

For a change (such as a resize) that takes effect only after a power cycle: shuts the guest down, then starts it again, as one operation.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca vms apply-pending-restart VM [flags]
```

### Options

```
      --guest-timeout int   Seconds the shutdown phase waits for the guest before giving up. Default 60.
  -h, --help                help for apply-pending-restart
      --no-wait             print the operation and return without waiting for it
      --timeout duration    how long to wait for the operation (default 10m0s)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms](ca_vms.md)	 - Manage virtual machines

