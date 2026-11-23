## ca bucket-key create

Create an object storage access key

### Synopsis

Creates one key, optionally attached to a bucket in the same call. Must carry an Idempotency-Key header — a blind retry without one mints a second live credential. The secret is in this response and in the list nowhere: a caller who discards it retrieves it through the reveal action. Returns 201 with the key and its secret. A retry carrying the same Idempotency-Key replays the 201 with the key and without the secret — a credential is never held for a replay — so a caller that lost the response reads the secret with the reveal action, which needs protected:s3. Refuses with 422 limit-reached (the BIGSTORAGE_KEY key) at the account's key cap, 400 for a name another key already holds, and 404 for a bucketResourceId that resolves to no bucket here. Requires protected:s3.

```
ca bucket-key create [flags]
```

### Examples

```
  ca bucket-key create --name ci --access-type read --bucket-resource-id "$(ca bucket get logs-7f3a -q)"
  # -q prints the secret key, not the id
  SECRET=$(ca bucket-key create --name ci2 --access-type full -q)
```

### Options

```
      --access-type string          read (objects readable) or full (objects readable and writable). (one of: read, full) (required)
      --bucket-resource-id string   the id of the bucket to attach the key to (ca bucket get NAME -q prints it), not its name; omit for an account-level key
  -d, --data string                 the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                        help for create
      --name string                 Key name: 1-60 characters of letters, digits, _ and -. (required)
  -y, --yes                         do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket-key](ca_bucket-key.md)	 - Manage object storage access keys

