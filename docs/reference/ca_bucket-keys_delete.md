## ca bucket-keys delete

Delete an object storage access key

### Synopsis

Detaches the key from every bucket, then deletes it. The account's master key cannot be deleted, because every bucket it owns would be orphaned. Needs protected:s3.

```
ca bucket-keys delete BUCKET_KEY [flags]
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

* [ca bucket-keys](ca_bucket-keys.md)	 - Manage object storage access keys

