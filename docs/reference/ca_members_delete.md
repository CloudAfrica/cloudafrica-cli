## ca members delete

Remove a member from the account's team

### Synopsis

Removes the member, and revokes their personal API tokens for this account so a token they hold stops working here. Removing someone who is not a member is refused. Needs protected:team.

```
ca members delete MEMBER [flags]
```

### Examples

```
  ca members delete jo@example.com
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

* [ca members](ca_members.md)	 - Manage the account's members

