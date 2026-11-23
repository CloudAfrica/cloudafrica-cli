## ca bucket delete

Delete a bucket and everything in it

### Synopsis

Irreversible: the objects go with the bucket, and the name is never reusable afterwards. Needs read:s3 as well as this route's own scope: the bucket is named by id, and the gateway resolves it through the account's gated bucket list, which is itself a read:s3 route. Requires protected:s3. The body must carry confirmBucketName, the bucket's name spelled exactly as the list returns it — the comparison is exact, matching the backend's own. Asynchronous: returns 202 with the operation to poll at Location, and 409 when a delete for this bucket is already in flight. 503 when the control plane is unavailable: the gateway renders it as its generic upstream-unavailable, an instruction to retry shortly.

```
ca bucket delete BUCKET [flags]
```

### Examples

```
  ca bucket delete logs-7f3a --confirm-bucket-name logs-7f3a
```

### Options

```
      --confirm-bucket-name string   the bucket's name, spelled exactly as ca bucket list shows it (required)
  -d, --data string                  the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                         help for delete
      --no-wait                      print the operation and return without waiting for it
      --timeout duration             how long to wait for the operation (default 10m0s)
  -y, --yes                          do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket](ca_bucket.md)	 - Manage object storage buckets

