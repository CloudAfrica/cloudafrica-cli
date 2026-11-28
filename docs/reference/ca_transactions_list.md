## ca transactions list

List transactions for the account

### Synopsis

Lists invoices and credit notes, newest first. Credit notes are negative and appear once posted to the accounting system. When credit notes cannot be read, the list fails rather than leave them out.

```
ca transactions list [flags]
```

### Examples

```
  ca transactions list
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

* [ca transactions](ca_transactions.md)	 - List invoices and credit notes

