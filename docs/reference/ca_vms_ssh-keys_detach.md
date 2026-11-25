## ca vms ssh-keys detach

Take an SSH key away from a virtual machine

### Synopsis

Does not delete the key; it stops being installed on this VM.

```
ca vms ssh-keys detach SSH_KEY [flags]
```

### Options

```
  -h, --help        help for detach
      --vm string   the vm id or name (required)
  -y, --yes         do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms ssh-keys](ca_vms_ssh-keys.md)	 - Manage the SSH keys on a VM

