## ca bucket-key delete

Delete an object storage access key

### Synopsis

Detaches the key from every bucket, then removes it. The account's master key cannot be deleted — that would orphan every bucket it owns. Requires protected:s3. Returns 204.

```
ca bucket-key delete BUCKET_KEY [flags]
```

### Options

```
  -h, --help   help for delete
  -y, --yes    do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket-key](ca_bucket-key.md)	 - Manage object storage access keys

