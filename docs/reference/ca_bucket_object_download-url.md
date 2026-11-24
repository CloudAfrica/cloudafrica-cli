## ca bucket object download-url

Mint a short-lived URL that downloads one object

### Synopsis

Returns a URL that downloads the object, or the earlier version `versionId` names, with no further credentials until `expiresAt` — about a minute. Treat it as a secret while it lives. 404 for a missing key, an unknown version id, or a delete marker's. An Idempotency-Key on this call replays an empty 200, not the URL: mint again without one. Needs read:s3 as well as this route's own scope: the bucket is named by id, and the gateway resolves it through the account's gated bucket list, which is itself a read:s3 route.

```
ca bucket object download-url BUCKET [flags]
```

### Examples

```
  ca bucket object download-url logs-7f3a --key reports/q3.pdf
  # -q prints the URL, which expires in about a minute
  curl -o q3.pdf "$(ca bucket object download-url logs-7f3a --key reports/q3.pdf --yes -q)"
```

### Options

```
  -d, --data string         the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                help for download-url
      --key string          The object's key. (required)
      --version-id string   an earlier version to download (ca bucket object versions lists them); omit for the current one
  -y, --yes                 do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket object](ca_bucket_object.md)	 - List a bucket's objects and get short-lived upload and download URLs

