## ca ssh-keys create

Register an SSH public key

### Synopsis

Registers a public key as yours. An account service token has no person to own the key, and is refused.

```
ca ssh-keys create [flags]
```

### Examples

```
  ca ssh-keys create --name laptop --public-key "$(cat ~/.ssh/id_ed25519.pub)"
```

### Options

```
  -d, --data string         the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                help for create
      --name string         Display name for the key. (required)
      --public-key string   The public key body, e.g. ssh-ed25519 AAAA…. (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca ssh-keys](ca_ssh-keys.md)	 - Manage SSH keys

