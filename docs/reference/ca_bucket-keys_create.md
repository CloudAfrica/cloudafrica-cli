## ca bucket-keys create

Create an object storage access key

### Synopsis

Creates an access key, attached to a bucket with --bucket-resource-id or account-wide without it. The secret is printed once, here; `ca bucket-keys reveal-secret` shows it again. A name another key holds is refused, and so is a key past the account's key limit. If ca fails without an answer, run `ca bucket-keys list` before trying again: a second run can create a second live key. Needs protected:s3.

```
ca bucket-keys create [flags]
```

### Examples

```
  # -q prints the secret key, not the id
  SECRET=$(ca bucket-keys create --name ci --access-type read --bucket-resource-id "$(ca buckets get logs-7f3a -q)" --yes -q)
```

### Options

```
      --access-type string          read (objects readable) or full (objects readable and writable). (one of: read, full) (required)
      --bucket-resource-id string   the id of the bucket to attach the key to (ca buckets get NAME -q prints it), not its name; omit for an account-level key
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

* [ca bucket-keys](ca_bucket-keys.md)	 - Manage object storage access keys

