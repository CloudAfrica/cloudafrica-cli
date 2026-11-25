## ca projects resources list

List the resources in a project

### Synopsis

Every live resource the project holds, including kinds that cannot be moved out of it: they still count toward the project's resourceCount and its delete guard. Ordered by name. Cursor-paginated: follow nextCursor via `?cursor=`.

```
ca projects resources list [flags]
```

### Options

```
      --cursor string    with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help             help for list
      --limit int        fetch one page of at most this many rows (the route's cap applies) and print it as returned
      --project string   the project id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca projects resources](ca_projects_resources.md)	 - Manage the resources in a project

