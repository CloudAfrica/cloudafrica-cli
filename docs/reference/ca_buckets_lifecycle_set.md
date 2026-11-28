## ca buckets lifecycle set

Replace a bucket's lifecycle rules

### Synopsis

Replaces every lifecycle rule with the rules you send; an empty list removes them all. Send the rules `ca buckets lifecycle get` returned as basedOn: the change is refused when the stored rules differ, or when you leave basedOn out and the bucket has rules. --replace-unsupported skips that check and replaces whatever is stored. Needs protected:s3, and read:s3 as well, because ca looks the bucket up in your bucket list. Enabling a rule, or --replace-unsupported, also needs a paired sign-in or token. Prints the rules as stored.

Some fields of the request body have no flag; set them with --data.

```
ca buckets lifecycle set BUCKET [flags]
```

### Examples

```
  # Expire current versions after 30 days; basedOn is the rules as you read them
  ca buckets lifecycle get logs-7f3a -o json \
    | jq '{rules: (.rules | map(.expirationDays = 30)), basedOn: .rules}' \
    | ca buckets lifecycle set logs-7f3a -d @- --yes
```

### Options

```
  -d, --data string           the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                  help for set
      --replace-unsupported   replace whatever is stored, without the basedOn check; always needs a paired sign-in or token
  -y, --yes                   do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca buckets lifecycle](ca_buckets_lifecycle.md)	 - Read and replace a bucket's lifecycle rules

