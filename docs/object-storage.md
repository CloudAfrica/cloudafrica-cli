# Object storage

Buckets, their objects, and the access keys that reach them. Every command is in the [reference](reference/ca.md).

Some of these commands, deletes among them, are protected actions: run `ca auth elevate` before them. A
protected command run without it exits 5 and prints the remedy. See [Protected actions](auth.md#protected-actions),
or [Scripts and CI](auth.md#scripts-and-ci) when you sign in with a token.

## Buckets

```sh
ca bucket list|get|create|delete
ca bucket usage logs-7f3a --hours 168
ca bucket versioning get|set logs-7f3a
```

`ca bucket delete` deletes the bucket and everything in it.

## Objects

```sh
ca bucket object list logs-7f3a --prefix reports/2026/
ca bucket object get logs-7f3a --key reports/q3.pdf
ca bucket object versions logs-7f3a --key reports/q3.pdf
ca bucket object list-deleted logs-7f3a
```

`list` and `list-deleted` follow every page; `--limit N` prints one page. `-q` prints keys.

The API moves an object's bytes through a short-lived URL, not through itself.

```sh
# Download: the URL works for about a minute and needs no credentials. Protected: elevate first.
curl -o q3.pdf "$(ca bucket object download-url logs-7f3a --key reports/q3.pdf --yes -q)"

# Upload one new object of at most 25 MiB. The PUT must send every header the API returns, and exactly
# --content-length bytes.
SIZE=$(stat -c %s q3.pdf)
SPEC=$(ca bucket object upload-url logs-7f3a --key reports/q3.pdf --content-type application/pdf \
  --content-length "$SIZE" -o json)
curl -X PUT "$(jq -r .url <<<"$SPEC")" \
  -K <(jq -r '.headers | to_entries[] | "header = \"\(.key): \(.value)\""' <<<"$SPEC") \
  --data-binary @q3.pdf
```

An upload to a key that already exists is refused. Overwriting, deleting or restoring an object, and any
file over 25 MiB, go through an S3 client with a [bucket key](#access-keys): point it at the S3 endpoint
the bucket's page in the portal shows (`s3.bigstorage.io`).

## Lifecycle, CORS and protection

```sh
ca bucket lifecycle get|set logs-7f3a
ca bucket cors get|set logs-7f3a
ca bucket protection logs-7f3a
```

`set` replaces every rule. Send the rules `get` returned as `basedOn`, or the API refuses the change with
a 409 when the stored rules differ: the request body has no flags for rules, so pass it with `-d`.

```sh
ca bucket lifecycle get logs-7f3a -o json \
  | jq '{rules: (.rules | map(.expirationDays = 30)), basedOn: .rules}' \
  | ca bucket lifecycle set logs-7f3a -d @- --yes
```

`protection` is read-only: object lock is chosen when the bucket is ordered.

## Access keys

```sh
ca bucket-key list|create|delete
ca bucket-key reveal-secret ci
ca bucket-key grant|revoke ci --bucket logs-7f3a
```

`-q` on `bucket-key create` and `reveal-secret` prints only the secret key, for a script to capture.
