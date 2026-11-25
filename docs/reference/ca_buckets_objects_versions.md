## ca buckets objects versions

List the versions of one object

### Synopsis

Lists every version of exactly one key, delete markers included: the current version first, then newest first. A bucket that was never versioned lists one entry with a null versionId. Cursor-paginated; a version written between two pages can list the version it replaced again, now as noncurrent, and when the newest versions are permanently deleted between pages, the version that becomes current can be skipped. Follow nextCursor until it is null; a page can be short or even empty while more follow, so never infer the end from a page's size.

```
ca buckets objects versions BUCKET [flags]
```

### Examples

```
  ca buckets objects versions logs-7f3a --key reports/q3.pdf
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for versions
      --key string      The object's key. (required)
      --limit int       fetch one page of at most this many rows (the route's cap applies) and print it as returned
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca buckets objects](ca_buckets_objects.md)	 - List a bucket's objects and get short-lived upload and download URLs

