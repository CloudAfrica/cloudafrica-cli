## ca buckets versioning set

Change a bucket's versioning state

### Synopsis

Turns versioning on. It cannot be turned off afterwards, and this API does not suspend it. Do not suspend it from an S3 client: a known storage fault can corrupt a suspended bucket. Requires protected:s3 for every mode. Needs read:s3 as well as this route's own scope: the bucket is named by id, and the gateway resolves it through the account's gated bucket list, which is itself a read:s3 route.

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

