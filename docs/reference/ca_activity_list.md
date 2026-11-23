## ca activity list

List activity

### Synopsis

Two modes on one route. The account feed takes `types`, `cursor` and `limit` and returns the account's activity newest first; pass the previous response's nextCursor as `cursor` for the next page. The subject history takes `subjectType` and `subjectId` (and `limit`) and returns that one subject's newest activity, at most `limit` rows and never a nextCursor. Mixing the two is 400. `limit` is 1 to 50 and defaults to 50. Rows are limited to what the token's scopes could read directly, and rows show teammates' short display names, or "Service token" for an account service token. Readable by any token of the account; it needs no scope of its own.

```
ca activity list [flags]
```

### Examples

```
  ca activity list --limit 20
  ca activity list --types vm,vpc
  ca activity list --subject-type vm --subject-id "$(ca vm get web-01 -q)"
```

### Options

```
      --cursor string         with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help                  help for list
      --limit int             fetch one page of at most this many rows (the route's cap applies) and print it as returned
      --subject-id string     Subject history: the subject's id.
      --subject-type string   Subject history: the kind of subject, for example vm.
      --types strings         Account feed: subject types to include, repeated or comma-separated. (repeat the flag or separate values with commas)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca activity](ca_activity.md)	 - List account activity

