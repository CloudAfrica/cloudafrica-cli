## ca member create

Invite someone to the account's team

### Synopsis

The invitation is PENDING until it is accepted: a 200 means the membership row exists, not that the person has access yet. Inviting someone who is already an accepted member changes nothing and answers 200 the same way. Requires protected:team — the backend requires a recent second factor for a session and the matching scope for a token.

```
ca member create [flags]
```

### Examples

```
  ca member create --email jo@example.com --role operator
```

### Options

```
  -d, --data string         the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --email string        The colleague's login email address. (required)
      --first-name string   First name; used only when the invitation creates a new login.
  -h, --help                help for create
      --last-name string    Last name; used only when the invitation creates a new login.
      --role string         The role to grant: operator, admin, billing or viewer. Only an owner may grant billing. owner is refused — ownership is transferred, not granted. (one of: operator, admin, billing, viewer) (required)
  -y, --yes                 do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca member](ca_member.md)	 - Manage the account's members

