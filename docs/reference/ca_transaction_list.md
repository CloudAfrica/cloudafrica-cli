## ca transaction list

List transactions for the account

### Synopsis

Invoices and credit notes, cursor-paginated, newest first. Credit notes are negative and appear once posted to the accounting system. Pass `?cursor=…` from the previous response's nextCursor to fetch the next page. Answers 502 when credit notes cannot be read, rather than a page without them.

```
ca transaction list [flags]
```

### Examples

```
  ca transaction list
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

* [ca transaction](ca_transaction.md)	 - List invoices and credit notes

