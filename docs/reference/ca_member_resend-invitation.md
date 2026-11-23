## ca member resend-invitation

Resend a pending invitation

### Synopsis

Sends the invitation again to someone who has not accepted yet. Pending invitations only: 409 means the invitation was already accepted, or the invitee has no identity to send to. An accepted member is removed with `DELETE /v1/members/{userId}` (protected:team). `userId` is the id `GET /v1/members` returns.

```
ca member resend-invitation MEMBER [flags]
```

### Examples

```
  ca member resend-invitation jo@example.com
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

* [ca member](ca_member.md)	 - Manage the account's members

