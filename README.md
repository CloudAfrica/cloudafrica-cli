# cloudafrica-cli

`ca` — the CloudAfrica command-line interface: the CloudAfrica API from a terminal or a script.

## Install

Then run `ca version` to check it worked.

### macOS and Linux

With Homebrew:

```sh
brew install cloudafrica/tap/ca
```

Without Homebrew, use the install script:

```sh
curl -fsSL https://github.com/CloudAfrica/cloudafrica-cli/releases/latest/download/install.sh | sh
```

### Windows

With Scoop:

```powershell
scoop bucket add cloudafrica https://github.com/CloudAfrica/scoop-bucket
scoop install ca
```

Without Scoop, run the install script in PowerShell:

```powershell
irm https://github.com/CloudAfrica/cloudafrica-cli/releases/latest/download/install.ps1 | iex
```

### Install script options

Each install also installs `cloudafrica`, the same binary, for when `ca` is taken on your machine; if
`ca` in the install directory is another program, `install.sh` and `install.ps1` leave it and install
only `cloudafrica`. Both take `CA_VERSION` (a tag) and `CA_INSTALL_DIR`; `install.ps1` defaults to
`%LOCALAPPDATA%\Programs\cloudafrica` and adds the directory to your user `PATH`.

### Download a release

Release archives are on the [releases page](https://github.com/CloudAfrica/cloudafrica-cli/releases)
(macOS and Linux as `.tar.gz`, Windows as `.zip`, each for `amd64` and `arm64`), with a `SHA256SUMS` file
to check them against. The checksum file is signed with cosign:

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

## Quick start

```sh
ca auth login
ca datacenters list
ca os-images list
ca vpcs create --name main
ca ssh-keys create --name laptop --public-key "$(cat ~/.ssh/id_ed25519.pub)"
ca vms create --name web-01 --datacenter <dc-slug> --image <image-id> --vpc main \
  --vcpu 2 --ram-gib 4 --boot-disk-gib 40 --ssh-key laptop --public-ip --accept-terms
ca vms public-ips list --vm web-01
```

`vms create` waits until the VM is built. When you are done, shut it down and delete it. Deleting is a
[protected action](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/auth.md#protected-actions), so run `ca auth elevate` first:

```sh
ca vms shutdown web-01
ca auth elevate
ca vms delete web-01
```

## Shell completion

```sh
source <(ca completion bash)          # bash; needs the bash-completion package
source <(ca completion zsh)           # zsh
ca completion fish | source           # fish
ca completion powershell | Out-String | Invoke-Expression   # PowerShell
```

`ca completion <shell> --help` shows how to load it in every new session.

## Documentation

- [Signing in and protected actions](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/auth.md), including scripts and CI
- [Conventions](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/conventions.md): names or ids, waiting, confirmation, output, exit codes
- [Environment variables](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/environment.md)
- Commands by domain:
  - [Compute](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/compute.md): VMs, disks, backups, placement groups, SSH keys, images
  - [Networking](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/networking.md): VPCs, reserved IPs, VIPs, firewall rules
  - [Object storage](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/object-storage.md): buckets, objects and access keys
  - [Account, team and billing](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/account.md)
  - [Support](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/support.md)
- [Command reference](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/docs/reference/ca.md): every command and flag, generated from `ca` itself
- [Contributing](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/CONTRIBUTING.md)

Licensed under the Apache License, Version 2.0. See [LICENSE](https://github.com/CloudAfrica/cloudafrica-cli/blob/main/LICENSE).
