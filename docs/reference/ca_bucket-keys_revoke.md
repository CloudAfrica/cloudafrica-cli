## ca bucket-keys revoke

Revoke a key's access to a bucket

### Synopsis

Rewrites the bucket's policy without the key; the key itself stays in the account. Needs read:s3 as well as this route's own scope: the bucket is named by id, and the gateway resolves it through the account's gated bucket list, which is itself a read:s3 route. Requires protected:s3. Returns 204.

```
ca bucket-keys revoke BUCKET_KEY [flags]
```

### Examples

```
  ca bucket-keys revoke ci --bucket logs-7f3a
```

### Options

```
      --bucket string   the bucket name or id (required)
  -h, --help            help for revoke
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

