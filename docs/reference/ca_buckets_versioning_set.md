## ca buckets versioning set

Change a bucket's versioning state

### Synopsis

Turns versioning on. It cannot be turned off again, and ca cannot suspend it. Do not suspend it from an S3 client either: a known storage fault can corrupt a suspended bucket. Needs protected:s3. Needs read:s3 as well, because ca looks the bucket up in your bucket list.

```
ca buckets versioning set BUCKET [flags]
```

### Examples

```
  ca buckets versioning set logs-7f3a --mode enabled
```

### Options

```
  -d, --data string   the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help          help for set
      --mode string   The new versioning state. (one of: off, enabled) (required)
  -y, --yes           do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca buckets versioning](ca_buckets_versioning.md)	 - Read and change a bucket's versioning

