## ca bucket-keys grant

Grant a key access to a bucket

### Synopsis

Gives the key its permission (read or full) on the bucket, and rewrites the bucket's policy to name it. Needs read:s3 as well, because ca looks the bucket up in your bucket list. Needs protected:s3.

```
ca bucket-keys grant BUCKET_KEY [flags]
```

### Examples

```
  ca bucket-keys grant ci --bucket logs-7f3a
```

### Options

```
      --bucket string   the bucket name or id (required)
  -h, --help            help for grant
  -y, --yes             do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket-keys](ca_bucket-keys.md)	 - Manage object storage access keys

