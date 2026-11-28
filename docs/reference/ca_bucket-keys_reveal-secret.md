## ca bucket-keys reveal-secret

Show a key's secret

### Synopsis

Shows the key's secret. A key created before secrets were stored has none, and the answer says so with available=false. Needs protected:s3.

```
ca bucket-keys reveal-secret BUCKET_KEY [flags]
```

### Examples

```
  # -q prints the secret key, not the id
  SECRET=$(ca bucket-keys reveal-secret ci --yes -q)
```

### Options

```
  -h, --help   help for reveal-secret
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

