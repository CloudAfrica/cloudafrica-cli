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

# ours reports whether $1 is this CLI: `ca version` prints "ca <version> (".
ours() {
    case "$("$1" version </dev/null 2>/dev/null | head -n 1)" in
        "ca "*" ("*) return 0 ;;
        *) return 1 ;;
    esac
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
        *) fail "unsupported OS $(uname -s); on Windows use install.ps1 or scoop" ;;
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
    if [ -e "$dir/ca" ] && ! ours "$dir/ca"; then
        # The name clash the cloudafrica alias exists for: keep the other ca.
        if [ -e "$dir/cloudafrica" ] && ! ours "$dir/cloudafrica"; then
            fail "$dir/ca and $dir/cloudafrica are other programs; set CA_INSTALL_DIR to install elsewhere"
        fi
        rm -f "$dir/cloudafrica"
        install -m 0755 "$tmp/ca" "$dir/cloudafrica"
        echo "$dir/ca is another program, so this installed only $dir/cloudafrica" >&2
        bin="$dir/cloudafrica"
    else
        install -m 0755 "$tmp/ca" "$dir/ca"
        if [ -L "$dir/cloudafrica" ] && [ "$(readlink "$dir/cloudafrica")" = ca ]; then
            :
        elif [ ! -L "$dir/cloudafrica" ] && [ -f "$dir/cloudafrica" ] && ours "$dir/cloudafrica"; then
            # Installed while another ca held the name; that ca has gone.
            rm -f "$dir/cloudafrica"
            ln -s ca "$dir/cloudafrica"
        elif [ -e "$dir/cloudafrica" ] || [ -L "$dir/cloudafrica" ]; then
            echo "Left $dir/cloudafrica alone: it is not a link to ca" >&2
        else
            ln -s ca "$dir/cloudafrica"
        fi
        bin="$dir/ca"
    fi

    echo "Installed $bin"
    case ":$PATH:" in
        *":$dir:"*) ;;
        *) echo "Add $dir to your PATH to run $(basename "$bin")." ;;
    esac
    "$bin" version
}

main "$@"
