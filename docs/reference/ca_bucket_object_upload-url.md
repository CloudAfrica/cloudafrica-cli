## ca bucket object upload-url

Mint a short-lived URL that uploads one new object

### Synopsis

Returns a URL to PUT the bytes of one new object to, valid until `expiresAt`. The PUT must send every header in `headers` exactly as given, because they are signed into the URL, and must send exactly `contentLength` bytes, at most 25 MiB. A key that already exists is refused: when the URL is minted on a bucket with versioning on, and by the PUT itself otherwise, because the URL signs `If-None-Match: *`. On a bucket with versioning on, a write to the key after the mint, this URL's included, becomes a new version and keeps the earlier one as noncurrent. For larger files or an overwrite, use an S3 client with a bucket key. An Idempotency-Key on this call replays an empty 200, not the URL: mint again without one. Needs read:s3 as well as this route's own scope: the bucket is named by id, and the gateway resolves it through the account's gated bucket list, which is itself a read:s3 route.

```
ca bucket object upload-url BUCKET [flags]
```

### Examples

```
  ca bucket object upload-url logs-7f3a --key reports/q3.pdf --content-type application/pdf \
    --content-length 2048 -o json
```

### Options

```
      --content-length int    the exact size of the upload in bytes, at most 25 MiB (required)
      --content-type string   the Content-Type the upload will send (required)
  -d, --data string           the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                  help for upload-url
      --key string            the new object's key, such as reports/q3.pdf; a key that already exists is refused (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket object](ca_bucket_object.md)	 - List a bucket's objects and get short-lived upload and download URLs

