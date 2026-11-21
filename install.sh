#!/bin/sh
# Install the CloudAfrica CLI (ca) on Linux or macOS.
#
#   curl -fsSL https://github.com/CloudAfrica/cloudafrica-cli/releases/latest/download/install.sh | sh
#
# CA_VERSION     a release tag such as v0.1.0 (default: the latest release)
# CA_INSTALL_DIR where to put ca and the cloudafrica link (default: /usr/local/bin
#                if writable, else ~/.local/bin)
set -eu

repo="https://github.com/CloudAfrica/cloudafrica-cli"

fail() {
    echo "install.sh: $*" >&2
    exit 1
}

need() {
    command -v "$1" >/dev/null 2>&1 || fail "needs $1"
}

# The body is one function called on the last line, so a download cut short
# by `curl | sh` runs nothing.
main() {
    need curl
    need tar
    need uname

    case "$(uname -s)" in
        Linux) os=linux ;;
        Darwin) os=darwin ;;
        *) fail "unsupported OS $(uname -s); on Windows use: scoop install ca" ;;
    esac

    case "$(uname -m)" in
        x86_64 | amd64) arch=amd64 ;;
        arm64 | aarch64) arch=arm64 ;;
        *) fail "unsupported architecture $(uname -m)" ;;
    esac

    if [ -n "${CA_VERSION:-}" ]; then
        tag="$CA_VERSION"
    else
        latest=$(curl -fsSLo /dev/null -w '%{url_effective}' "$repo/releases/latest") ||
            fail "cannot reach $repo"
        tag="${latest##*/}"
    fi
    case "$tag" in
        v[0-9]*) ;;
        *) fail "no release found (got '$tag')" ;;
    esac
    version="${tag#v}"

    if [ -n "${CA_INSTALL_DIR:-}" ]; then
        dir="$CA_INSTALL_DIR"
    elif [ -w /usr/local/bin ]; then
        dir=/usr/local/bin
    else
        dir="$HOME/.local/bin"
    fi
    mkdir -p "$dir"

    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' EXIT

    archive="cloudafrica-cli_${version}_${os}_${arch}.tar.gz"
    sums="cloudafrica-cli_${version}_SHA256SUMS"
    echo "Downloading ca $tag for $os/$arch"
    curl -fsSLo "$tmp/$archive" "$repo/releases/download/$tag/$archive" || fail "download failed: $archive"
    curl -fsSLo "$tmp/$sums" "$repo/releases/download/$tag/$sums" || fail "download failed: $sums"

    want=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/$sums")
    [ -n "$want" ] || fail "$archive is not listed in $sums"
    if command -v sha256sum >/dev/null 2>&1; then
        got=$(sha256sum "$tmp/$archive" | awk '{ print $1 }')
    elif command -v shasum >/dev/null 2>&1; then
        got=$(shasum -a 256 "$tmp/$archive" | awk '{ print $1 }')
    else
        fail "needs sha256sum or shasum to verify the download"
    fi
    [ "$got" = "$want" ] || fail "checksum mismatch for $archive"

    tar -xzf "$tmp/$archive" -C "$tmp" ca
    install -m 0755 "$tmp/ca" "$dir/ca" 2>/dev/null || {
        cp "$tmp/ca" "$dir/ca"
        chmod 0755 "$dir/ca"
    }
    ln -sf ca "$dir/cloudafrica"

    echo "Installed $dir/ca (and $dir/cloudafrica)"
    case ":$PATH:" in
        *":$dir:"*) ;;
        *) echo "Add $dir to your PATH to run ca." ;;
    esac
    "$dir/ca" version
}

main "$@"
