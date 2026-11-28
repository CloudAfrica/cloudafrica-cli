## ca buckets objects get

Get one object's metadata, tags and retention

### Synopsis

Shows the current version's content headers, user metadata, tags and retention. A missing key, or one whose current version is a delete marker, is refused.

```
ca buckets objects get BUCKET [flags]
```

### Examples

```
  ca buckets objects get logs-7f3a --key reports/q3.pdf
```

### Options

```
  -h, --help         help for get
      --key string   The object's key. (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca buckets objects](ca_buckets_objects.md)	 - List a bucket's objects and get short-lived upload and download URLs

