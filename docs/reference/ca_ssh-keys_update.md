## ca ssh-keys update

Rename an SSH key

### Synopsis

Renames the key. The key itself cannot be changed: delete it and create a new one.

```
ca ssh-keys update SSH_KEY [flags]
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

* [ca ssh-keys](ca_ssh-keys.md)	 - Manage SSH keys

