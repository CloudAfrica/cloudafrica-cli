## ca ssh-keys delete

Delete an SSH key

### Synopsis

Refused with 409 while the key is still linked to a virtual machine — unlink it from them first, then delete it.

```
ca ssh-keys delete SSH_KEY [flags]
```

### Options

```
  -h, --help   help for delete
  -y, --yes    do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca ssh-keys](ca_ssh-keys.md)	 - Manage SSH keys

