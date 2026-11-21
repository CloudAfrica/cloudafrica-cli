#!/usr/bin/env bash
# Tests install.sh and the release layout it depends on, against the archives
# of `goreleaser release --snapshot` in dist/. curl is faked: it serves files
# from a copy of dist/, so install.sh runs unmodified.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
dist="$root/dist"
sums=$(ls "$dist"/cloudafrica-cli_*_SHA256SUMS)
version=$(basename "$sums" | sed 's/^cloudafrica-cli_\(.*\)_SHA256SUMS$/\1/')
case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    *) arch=arm64 ;;
esac
os=$(uname -s | tr '[:upper:]' '[:lower:]')
archive="cloudafrica-cli_${version}_${os}_${arch}.tar.gz"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/fakebin"
cat >"$work/fakebin/curl" <<'CURL'
#!/usr/bin/env bash
out=""
while [ $# -gt 1 ]; do
    case "$1" in
        -fsSLo) out="$2"; shift 2 ;;
        # The latest-release lookup: print where releases/latest redirects.
        -w) [ -n "${LATEST:-}" ] || exit 22; printf '%s' "$LATEST"; exit 0 ;;
        *) shift ;;
    esac
done
src="$SERVE/$(basename "$1")"
[ -f "$src" ] || { echo "curl: (22) 404 $1" >&2; exit 22; }
cp "$src" "$out"
CURL
chmod +x "$work/fakebin/curl"

pass=0
check() {
    echo "ok - $1"
    pass=$((pass + 1))
}
die() {
    echo "FAIL - $1" >&2
    exit 1
}

serve() {
    rm -rf "$work/serve"
    mkdir -p "$work/serve"
    cp "$dist/$archive" "$sums" "$work/serve/"
}

run_install() {
    PATH="$work/fakebin:$PATH" SERVE="$work/serve" CA_VERSION="${CA_VERSION-v$version}" CA_INSTALL_DIR="$1" \
        sh "$root/install.sh" >"$work/out" 2>&1
}

# Release layout.
tar tzf "$dist/$archive" | grep -qx ca || die "$archive has no ca at its root"
check "unix archive has ca at its root"
for a in amd64 arm64; do
    zip="$dist/cloudafrica-cli_${version}_windows_${a}.zip"
    listing=$(unzip -Z1 "$zip")
    grep -qx ca.exe <<<"$listing" || die "$zip lacks ca.exe"
    grep -qx cloudafrica.exe <<<"$listing" || die "$zip lacks cloudafrica.exe"
done
check "windows zips carry ca.exe and cloudafrica.exe"
[ "$(jq -c '.architecture."64bit".bin | sort' "$dist/scoop/ca.json")" = '["ca.exe","cloudafrica.exe"]' ] ||
    die "scoop manifest does not shim both names"
check "scoop manifest shims ca and cloudafrica"
grep -q 'bin.install_symlink "ca" => "cloudafrica"' "$dist/homebrew/Formula/ca.rb" ||
    die "formula does not link cloudafrica"
check "formula links cloudafrica"

# Fresh install, then an upgrade over our own binary.
serve
d="$work/fresh"
run_install "$d" || die "fresh install failed: $(cat "$work/out")"
[ -x "$d/ca" ] && [ "$(readlink "$d/cloudafrica")" = ca ] || die "fresh install left $(ls -l "$d")"
"$d/ca" version | grep -q "^ca $version (" || die "installed ca reports the wrong version"
check "fresh install puts ca and the cloudafrica link"
run_install "$d" || die "upgrade over our own ca failed: $(cat "$work/out")"
check "reinstall over our own ca succeeds"

# Another program named ca is kept; the CLI installs as cloudafrica.
d="$work/foreign"
mkdir -p "$d"
printf '#!/bin/sh\necho certificate-authority\n' >"$d/ca"
chmod +x "$d/ca"
run_install "$d" || die "install beside a foreign ca failed: $(cat "$work/out")"
grep -q certificate-authority "$d/ca" || die "foreign ca was modified"
[ -f "$d/cloudafrica" ] && [ ! -L "$d/cloudafrica" ] || die "cloudafrica is not a binary: $(ls -l "$d")"
"$d/cloudafrica" version | grep -q "^ca $version (" || die "cloudafrica is not the CLI"
grep -q "installed only $d/cloudafrica" "$work/out" || die "no notice for a foreign ca"
check "a foreign ca is kept and the CLI installs as cloudafrica"
run_install "$d" || die "reinstall beside a foreign ca failed: $(cat "$work/out")"
check "reinstall beside a foreign ca replaces our cloudafrica"

# Both names taken by other programs: refused, both kept.
d="$work/foreign-both"
mkdir -p "$d"
printf '#!/bin/sh\necho certificate-authority\n' >"$d/ca"
echo other >"$d/cloudafrica"
chmod +x "$d/ca"
if run_install "$d"; then die "installed over foreign ca and cloudafrica"; fi
grep -q "are other programs" "$work/out" || die "both-foreign refusal said: $(cat "$work/out")"
grep -q certificate-authority "$d/ca" && [ "$(cat "$d/cloudafrica")" = other ] || die "a foreign binary was modified"
check "foreign ca and cloudafrica are refused and kept"

# A foreign cloudafrica is kept; ca still installs.
d="$work/foreign-alias"
mkdir -p "$d"
echo other >"$d/cloudafrica"
run_install "$d" || die "install beside a foreign cloudafrica failed: $(cat "$work/out")"
[ "$(cat "$d/cloudafrica")" = other ] && [ -x "$d/ca" ] || die "foreign cloudafrica not kept"
grep -q "Left $d/cloudafrica alone" "$work/out" || die "no warning for a foreign cloudafrica"
check "a foreign cloudafrica is kept with a warning"

# Our cloudafrica binary, left by a foreign ca since removed, becomes the link.
d="$work/foreign-gone"
mkdir -p "$d"
cp "$work/foreign/cloudafrica" "$d/cloudafrica"
run_install "$d" || die "install over our own cloudafrica binary failed: $(cat "$work/out")"
[ "$(readlink "$d/cloudafrica")" = ca ] || die "our cloudafrica binary was not relinked: $(ls -l "$d")"
check "our cloudafrica binary is replaced by the link once ca is free"

# Without CA_VERSION the tag comes from where releases/latest redirects.
d="$work/latest"
CA_VERSION="" LATEST="https://github.com/CloudAfrica/cloudafrica-cli/releases/tag/v$version" run_install "$d" ||
    die "latest install failed: $(cat "$work/out")"
"$d/ca" version | grep -q "^ca $version (" || die "latest install put the wrong version"
check "without CA_VERSION the latest release is installed"
d="$work/no-release"
if CA_VERSION="" LATEST="https://github.com/CloudAfrica/cloudafrica-cli/releases" run_install "$d"; then
    die "installed with no release"
fi
grep -q "no release found" "$work/out" || die "no-release said: $(cat "$work/out")"
check "a repository with no release is refused"

# A tampered archive is refused before anything is installed.
serve
printf x >>"$work/serve/$archive"
d="$work/tampered"
if run_install "$d"; then die "installed a tampered archive"; fi
grep -q "checksum mismatch" "$work/out" || die "tampered archive said: $(cat "$work/out")"
[ ! -e "$d/ca" ] || die "tampered archive left $d/ca"
check "a tampered archive is refused"

# An archive missing from the checksum file is refused.
serve
grep -v " $archive\$" "$sums" >"$work/serve/$(basename "$sums")"
d="$work/unlisted"
if run_install "$d"; then die "installed an archive missing from the checksums"; fi
grep -q "is not listed in" "$work/out" || die "unlisted archive said: $(cat "$work/out")"
[ ! -e "$d/ca" ] || die "unlisted archive left $d/ca"
check "an archive missing from the checksum file is refused"

echo "$pass passed"
