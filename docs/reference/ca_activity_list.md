## ca activity list

List activity

### Synopsis

Lists the account's activity, newest first. --types narrows it to kinds of resource. --subject-type with --subject-id shows one resource's newest activity instead, at most 50 rows and in one page. The two do not mix. You see the rows your scopes could read directly; teammates show by display name, and an account service token as "Service token". Needs no scope of its own.

```
ca activity list [flags]
```

### Examples

```
  ca activity list --subject-type vm --subject-id "$(ca vms get web-01 -q)"
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

