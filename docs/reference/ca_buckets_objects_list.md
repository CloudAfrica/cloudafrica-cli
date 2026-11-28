## ca buckets objects list

List the objects in a bucket

### Synopsis

Lists the objects in key order, under --prefix or across the bucket. With --limit, a page can be short or empty while more follow: keep going until it has no nextCursor.

```
ca buckets objects list BUCKET [flags]
```

### Examples

```
  ca buckets objects list logs-7f3a --prefix reports/2026/ --limit 50
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for list
      --limit int       fetch one page of at most this many rows (the route's cap applies) and print it as returned
      --prefix string   List only keys starting with this text.
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca buckets objects](ca_buckets_objects.md)	 - List a bucket's objects and get short-lived upload and download URLs

