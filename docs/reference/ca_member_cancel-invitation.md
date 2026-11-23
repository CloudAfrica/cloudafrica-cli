## ca member cancel-invitation

Cancel a pending invitation

### Synopsis

Withdraws an invitation that has not been accepted yet. Pending invitations only: 409 means the invitation was already accepted. An accepted member is removed with `DELETE /v1/members/{userId}` (protected:team). `userId` is the id `GET /v1/members` returns.

```
ca member cancel-invitation MEMBER [flags]
```

### Examples

```
  ca member cancel-invitation jo@example.com
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

* [ca member](ca_member.md)	 - Manage the account's members

