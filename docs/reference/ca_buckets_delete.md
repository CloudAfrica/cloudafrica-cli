## ca buckets delete

Delete a bucket and everything in it

### Synopsis

Deletes the bucket and every object in it. It cannot be undone, and the name can never be used again. --confirm-bucket-name must spell the bucket's name exactly as `ca buckets list` shows it. A second delete while one is running is refused. Needs read:s3 as well, because ca looks the bucket up in your bucket list. Needs protected:s3.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca buckets delete BUCKET [flags]
```

### Examples

```
  ca buckets delete logs-7f3a --confirm-bucket-name logs-7f3a
```

### Options

```
      --confirm-bucket-name string   the bucket's name, spelled exactly as ca buckets list shows it (required)
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

* [ca buckets](ca_buckets.md)	 - Manage object storage buckets

