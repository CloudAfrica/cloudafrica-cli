# Conventions

Every command follows these, and every command has `--help`.

- **Names or ids.** Where a command takes a VM, VPC, SSH key, VIP, placement group, bucket, access key or
  project as its argument or through a flag such as `--vm`, a name works as well as an id; a member is
  named by email and a support ticket by its reference number. A flag ending in `-resource-id`
  (`--vpc-resource-id`, `--bucket-resource-id`) is sent as given and needs the id: `ca vpcs get main -q`
  prints it. A name that matches nothing is refused; one that matches several is refused with
  their ids. When the names cannot be listed (the credential lacks the scope, say), the value is used as
  an id.
- **Waiting.** A command that starts an operation waits for it, printing progress on a terminal, and
  exits 1 if it fails. `--no-wait` prints the operation and returns; `ca operations wait` follows it later.
- **Confirmation.** Deletes and protected actions ask first on a terminal; off a terminal they need
  `--yes`.
- **Request bodies.** Flags set the top-level fields; `-d/--data` takes the whole JSON body
  (`-d @file.json` or `-d @-` for stdin), and flags given with it win. `vms create` takes flags
  only.
- **Lists** follow every page. `--limit N` prints one page as the API returns it; the next page is
  `--limit N --cursor <next>`.
- **Output** is a table on a terminal and JSON otherwise; `-o table`, `-o json` or `-o yaml` chooses one, and `-q` prints ids only (the
  secret key for `bucket-keys create` and `reveal-secret`). JSON and YAML are the API's response as
  returned, with two exceptions. A list followed across pages prints its rows as one array (`--limit`
  prints the page as returned). A command that starts an operation prints the finished operation, or
  with `--no-wait` just `{"operationId": "..."}`.

## Operations

`ca operations get` shows an operation, `ca operations wait` waits for it to finish and `ca operations cancel` asks for it to be cancelled. Each takes the operation's id.

For example, following an operation that `--no-wait` left running:

```sh
ca operations wait <operation-id>
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | API error, or a failed operation |
| 2 | usage error, or `ca update` refused (a development build, or a directory it cannot write to) |
| 3 | waiting for an operation timed out (`--timeout`), the pairing of `CLOUDAFRICA_TOKEN` by `ca auth elevate` was not confirmed in 5 minutes, or `ca update` timed out reaching GitHub |
| 4 | not signed in, or the session ended (`ca auth login`) |
| 5 | protected action refused: run `ca auth login` and approve it (the sign-in is not paired, or lacks the scope); with `CLOUDAFRICA_TOKEN`, run `ca auth elevate` |
