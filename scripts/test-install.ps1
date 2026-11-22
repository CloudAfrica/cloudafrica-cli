# Tests install.ps1 on Windows against the archives of `goreleaser release --snapshot` in dist/.
# Invoke-WebRequest is faked by a function that serves files from a copy of dist/, and each install
# runs the way users run it, `<script> | Invoke-Expression`, in a fresh pwsh, so install.ps1 runs
# unmodified.
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$installer = Join-Path $root 'install.ps1'
$sums = Get-Item (Join-Path $root 'dist\cloudafrica-cli_*_SHA256SUMS')
$version = $sums.Name -replace '^cloudafrica-cli_(.*)_SHA256SUMS$', '$1'
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$archive = "cloudafrica-cli_${version}_windows_${arch}.zip"

$work = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $work | Out-Null
$fake = Join-Path $work 'fake.ps1'
Set-Content -Path $fake -Value @'
function Invoke-WebRequest([string] $Uri, [string] $OutFile, [switch] $UseBasicParsing) {
    if ($Uri.EndsWith('/releases/latest')) {
        # The latest-release lookup: answer with where releases/latest redirects.
        if (-not $env:LATEST)
            { throw 'unreachable' }
        return [pscustomobject] @{ BaseResponse = [pscustomobject] @{
            RequestMessage = [pscustomobject] @{ RequestUri = [uri] $env:LATEST } } }
    }
    $src = Join-Path $env:SERVE ($Uri -split '/')[-1]
    if (-not (Test-Path $src))
        { throw "404 $Uri" }
    Copy-Item $src $OutFile
}
'@
$out = Join-Path $work 'out'
$script:pass = 0

function Check([string] $Name) {
    Write-Host "ok - $Name"
    $script:pass++
}

function Die([string] $Message) {
    Write-Host "FAIL - $Message"
    exit 1
}

function Serve {
    $serve = Join-Path $work 'serve'
    Remove-Item $serve -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Path $serve | Out-Null
    Copy-Item (Join-Path $root "dist\$archive"), $sums.FullName $serve
}

function Output { Get-Content -Raw $out }

# Runs install.ps1 into $Dir (none: the default) and reports whether it succeeded.
function Invoke-Install([string] $Dir, [string] $Version = "v$version", [string] $Latest = '') {
    $env:SERVE = Join-Path $work 'serve'
    $env:CA_VERSION = $Version
    $env:CA_INSTALL_DIR = $Dir
    $env:LATEST = $Latest
    pwsh -NoProfile -Command ". '$fake'; Get-Content -Raw '$installer' | Invoke-Expression" *> $out
    return $LASTEXITCODE -eq 0
}

function Reports-Version([string] $Bin) {
    return "$(& $Bin version)" -like "ca $version (*"
}

function User-Path-Count([string] $Dir) {
    $path = (Get-Item 'HKCU:\Environment').GetValue('Path', '', 'DoNotExpandEnvironmentNames')
    return @($path -split ';' | Where-Object { $_.TrimEnd('\') -eq $Dir }).Count
}

$foreign = Join-Path $env:SystemRoot 'System32\whoami.exe'
$foreignHash = (Get-FileHash $foreign).Hash

# Fresh install, then an upgrade over our own binaries.
Serve
$d = Join-Path $work 'fresh'
if (-not (Invoke-Install $d)) { Die "fresh install failed: $(Output)" }
if (-not (Reports-Version "$d\ca.exe")) { Die 'installed ca.exe reports the wrong version' }
if (-not (Reports-Version "$d\cloudafrica.exe")) { Die 'installed cloudafrica.exe reports the wrong version' }
if ((User-Path-Count $d) -ne 1) { Die "the user PATH does not hold $d once" }
Check 'fresh install puts ca.exe and cloudafrica.exe on the user PATH'
if (-not (Invoke-Install $d)) { Die "upgrade over our own ca failed: $(Output)" }
if ((User-Path-Count $d) -ne 1) { Die "a reinstall added $d to the user PATH again" }
Check 'reinstall over our own ca succeeds and leaves the user PATH alone'

# Another program named ca is kept; the CLI installs as cloudafrica.
$d = Join-Path $work 'foreign'
New-Item -ItemType Directory -Path $d | Out-Null
Copy-Item $foreign "$d\ca.exe"
if (-not (Invoke-Install $d)) { Die "install beside a foreign ca failed: $(Output)" }
if ((Get-FileHash "$d\ca.exe").Hash -ne $foreignHash) { Die 'foreign ca was modified' }
if (-not (Reports-Version "$d\cloudafrica.exe")) { Die 'cloudafrica.exe is not the CLI' }
if ((Output) -notlike "*installed only $d\cloudafrica.exe*") { Die "no notice for a foreign ca: $(Output)" }
Check 'a foreign ca is kept and the CLI installs as cloudafrica'
if (-not (Invoke-Install $d)) { Die "reinstall beside a foreign ca failed: $(Output)" }
Check 'reinstall beside a foreign ca replaces our cloudafrica'

# Both names taken by other programs: refused, both kept.
$d = Join-Path $work 'foreign-both'
New-Item -ItemType Directory -Path $d | Out-Null
Copy-Item $foreign "$d\ca.exe"
Set-Content "$d\cloudafrica.exe" 'other'
if (Invoke-Install $d) { Die 'installed over foreign ca and cloudafrica' }
if ((Output) -notlike '*are other programs*') { Die "both-foreign refusal said: $(Output)" }
if ((Get-FileHash "$d\ca.exe").Hash -ne $foreignHash -or (Get-Content "$d\cloudafrica.exe") -ne 'other') {
    Die 'a foreign binary was modified'
}
Check 'foreign ca and cloudafrica are refused and kept'

# A foreign cloudafrica is kept; ca still installs.
$d = Join-Path $work 'foreign-alias'
New-Item -ItemType Directory -Path $d | Out-Null
Set-Content "$d\cloudafrica.exe" 'other'
if (-not (Invoke-Install $d)) { Die "install beside a foreign cloudafrica failed: $(Output)" }
if ((Get-Content "$d\cloudafrica.exe") -ne 'other' -or -not (Reports-Version "$d\ca.exe")) {
    Die 'foreign cloudafrica not kept'
}
if ((Output) -notlike "*Left $d\cloudafrica.exe alone*") { Die "no warning for a foreign cloudafrica: $(Output)" }
Check 'a foreign cloudafrica is kept with a warning'

# Without CA_INSTALL_DIR the CLI goes under LOCALAPPDATA.
$d = Join-Path $env:LOCALAPPDATA 'Programs\cloudafrica'
if (-not (Invoke-Install '')) { Die "default-directory install failed: $(Output)" }
if (-not (Reports-Version "$d\ca.exe")) { Die "the default install did not put ca.exe in $d" }
Check 'without CA_INSTALL_DIR the CLI installs under LOCALAPPDATA'

# Without CA_VERSION the tag comes from where releases/latest redirects.
$d = Join-Path $work 'latest'
if (-not (Invoke-Install $d '' "https://github.com/CloudAfrica/cloudafrica-cli/releases/tag/v$version")) {
    Die "latest install failed: $(Output)"
}
if (-not (Reports-Version "$d\ca.exe")) { Die 'latest install put the wrong version' }
Check 'without CA_VERSION the latest release is installed'
$d = Join-Path $work 'no-release'
if (Invoke-Install $d '' 'https://github.com/CloudAfrica/cloudafrica-cli/releases') { Die 'installed with no release' }
if ((Output) -notlike '*no release found*') { Die "no-release said: $(Output)" }
Check 'a repository with no release is refused'

# A tampered archive is refused before anything is installed.
Serve
Add-Content -Path (Join-Path $work "serve\$archive") -Value 'x' -NoNewline
$d = Join-Path $work 'tampered'
if (Invoke-Install $d) { Die 'installed a tampered archive' }
if ((Output) -notlike '*checksum mismatch*') { Die "tampered archive said: $(Output)" }
if (Test-Path "$d\ca.exe") { Die "tampered archive left $d\ca.exe" }
Check 'a tampered archive is refused'

# An archive missing from the checksum file is refused.
Serve
Get-Content $sums.FullName | Where-Object { $_ -notlike "* $archive" } |
    Set-Content (Join-Path $work "serve\$($sums.Name)")
$d = Join-Path $work 'unlisted'
if (Invoke-Install $d) { Die 'installed an archive missing from the checksums' }
if ((Output) -notlike '*is not listed in*') { Die "unlisted archive said: $(Output)" }
if (Test-Path "$d\ca.exe") { Die "unlisted archive left $d\ca.exe" }
Check 'an archive missing from the checksum file is refused'

Write-Host "$script:pass passed"
