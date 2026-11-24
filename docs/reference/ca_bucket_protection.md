## ca bucket protection

A bucket's object lock and default encryption

### Synopsis

Read-only: object lock is chosen when the bucket is ordered.

```
ca bucket protection BUCKET [flags]
```

### Examples

```
  ca bucket protection logs-7f3a
```

### Options

```
  -h, --help   help for protection
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket](ca_bucket.md)	 - Manage object storage buckets

