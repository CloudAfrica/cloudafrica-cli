# Scripting with ca

`ca` behaves the same in a script as at a terminal, with three differences: it prints JSON instead of
a table, it prints no progress, and it asks no questions. A delete or protected action that would ask
at a terminal needs `--yes` off one. The [conventions](conventions.md) and the
[exit codes](conventions.md#exit-codes) apply throughout.

## Ids and fields

`-q` prints ids only, one per line (for `bucket-keys create` and `reveal-secret`, the secret key
instead). A name works for most resources, so a script rarely needs their ids, but a flag ending in
`-resource-id` does:

```sh
vpc=$(ca vpcs get main -q)
```

For any other field, read the JSON with `jq`. A VM's public IP, for example:

```sh
ca vms get web-01 -o json | jq -r .publicIp
```

A list prints every page as one JSON array:

```sh
ca vms list -o json | jq -r '.[] | select(.status == "running") | .name'
```

## Waiting

A command that starts an operation waits for it and exits 1 if it fails, so `set -e` or `|| exit`
stops the script at the first failure. With `-q`, a create prints the new resource's id once it is
done:

```sh
vm=$(ca vms create --name web-01 --datacenter <datacenter-slug> --image <image-id> --vpc main \
    --vcpu 2 --ram-gib 4 --boot-disk-gib 40 --ssh-key laptop --public-ip --accept-terms -q) || exit
```

`--timeout` sets how long to wait; running out of time exits 3 and leaves the operation running. To
start several operations at once, pass `--no-wait`, collect the operation ids, and wait for each:

```sh
op1=$(ca vms reboot web-01 --no-wait -q)
op2=$(ca vms reboot web-02 --no-wait -q)
ca operations wait "$op1" && ca operations wait "$op2"
```

## Retrying

Only run a write again once you know the first one failed. When `ca` exits without an answer from the
API, a timeout or a dropped connection say, the write may still have happened, and running it again
can do it twice. Check first: `ca vms get web-01` after a create, or `ca operations get` with the
operation id.

## In CI

A pipeline signs in with `CLOUDAFRICA_TOKEN`, a user API token, which overrides any profile. For
protected actions such as `vms delete`, a person pairs the token before the run with `ca auth elevate`
and the pipeline stores the result as `CLOUDAFRICA_ELEVATION`; [Scripts and CI](auth.md#scripts-and-ci)
has the steps. A teardown job then runs:

```sh
ca vms delete web-01 --yes
```

Exit code 5 means the token is not paired, or lacks the scope: pair it again, and check the token's
scopes if that does not help. The [exit codes](conventions.md#exit-codes) list the rest.
