## ca vms ssh-keys attach

Make an SSH key usable by a virtual machine

### Synopsis

The key must belong to the token's own person — a key is never usable by an account at large. Use `GET /v1/ssh-keys` for the ids.

```
ca vms ssh-keys attach SSH_KEY [flags]
```

### Options

```
  -h, --help        help for attach
      --vm string   the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms ssh-keys](ca_vms_ssh-keys.md)	 - Manage the SSH keys on a VM

