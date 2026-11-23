## ca ssh-key update

Rename an SSH key

### Synopsis

Rename only — the key body cannot be rotated; delete and re-create to replace it. The backend authorizes on the stored key's owner and reads only the new name from the body, so the forwarded userId is inert here.

```
ca ssh-key update SSH_KEY [flags]
```

### Options

```
  -d, --data string   the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help          help for update
      --name string   The new display name. (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca ssh-key](ca_ssh-key.md)	 - Manage SSH keys

