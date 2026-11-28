## ca vms backup-policy get

Get a virtual machine's backup policy

### Synopsis

Shows the VM's backup policy. A VM with no policy is refused, the same as an unknown VM.

```
ca vms backup-policy get [flags]
```

### Options

```
  -h, --help        help for get
      --vm string   the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms backup-policy](ca_vms_backup-policy.md)	 - Manage a VM's backup policy

