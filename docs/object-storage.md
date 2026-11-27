# Object storage

Buckets, their objects, and the access keys that reach them. Every command is in the [reference](reference/ca.md).

Some of these commands, deletes among them, are protected actions: sign in with `ca auth login` and approve them on the consent page. A
protected command that is refused exits 5 and prints the remedy. See [Protected actions](auth.md#protected-actions),
or [Scripts and CI](auth.md#scripts-and-ci) when you sign in with a token.

## Buckets

```sh
ca buckets list|get|create|delete
ca buckets usage logs-7f3a --hours 168
ca buckets versioning get|set logs-7f3a
```

`ca buckets delete` deletes the bucket and everything in it.

## Objects

```sh
ca buckets objects list logs-7f3a --prefix reports/2026/
ca buckets objects get logs-7f3a --key reports/q3.pdf
ca buckets objects versions logs-7f3a --key reports/q3.pdf
ca buckets objects list-deleted logs-7f3a
```

`list` and `list-deleted` follow every page; `--limit N` prints one page. `-q` prints keys for `list`, every version id for `versions`, and the delete markers' version ids for `list-deleted`.

The API moves an object's bytes through a short-lived URL, not through itself.

```bash
# Download: the URL works for about a minute and needs no credentials. Protected: sign in with protected scopes first.
URL=$(ca buckets objects download-url logs-7f3a --key reports/q3.pdf --yes -q) &&
  curl -fsS -o q3.pdf "$URL"

# Upload one new object of at most 25 MiB. The PUT must send every header the API returns, and exactly
# --content-length bytes.
SIZE=$(wc -c < q3.pdf | tr -d ' ')
SPEC=$(ca buckets objects upload-url logs-7f3a --key reports/q3-copy.pdf --content-type application/pdf \
  --content-length "$SIZE" -o json) &&
  curl -fsS -X PUT "$(jq -r .url <<<"$SPEC")" \
    -K <(jq -r '.headers | to_entries[] | "header = \"\(.key): \(.value)\""' <<<"$SPEC") \
    --data-binary @q3.pdf
```

An upload to a key that already exists is refused. Overwriting, deleting or restoring an object, and any
file over 25 MiB, go through an S3 client with a [bucket key](#access-keys): point it at `s3.bigstorage.io`.

## Lifecycle, CORS and protection

```sh
ca buckets lifecycle get|set logs-7f3a
ca buckets cors get|set logs-7f3a
ca buckets protection logs-7f3a
```

`set` replaces every rule. Send the rules `get` returned as `basedOn`, or the API refuses the change with
a 409 when the stored rules differ: the request body has no flags for rules, so pass it with `-d`.

```sh
ca buckets lifecycle get logs-7f3a -o json \
  | jq '{rules: (.rules | map(.expirationDays = 30)), basedOn: .rules}' \
  | ca buckets lifecycle set logs-7f3a -d @- --yes
```

`protection` is read-only: object lock is chosen when the bucket is ordered.

## Access keys

```sh
ca bucket-keys list|create|delete
ca bucket-keys reveal-secret ci
ca bucket-keys grant|revoke ci --bucket logs-7f3a
```

`-q` on `bucket-keys create` and `reveal-secret` prints only the secret key, for a script to capture.
