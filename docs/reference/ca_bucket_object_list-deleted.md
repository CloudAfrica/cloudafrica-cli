## ca bucket object list-deleted

List deleted objects that can still be recovered

### Synopsis

Lists the keys whose current version is a delete marker, under `prefix` or across the bucket: deleted, with earlier versions still held. Each row is that delete marker. Cursor-paginated. Follow nextCursor until it is null; a page can be short or even empty while more follow, so never infer the end from a page's size.

```
ca bucket object list-deleted BUCKET [flags]
```

### Examples

```
  ca bucket object list-deleted logs-7f3a --prefix reports/
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for list-deleted
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

* [ca bucket object](ca_bucket_object.md)	 - List a bucket's objects and get short-lived upload and download URLs

