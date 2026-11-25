## ca buckets lifecycle set

Replace a bucket's lifecycle rules

### Synopsis

Replaces every rule with `rules`; an empty list removes them all. Send the `rules` GET /lifecycle returned as `basedOn`: the save is a 409 when the stored rules differ, including when `basedOn` is omitted and the bucket has rules. `replaceUnsupported` skips that check and replaces whatever is stored, rules this API cannot represent included. Step-up applies when any rule is enabled or `replaceUnsupported` is set: a personal access token then also needs a paired client. Returns the rules as stored. Needs read:s3 as well as this route's own scope: the bucket is named by id, and the gateway resolves it through the account's gated bucket list, which is itself a read:s3 route.

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
      --replace-unsupported   True to replace whatever is stored, rules this API cannot represent included, without the basedOn check; always needs step-up.
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

