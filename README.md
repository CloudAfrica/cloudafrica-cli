# cloudafrica-cli

`ca` — the CloudAfrica command-line interface.

## Install

```sh
brew install cloudafrica/tap/ca                     # macOS, Linux

scoop bucket add cloudafrica https://github.com/CloudAfrica/scoop-bucket
scoop install ca                                    # Windows

curl -fsSL https://github.com/CloudAfrica/cloudafrica-cli/releases/latest/download/install.sh | sh
```

```powershell
irm https://github.com/CloudAfrica/cloudafrica-cli/releases/latest/download/install.ps1 | iex
```

Each also installs `cloudafrica`, the same binary, for when `ca` is taken on your machine; if
`ca` in the install directory is another program, `install.sh` and `install.ps1` leave it and install
only `cloudafrica`. Both take `CA_VERSION` (a tag) and `CA_INSTALL_DIR`; `install.ps1` defaults to
`%LOCALAPPDATA%\Programs\cloudafrica` and adds the directory to your user `PATH`. Release archives are on the
[releases page](https://github.com/CloudAfrica/cloudafrica-cli/releases); the checksum file is
signed with cosign:

```sh
cosign verify-blob cloudafrica-cli_<version>_SHA256SUMS \
  --bundle cloudafrica-cli_<version>_SHA256SUMS.sigstore.json \
  --certificate-identity-regexp '^https://github\.com/CloudAfrica/cloudafrica-cli/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## Update

```sh
ca update           # a ca from install.sh or install.ps1: download, check against SHA256SUMS, replace
ca update --check   # only report whether a newer release exists
brew upgrade ca     # Homebrew
scoop update ca     # Scoop
```

`ca update` on a Homebrew or Scoop install changes nothing and names that command instead. It checks the
download against the release's `SHA256SUMS`, the same check the install scripts make; it does not verify the
cosign signature. To go back to an earlier release, rerun the install script with `CA_VERSION`.

## Sign in

```sh
ca auth login     # opens the browser
ca auth login --scope write:billing   # also request a scope login does not ask for by default
ca auth elevate   # allow protected actions (such as vm delete, k8s upgrade-workers) for this sign-in
ca auth status
ca auth list
ca auth logout
```

Tokens are stored per profile in the user config directory (`~/.config/cloudafrica/<profile>.json`
on Linux), mode 0600. Pick a profile with `--profile`, else `CLOUDAFRICA_PROFILE`, else `default`.

Signing in grants scopes but never allows protected actions. `ca auth elevate` opens a portal page
where you step up and confirm the code it prints; the sign-in may then use protected actions for the
account's elevation window. Connected apps → "Allow protected actions" in the portal does the same.
A sign-in made before `ca auth login` requested a protected: scope must sign in again first. Scopes
added with `--scope` are kept, and requested again by every later `ca auth login` on that profile.

For automation, `CLOUDAFRICA_TOKEN` (a user API token) overrides any profile. `CLOUDAFRICA_ENDPOINT`
selects another API endpoint, which must be https (plain http only to a loopback address); a profile
can only be used against the endpoint it signed in to.

## Scripts

A script uses `CLOUDAFRICA_TOKEN`. For protected actions (such as `vm delete` and `k8s upgrade-workers`) the token must hold the
protected scope, and a person pairs it before the run:

```sh
export CLOUDAFRICA_TOKEN=cak_...
CLOUDAFRICA_ELEVATION=$(ca auth elevate) || exit
export CLOUDAFRICA_ELEVATION
./teardown.sh
```

Keep the assignment on its own line. `eval "$(ca auth elevate)"` and
`export CLOUDAFRICA_ELEVATION=$(ca auth elevate)` both return 0 when `ca auth elevate` fails, so the
`|| exit` would never fire. In PowerShell, check the exit code yourself:

```powershell
$env:CLOUDAFRICA_ELEVATION = ca auth elevate
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
```

In CI, store the printed value as a masked variable alongside the token. Running `ca auth elevate`
without capturing it prints the secret to the terminal, where it stays in the scrollback.

`ca auth elevate` opens a portal page where the person steps up and confirms the code; it then prints
only the pairing secret on stdout, which `ca` sends with every request while `CLOUDAFRICA_ELEVATION`
is set. The pairing lasts for the account's elevation window and only while that person's portal
session does. Run `ca auth elevate` again with the variable still set to pair the same value. The
secret is useless without the token, but treat it as one: do not log it or commit it.

## Commands

```sh
ca datacenter list
ca os-image list
ca vpc list|get|create|update|delete
ca vpc firewall-rule list --vpc web
ca vpc reserved-ip list|create|delete --vpc web
ca vpc vip create|delete --vpc web
ca vpc vip public-ip attach|detach --vpc web --vip lb1
ca vpc vip firewall-rule list|create|delete|set-description --vpc web --vip lb1
ca vip list
ca ip-address list
ca bucket list|get|create|delete
ca bucket usage logs-7f3a --hours 168
ca bucket versioning get|set logs-7f3a
ca bucket-key list|create|delete|reveal-secret
ca bucket-key grant|revoke ci --bucket logs-7f3a
ca ssh-key list|create|update|delete
ca vm list
ca vm get web-01
ca vm create --name web-01 --datacenter <dc-slug> --image <image-id> --vpc main \
  --vcpu 2 --ram-gib 4 --boot-disk-gib 40 --ssh-key laptop --accept-terms
ca vm start|shutdown|reboot|force-stop|restore web-01 [--no-wait] [--timeout 10m]
ca vm resize web-01 --vcpu 4 --ram-gib 8
ca vm rename web-01 --name db-01
ca vm delete web-01   # the VM must be stopped
ca vm disk|nic|public-ip|private-ip|ssh-key|vip|firewall-rule ... --vm web-01
ca vm backup list|create|delete|restore-to-new-vm|set-protection --vm web-01
ca vm backup-policy get|set|delete|preview-rotation --vm web-01
ca placement-group list|get|create|delete
ca placement-group member add|remove web-01 --placement-group spread
ca operation get|wait|cancel <operation-id>
ca api GET /v1/account   # any /v1 call
```

Every command has `--help`. The conventions:

- **Names or ids.** Where a command takes a VM, VPC, SSH key, VIP or placement group, a name works as
  well as an id. A name that matches nothing is refused; one that matches several is refused with their ids.
  When the names cannot be listed (the credential lacks the scope, say), the value is used as an id.
- **Waiting.** A command that starts an operation waits for it, printing progress on a terminal, and
  exits 1 if it fails. `--no-wait` prints the operation and returns; `ca operation wait` follows it later.
- **Confirmation.** Deletes and protected actions ask first on a terminal; off a terminal they need `--yes`.
  `k8s upgrade-workers` and `ca api` do not ask.
- **Request bodies.** Flags set the top-level fields; `-d/--data` takes the whole JSON body (`-d @file.json`
  or `-d @-` for stdin), and flags given with it win. `vm create` and the `k8s` commands take flags only;
  `ca api` takes `-d` alone.
- **Lists** follow every page. `--limit N` prints one page as the API returns it; the next page is
  `--cursor <next>`.
- **Output** is a table on a terminal and JSON otherwise; `-o table|json|yaml`, `-q` prints ids only.
  JSON and YAML are the API's response as returned, except that a list followed across pages prints its
  rows as one array (`--limit` prints the page as returned).

## Kubernetes

```sh
ca k8s list               # UPGRADE: the workers' upgrade state
ca k8s get <id>           # versions, upgrade state and deadlines, worker targets, nodes
ca k8s upgrade-workers <id> [--version v1.33.1] [--no-wait] [--timeout 30m]
ca k8s kubeconfig <id> [--managed-vpn] > prod.kubeconfig
ca k8s token <id>         # the 15-minute credential kubectl reads; the kubeconfig runs it
```

`ca kubernetes` is the same command. `upgrade-workers` is a protected action. Each worker drains and
restarts in turn, so a workload with one replica is down while its node restarts. Without `--version`
it takes the newest worker target. If there is none, it says so on stderr. With `--no-wait` it then
exits 0 when the workers are CURRENT, DEADLINE or UPGRADE_AVAILABLE (worker targets never go past the
control plane's version), so a scheduled `--no-wait` run stays quiet until a target is offered. Any other
state with no target exits 1, and so does waiting (the default) unless the workers are CURRENT.
Running it again after an upgrade finishes starts the next one, if there is a newer target.

`ca k8s kubeconfig` prints the cluster's kubeconfig with its user set to run `ca k8s token`, in place of
the downloaded `sh -c curl` command, so it also works on native Windows. It holds no credential. kubectl
runs `ca k8s token` whenever it needs a token: that mints a 15-minute cluster-admin token and needs
`CLOUDAFRICA_TOKEN` (your own API token with `protected:compute`)
and `CLOUDAFRICA_ELEVATION` paired with it, as for a script (see Scripts). A `ca auth login` sign-in
cannot mint a cluster token. The kubeconfig pins `CLOUDAFRICA_ENDPOINT` to the API it came from.

```sh
export CLOUDAFRICA_TOKEN=cak_...
CLOUDAFRICA_ELEVATION=$(ca auth elevate) || exit
export CLOUDAFRICA_ELEVATION
ca k8s kubeconfig <id> > prod.kubeconfig
KUBECONFIG=prod.kubeconfig kubectl get nodes
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | API error, or a failed operation |
| 2 | usage error, or `ca update` refused (a development build, or a directory it cannot write to) |
| 3 | waiting for an operation timed out (`--timeout`), `ca auth elevate` was not confirmed in 5 minutes, or `ca update` timed out reaching GitHub |
| 4 | not signed in, or the session ended (`ca auth login`) |
| 5 | protected action refused: run `ca auth elevate` (not paired), or `ca auth login` (the sign-in lacks the scope) |

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
