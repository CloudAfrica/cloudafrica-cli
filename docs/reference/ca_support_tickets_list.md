## ca support tickets list

List the account's support tickets

### Synopsis

Lists the account's tickets, most recently updated first. --status filters by status (open, waiting-on-you, resolved or closed) and --search by text. It shows tickets raised through the portal or the API for this account and, with a personal token, that person's own requests too. Internal notes are never shown, and HTML is sanitised.

```
ca support tickets list [flags]
```

### Examples

```
  ca support tickets list --status open,waiting-on-you
```

### Options

```
      --cursor string    with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help             help for list
      --limit int        fetch one page of at most this many rows (the route's cap applies) and print it as returned
      --search string    only tickets that match this text
      --status strings   (repeat the flag or separate values with commas)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca support tickets](ca_support_tickets.md)	 - Manage support tickets

