# cloudafrica-cli

`ca` — the CloudAfrica command-line interface.

## Install

```sh
brew install cloudafrica/tap/ca                     # macOS, Linux

scoop bucket add cloudafrica https://github.com/CloudAfrica/scoop-bucket
scoop install ca                                    # Windows

curl -fsSL https://github.com/CloudAfrica/cloudafrica-cli/releases/latest/download/install.sh | sh
```

Each also installs `cloudafrica`, the same binary, for when `ca` is taken on your machine; if
`ca` in the install directory is another program, `install.sh` leaves it and installs only `cloudafrica`.
`install.sh` takes `CA_VERSION` (a tag) and `CA_INSTALL_DIR`. Release archives are on the
[releases page](https://github.com/CloudAfrica/cloudafrica-cli/releases); the checksum file is
signed with cosign:

```sh
cosign verify-blob cloudafrica-cli_<version>_SHA256SUMS \
  --bundle cloudafrica-cli_<version>_SHA256SUMS.sigstore.json \
  --certificate-identity-regexp '^https://github\.com/CloudAfrica/cloudafrica-cli/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## Sign in

```sh
ca auth login     # opens the browser
ca auth elevate   # allow protected actions (vm delete) for this sign-in
ca auth status
ca auth list
ca auth logout
```

Tokens are stored per profile in the user config directory (`~/.config/cloudafrica/<profile>.json`
on Linux), mode 0600. Pick a profile with `--profile`, else `CLOUDAFRICA_PROFILE`, else `default`.

Signing in grants scopes but never allows protected actions. `ca auth elevate` opens a portal page
where you step up and confirm the code it prints; the sign-in may then use protected actions for the
account's elevation window. Connected apps → "Allow protected actions" in the portal does the same.
A sign-in made before `ca auth login` requested protected:compute must sign in again first.

For automation, `CLOUDAFRICA_TOKEN` (an API key) overrides any profile. `CLOUDAFRICA_ENDPOINT`
selects another API endpoint, which must be https (plain http only to a loopback address); a profile
can only be used against the endpoint it signed in to.

## Virtual machines

```sh
ca datacenter list   # slugs for --datacenter
ca vpc list          # ids for --vpc
ca ssh-key list      # ids for --ssh-key
ca vm list
ca vm get <id>
ca vm create --name web --datacenter <dc-slug> --image <image-id> --vpc <vpc> \
  --vcpu 2 --ram-gib 4 --boot-disk-gib 40 --ssh-key <key-id> --accept-terms --wait
ca vm start|shutdown|reboot|force-stop <id> [--wait] [--timeout 10m]
ca vm resize <id> --vcpu 4 --ram-gib 8
ca vm rename <id> <name>
ca vm delete <id> --yes   # the VM must be stopped
ca operation get|wait <operation-id>
ca api GET /v1/account   # any /v1 call
```

Output is a table on a terminal and JSON otherwise; `-o table|json|yaml`, `-q` prints ids only.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | API error, or a failed operation |
| 2 | usage error |
| 3 | `--wait` timed out, or `ca auth elevate` was not confirmed in 5 minutes |
| 4 | not signed in, or the session ended (`ca auth login`) |
| 5 | protected action refused: run `ca auth elevate` (not paired), or `ca auth login` (the sign-in lacks the scope); `ca` cannot pair an API key, and cannot perform a protected action whose scope `ca auth login` does not request |

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
