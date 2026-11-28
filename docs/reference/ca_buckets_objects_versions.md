## ca buckets objects versions

List the versions of one object

### Synopsis

Lists every version of one key, delete markers included: the current version first, then newest first. A bucket that was never versioned lists one entry with no version id. A version written while the list runs can show twice, the second time as noncurrent; one that becomes current because newer versions were deleted meanwhile can be missed. With --limit, a page can be short or empty while more follow: keep going until it has no nextCursor.

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

