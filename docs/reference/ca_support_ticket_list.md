## ca support ticket list

List the account's support tickets

### Synopsis

Cursor-paginated, most recently updated first. Pass `?cursor=…` from the previous response's nextCursor to fetch the next page. Filter with `status` (repeatable: open, waiting-on-you, resolved, closed) and `q` (free text). The tickets returned are those raised through the portal or the API for this account, plus, for a personal token, the token owner's own requests. Internal notes are never returned, and HTML bodies are sanitised.

```
ca support ticket list [flags]
```

### Examples

```
  ca support ticket list
  ca support ticket list --status open,waiting-on-you
```

### Options

```
      --cursor string    with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help             help for list
      --limit int        fetch one page of at most this many rows (the route's cap applies) and print it as returned
      --search string    
      --status strings   (repeat the flag or separate values with commas)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca support ticket](ca_support_ticket.md)	 - Manage support tickets

