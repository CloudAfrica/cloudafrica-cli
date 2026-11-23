## ca member list

List the account's team members

### Synopsis

Includes pending invitees, as the Users section of the portal's Account page does. Ordered by name, or by email where there is none. Cursor-paginated: follow nextCursor via `?cursor=`.

```
ca member list [flags]
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for list
      --limit int       fetch one page of at most this many rows (the route's cap applies) and print it as returned
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca member](ca_member.md)	 - Manage the account's members

