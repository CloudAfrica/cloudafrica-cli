## ca bucket-keys reveal-secret

Reveal the secret the backend holds for a key

### Synopsis

Requires protected:s3. 200 with the secret, or 200 with available=false and no secret when the backend holds none — a key created before secret storage was backfilled. 404 only for a key that is not in this account. Answered no-store: a live credential must not sit in a response an intermediary may cache. A retry carrying the same Idempotency-Key replays the 200 without the secret, so call it again without that header to read one.

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

