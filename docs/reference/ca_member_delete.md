## ca member delete

Remove a member from the account's team

### Synopsis

Also revokes that person's personal API tokens for this account, so a credential they already hold stops working here — removing the membership alone would leave a token that keeps confirming the account. A repeat is 404: they are no longer a member. `userId` is the id `GET /v1/members` returns. Requires protected:team — the backend requires a recent second factor for a session and the matching scope for a token.

```
ca member delete MEMBER [flags]
```

### Examples

```
  ca member delete jo@example.com
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

* [ca member](ca_member.md)	 - Manage the account's members

