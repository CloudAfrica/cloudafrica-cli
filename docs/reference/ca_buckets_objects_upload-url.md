## ca buckets objects upload-url

Mint a short-lived URL that uploads one new object

### Synopsis

Prints a URL and headers to upload one new object of at most 25 MiB. The PUT must send every header exactly as given, because they are signed into the URL, and exactly --content-length bytes. A key that already exists is refused. On a bucket with versioning on, a later write to the key becomes a new version and keeps this one. For a larger file or an overwrite, use an S3 client with a bucket key. Needs read:s3 as well, because ca looks the bucket up in your bucket list.

```
ca buckets objects upload-url BUCKET [flags]
```

### Examples

```
  ca buckets objects upload-url logs-7f3a --key reports/q3.pdf --content-type application/pdf \
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

* [ca buckets objects](ca_buckets_objects.md)	 - List a bucket's objects and get short-lived upload and download URLs

