## ca ssh-keys list

List the token's own SSH keys

### Synopsis

Keys are person-owned. A CUSTOMER_PAT sees its own person's keys; an ACCOUNT_SERVICE token has no person and is refused. Cursor-paginated: follow nextCursor via `?cursor=`.

```
ca ssh-keys list [flags]
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

* [ca ssh-keys](ca_ssh-keys.md)	 - Manage SSH keys

