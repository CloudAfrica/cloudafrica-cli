## ca bucket object get

Get one object's metadata, tags and retention

### Synopsis

Reads the current version's content headers, user metadata, tags and object-lock retention. 404 for a missing key, and for an object whose current version is a delete marker.

```
ca bucket object get BUCKET [flags]
```

### Examples

```
  ca bucket object get logs-7f3a --key reports/q3.pdf
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

* [ca bucket object](ca_bucket_object.md)	 - List a bucket's objects and get short-lived upload and download URLs

