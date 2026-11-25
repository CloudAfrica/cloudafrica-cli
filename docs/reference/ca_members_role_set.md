## ca members role set

Change a member's role

### Synopsis

`operator`, `admin`, `billing` or `viewer`; only an owner may change a role to or from `billing`. Valid for a pending invitee too, whose role is set before they ever sign in. `owner` is refused, and an owner's role cannot be changed here either: ownership is transferred, not granted. Requires protected:team — the backend requires a recent second factor for a session and the matching scope for a token.

```
ca members role set MEMBER [flags]
```

### Examples

```
  ca members role set jo@example.com --role admin
```

### Options

```
  -d, --data string   the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help          help for set
      --role string   The new role: operator, admin, billing or viewer. Only an owner may change a role to or from billing. owner is refused, and an existing owner's role cannot be changed here — ownership is transferred, not granted. (one of: operator, admin, billing, viewer) (required)
  -y, --yes           do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca members role](ca_members_role.md)	 - Change a member's role

