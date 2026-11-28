## ca buckets create

Order a new object storage bucket

### Synopsis

Orders a bucket. The platform names it, and it shows in `ca buckets list` once it exists. A bucket past the account's bucket limit is refused, with the reason. Needs write:s3.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca buckets create [flags]
```

### Examples

```
  ca buckets create --display-name backups
```

### Options

```
  -d, --data string           the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --display-name string   Customer-chosen display name (max 120).
  -h, --help                  help for create
      --no-wait               print the operation and return without waiting for it
      --object-lock           Enable S3 object lock. Write-once: a bucket ordered without it can never be locked afterwards.
      --timeout duration      how long to wait for the operation (default 10m0s)
      --versioning string     Versioning at creation, off / enabled; absent leaves versioning off. (one of: off, enabled)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca buckets](ca_buckets.md)	 - Manage object storage buckets

