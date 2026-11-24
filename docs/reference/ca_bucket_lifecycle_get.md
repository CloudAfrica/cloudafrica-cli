## ca bucket lifecycle get

A bucket's lifecycle rules

### Synopsis

Read live from the storage cluster. `unsupported` means the bucket also holds rules this API cannot represent.

```
ca bucket lifecycle get BUCKET [flags]
```

### Examples

```
  ca bucket lifecycle get logs-7f3a
```

### Options

```
  -h, --help   help for get
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket lifecycle](ca_bucket_lifecycle.md)	 - Read and replace a bucket's lifecycle rules

