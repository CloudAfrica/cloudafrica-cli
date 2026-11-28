## ca members resend-invitation

Resend a pending invitation

### Synopsis

Sends a pending invitation again. It is refused when the invitation was already accepted, or when there is no one to send it to; remove an accepted member with `ca members delete`.

```
ca members resend-invitation MEMBER [flags]
```

### Examples

```
  ca members resend-invitation jo@example.com
```

### Options

```
  -h, --help   help for resend-invitation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca members](ca_members.md)	 - Manage the account's members

