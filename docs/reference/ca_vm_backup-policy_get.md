## ca vm backup-policy get

Get a virtual machine's backup policy

### Synopsis

404 for either an unknown VM or a VM with no policy — absence is the status, not a policy with empty fields.

```
ca vm backup-policy get [flags]
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

* [ca vm backup-policy](ca_vm_backup-policy.md)	 - Manage a VM's backup policy

