## ca members create

Invite someone to the account's team

### Synopsis

Invites someone by email. The invitation stays pending until they accept it, so they have no access yet when this returns. Inviting an accepted member changes nothing. Needs protected:team.

```
ca members create [flags]
```

### Examples

```
  ca members create --email jo@example.com --role operator
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

* [ca members](ca_members.md)	 - Manage the account's members

