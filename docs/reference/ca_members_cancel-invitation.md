## ca members cancel-invitation

Cancel a pending invitation

### Synopsis

Withdraws an invitation that has not been accepted. An accepted invitation is refused; remove that member with `ca members delete`.

```
ca members cancel-invitation MEMBER [flags]
```

### Examples

```
  ca members cancel-invitation jo@example.com
```

### Options

```
  -h, --help   help for cancel-invitation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca members](ca_members.md)	 - Manage the account's members

