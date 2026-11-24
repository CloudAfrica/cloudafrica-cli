## ca bucket cors set

Replace a bucket's CORS rules

### Synopsis

Replaces every rule with `rules`; an empty list removes them all. Send the `rules` GET /cors returned as `basedOn`: the save is a 409 when the stored rules differ, including when `basedOn` is omitted and the bucket has rules. No step-up. Returns the rules as stored. Needs read:s3 as well as this route's own scope: the bucket is named by id, and the gateway resolves it through the account's gated bucket list, which is itself a read:s3 route.

Some fields of the request body have no flag; set them with --data.

```
ca bucket cors set BUCKET [flags]
```

### Examples

```
  # Remove every rule; basedOn is the rules as you read them
  ca bucket cors get logs-7f3a -o json \
    | jq '{rules: [], basedOn: .rules}' \
    | ca bucket cors set logs-7f3a -d @- --yes
```

### Options

```
  -d, --data string   the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help          help for set
  -y, --yes           do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket cors](ca_bucket_cors.md)	 - Read and replace a bucket's CORS rules

