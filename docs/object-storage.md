# Object storage

Buckets and the access keys that reach them. Every command is in the [reference](reference/ca.md).

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

## Access keys

```sh
ca bucket-key list|create|delete
ca bucket-key reveal-secret ci
ca bucket-key grant|revoke ci --bucket logs-7f3a
```

`-q` on `bucket-key create` and `reveal-secret` prints only the secret key, for a script to capture.
