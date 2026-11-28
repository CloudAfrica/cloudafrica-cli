# Object storage

Buckets, their objects, and the access keys that reach them. Every command is in the [reference](reference/ca.md).

Some commands, `buckets delete` among them, are [protected actions](auth.md#protected-actions).

## Buckets

`ca buckets list` shows your buckets and `ca buckets get` one of them. `ca buckets create` orders a bucket and `ca buckets delete` deletes it and everything in it.

`ca buckets usage` shows a bucket's storage and traffic over a window. `ca buckets versioning get` shows its versioning state and `ca buckets versioning set` changes it.

For example, a week of usage:

```sh
ca buckets usage logs-7f3a --hours 168
```

## Objects

`ca buckets objects list` shows the objects in a bucket and `ca buckets objects get` one object's metadata, tags and retention. `ca buckets objects versions` shows the versions of one object, and `ca buckets objects list-deleted` the deleted objects that can still be recovered. `get` and `versions` take `--key`.

For example, the objects under one prefix:

```sh
ca buckets objects list logs-7f3a --prefix reports/2026/
```

`list` and `list-deleted` follow every page; `--limit N` prints one page. `-q` prints keys for `list`, every version id for `versions`, and the delete markers' version ids for `list-deleted`.

The API moves an object's bytes through a short-lived URL, not through itself. `ca buckets objects download-url` mints a URL that downloads one object. The URL works for about a minute and needs no credentials; the command is a protected action, so sign in with protected scopes first. For example, downloading one object:

```bash
URL=$(ca buckets objects download-url logs-7f3a --key reports/q3.pdf --yes -q) &&
  curl -fsS -o q3.pdf "$URL"
```

`ca buckets objects upload-url` mints a URL that uploads one new object of at most 25 MiB. The PUT must send every header the API returns, and exactly `--content-length` bytes. For example, uploading a copy of the file:

```bash
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

`ca buckets lifecycle get` shows a bucket's lifecycle rules and `ca buckets lifecycle set` replaces them. `ca buckets cors get` and `ca buckets cors set` do the same for its CORS rules. `ca buckets protection` shows its object lock and default encryption; it is read-only, because object lock is chosen when the bucket is ordered.

`set` replaces every rule. Send the rules `get` returned as `basedOn`, or the API refuses the change with
a 409 when the stored rules differ: the request body has no flags for rules, so pass it with `-d`.

For example, expiring current versions after 30 days:

```sh
ca buckets lifecycle get logs-7f3a -o json \
  | jq '{rules: (.rules | map(.expirationDays = 30)), basedOn: .rules}' \
  | ca buckets lifecycle set logs-7f3a -d @- --yes
```

## Access keys

`ca bucket-keys list` shows the account's access keys, `ca bucket-keys create` makes one and `ca bucket-keys delete` deletes one. `ca bucket-keys reveal-secret` shows a key's secret. `ca bucket-keys grant` gives a key access to a bucket and `ca bucket-keys revoke` takes it away.

`-q` on `bucket-keys create` and `reveal-secret` prints only the secret key, for a script to capture.

For example, giving the key `ci` access to a bucket:

```sh
ca bucket-keys grant ci --bucket logs-7f3a
```
