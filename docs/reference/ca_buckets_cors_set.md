## ca buckets cors set

Replace a bucket's CORS rules

### Synopsis

Replaces every CORS rule with the rules you send; an empty list removes them all. Send the rules `ca buckets cors get` returned as basedOn: the change is refused when the stored rules differ, or when you leave basedOn out and the bucket has rules. Prints the rules as stored. Needs read:s3 as well, because ca looks the bucket up in your bucket list.

Some fields of the request body have no flag; set them with --data.

```
ca buckets cors set BUCKET [flags]
```

### Examples

```
  # Remove every rule; basedOn is the rules as you read them
  ca buckets cors get logs-7f3a -o json \
    | jq '{rules: [], basedOn: .rules}' \
    | ca buckets cors set logs-7f3a -d @- --yes
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

* [ca buckets cors](ca_buckets_cors.md)	 - Read and replace a bucket's CORS rules

