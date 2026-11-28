## ca buckets objects download-url

Mint a short-lived URL that downloads one object

### Synopsis

Prints a URL that downloads the object, or the version --version-id names, with no other credentials, for about a minute. Treat it as a secret while it works. A missing key, an unknown version, or a delete marker is refused. Needs protected:s3, and read:s3 as well, because ca looks the bucket up in your bucket list.

```
ca buckets objects download-url BUCKET [flags]
```

### Examples

```
  # -q prints the URL, which expires in about a minute
  URL=$(ca buckets objects download-url logs-7f3a --key reports/q3.pdf --yes -q) && curl -fsS -o q3.pdf "$URL"
```

### Options

```
  -d, --data string         the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                help for download-url
      --key string          The object's key. (required)
      --version-id string   an earlier version to download (ca buckets objects versions lists them); omit for the current one
  -y, --yes                 do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca buckets objects](ca_buckets_objects.md)	 - List a bucket's objects and get short-lived upload and download URLs

