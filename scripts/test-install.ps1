# Tests install.ps1 on Windows against the archives of `goreleaser release --snapshot` in dist/.
# Invoke-WebRequest is faked by a function that serves files from a copy of dist/, and each install
# runs the way users run it, `<script> | Invoke-Expression`, in a fresh child of the shell running
# this script, so `powershell -File` tests Windows PowerShell 5.1 and `pwsh -File` PowerShell 7.
#
# It writes the user PATH in HKCU and restores it on exit. The default-directory case writes
# %LOCALAPPDATA%\Programs\cloudafrica, so it runs only where CI is set.
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$installer = Join-Path $root 'install.ps1'
$sums = Get-Item (Join-Path $root 'dist\cloudafrica-cli_*_SHA256SUMS')
$version = $sums.Name -replace '^cloudafrica-cli_(.*)_SHA256SUMS$', '$1'
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$archive = "cloudafrica-cli_${version}_windows_${arch}.zip"
$shell = (Get-Process -Id $PID).Path

$work = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $work | Out-Null
$fake = Join-Path $work 'fake.ps1'
# The latest-release lookup answers in the shape the running edition's real response has:
# BaseResponse.ResponseUri on Windows PowerShell, BaseResponse.RequestMessage.RequestUri on 7.
Set-Content -Path $fake -Value @'
function Invoke-WebRequest([string] $Uri, [string] $OutFile, [switch] $UseBasicParsing) {
    if ($Uri.EndsWith('/releases/latest')) {
        if (-not $env:LATEST)
            { throw 'unreachable' }
        if ($PSVersionTable.PSEdition -eq 'Desktop')
            { return [pscustomobject] @{ BaseResponse = [pscustomobject] @{ ResponseUri = [uri] $env:LATEST } } }
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
    throw $Message
}

function Serve {
    $serve = Join-Path $work 'serve'
    Remove-Item $serve -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Path $serve | Out-Null
    Copy-Item (Join-Path $root "dist\$archive"), $sums.FullName $serve
}

function Output { Get-Content -Raw $out }

# Whether the output holds $Text. Whitespace is ignored: Windows PowerShell wraps a long warning at
# the console width, even into a file.
function Output-Has([string] $Text) {
    return ((Output) -replace '\s', '').Contains(($Text -replace '\s', ''))
}

# Runs install.ps1 into $Dir (none: the default) and reports whether it returned or threw. An
# install that does neither exited, which under `irm | iex` would close the user's shell.
function Invoke-Install([string] $Dir, [string] $Version = "v$version", [string] $Latest = '') {
    $env:SERVE = Join-Path $work 'serve'
    $env:CA_VERSION = $Version
    $env:CA_INSTALL_DIR = $Dir
    $env:LATEST = $Latest
    # Strict mode as a caller might have it set, since iex runs the installer in the caller's scope.
    $command = "Set-StrictMode -Version Latest; . '$fake'; try { Get-Content -Raw '$installer' | Invoke-Expression } " +
        "catch { Write-Host `$_.Exception.Message; Write-Host 'CA-TEST: threw'; exit 1 }; Write-Host 'CA-TEST: returned'"
    # Windows PowerShell turns a redirected native stderr into a terminating error under Stop.
    $ErrorActionPreference = 'Continue'
    & $shell -NoProfile -Command $command *> $out
    $ErrorActionPreference = 'Stop'
    $text = Output
    if ($text -like '*CA-TEST: returned*')
        { return $true }
    if ($text -like '*CA-TEST: threw*')
        { return $false }
    Die "install.ps1 exited instead of returning or throwing: $text"
}

function Reports-Version([string] $Bin) {
    return "$(& $Bin version)" -like "ca $version (*"
}

function Get-RawUserPath {
    return (Get-Item 'HKCU:\Environment').GetValue('Path', $null, 'DoNotExpandEnvironmentNames')
}

function User-Path-Count([string] $Dir) {
    return @((Get-RawUserPath) -split ';' | Where-Object { $_.TrimEnd('\') -eq $Dir }).Count
}

$savedPath = Get-RawUserPath
$savedKind = if ($null -ne $savedPath) { (Get-Item 'HKCU:\Environment').GetValueKind('Path') }
$defaultDir = Join-Path $env:LOCALAPPDATA 'Programs\cloudafrica'
$defaultExisted = Test-Path $defaultDir
$foreign = Join-Path $env:SystemRoot 'System32\whoami.exe'
$foreignHash = (Get-FileHash $foreign).Hash
$failed = $false
Write-Host "# $shell $($PSVersionTable.PSVersion)"

try {
    # Fresh install, then an upgrade over our own binaries. The seeded PATH holds a %VARIABLE%,
    # which must survive unexpanded in a REG_EXPAND_SZ value.
    Set-ItemProperty -Path 'HKCU:\Environment' -Name 'Path' -Value '%USERPROFILE%\ca-test-keep;C:\ca-test-keep' -Type ExpandString
    Serve
    $d = Join-Path $work 'fresh'
    if (-not (Invoke-Install $d)) { Die "fresh install failed: $(Output)" }
    if (-not (Reports-Version "$d\ca.exe")) { Die 'installed ca.exe reports the wrong version' }
    if (-not (Reports-Version "$d\cloudafrica.exe")) { Die 'installed cloudafrica.exe reports the wrong version' }
    if ((User-Path-Count $d) -ne 1) { Die "the user PATH does not hold $d once" }
    Check 'fresh install puts ca.exe and cloudafrica.exe on the user PATH'
    if ((Get-RawUserPath) -notlike '%USERPROFILE%\ca-test-keep;C:\ca-test-keep;*') { Die "the user PATH lost its entries: $(Get-RawUserPath)" }
    if ((Get-Item 'HKCU:\Environment').GetValueKind('Path') -ne 'ExpandString') { Die 'the user PATH is no longer REG_EXPAND_SZ' }
    Check 'the user PATH keeps its %VARIABLE% entries and REG_EXPAND_SZ'
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
    if (-not (Output-Has "installed only $d\cloudafrica.exe")) { Die "no notice for a foreign ca: $(Output)" }
    Check 'a foreign ca is kept and the CLI installs as cloudafrica'
    if (-not (Invoke-Install $d)) { Die "reinstall beside a foreign ca failed: $(Output)" }
    Check 'reinstall beside a foreign ca replaces our cloudafrica'

    # Both names taken by other programs: refused, both kept.
    $d = Join-Path $work 'foreign-both'
    New-Item -ItemType Directory -Path $d | Out-Null
    Copy-Item $foreign "$d\ca.exe"
    Set-Content "$d\cloudafrica.exe" 'other'
    if (Invoke-Install $d) { Die 'installed over foreign ca and cloudafrica' }
    if (-not (Output-Has 'are other programs')) { Die "both-foreign refusal said: $(Output)" }
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
    if (-not (Output-Has "Left $d\cloudafrica.exe alone")) { Die "no warning for a foreign cloudafrica: $(Output)" }
    Check 'a foreign cloudafrica is kept with a warning'

    # Without CA_INSTALL_DIR the CLI goes under LOCALAPPDATA.
    # The PATH already names it through %LOCALAPPDATA%, which must not gain a duplicate.
    if ($env:CI) {
        $seeded = 'C:\ca-test-keep;%LOCALAPPDATA%\Programs\cloudafrica'
        Set-ItemProperty -Path 'HKCU:\Environment' -Name 'Path' -Value $seeded -Type ExpandString
        if (-not (Invoke-Install '')) { Die "default-directory install failed: $(Output)" }
        if (-not (Reports-Version "$defaultDir\ca.exe")) { Die "the default install did not put ca.exe in $defaultDir" }
        if ((Get-RawUserPath) -ne $seeded) { Die "a PATH entry naming the directory through a variable was duplicated: $(Get-RawUserPath)" }
        Check 'without CA_INSTALL_DIR the CLI installs under LOCALAPPDATA, already on PATH by variable'
    } else {
        Write-Host "skip - the default-directory case writes $defaultDir; set CI to run it"
    }

    # Without CA_VERSION the tag comes from where releases/latest redirects.
    $d = Join-Path $work 'latest'
    if (-not (Invoke-Install $d '' "https://github.com/CloudAfrica/cloudafrica-cli/releases/tag/v$version")) {
        Die "latest install failed: $(Output)"
    }
    if (-not (Reports-Version "$d\ca.exe")) { Die 'latest install put the wrong version' }
    Check 'without CA_VERSION the latest release is installed'
    $d = Join-Path $work 'no-release'
    if (Invoke-Install $d '' 'https://github.com/CloudAfrica/cloudafrica-cli/releases') { Die 'installed with no release' }
    if (-not (Output-Has 'no release found')) { Die "no-release said: $(Output)" }
    Check 'a repository with no release is refused'

    # A tampered archive is refused before anything is installed.
    Serve
    Add-Content -Path (Join-Path $work "serve\$archive") -Value 'x' -NoNewline
    $d = Join-Path $work 'tampered'
    if (Invoke-Install $d) { Die 'installed a tampered archive' }
    if (-not (Output-Has 'checksum mismatch')) { Die "tampered archive said: $(Output)" }
    if (Test-Path "$d\ca.exe") { Die "tampered archive left $d\ca.exe" }
    Check 'a tampered archive is refused'

    # An archive missing from the checksum file is refused.
    Serve
    Get-Content $sums.FullName | Where-Object { $_ -notlike "* $archive" } |
        Set-Content (Join-Path $work "serve\$($sums.Name)")
    $d = Join-Path $work 'unlisted'
    if (Invoke-Install $d) { Die 'installed an archive missing from the checksums' }
    if (-not (Output-Has 'is not listed in')) { Die "unlisted archive said: $(Output)" }
    if (Test-Path "$d\ca.exe") { Die "unlisted archive left $d\ca.exe" }
    Check 'an archive missing from the checksum file is refused'

    # An archive whose ca.exe does not run is refused before it replaces a working install.
    Serve
    $broken = Join-Path $work 'broken'
    New-Item -ItemType Directory -Path $broken | Out-Null
    Copy-Item $foreign "$broken\ca.exe"
    $zip = Join-Path $work "serve\$archive"
    Remove-Item $zip
    Compress-Archive -Path "$broken\ca.exe" -DestinationPath $zip
    $hash = (Get-FileHash $zip -Algorithm SHA256).Hash.ToLower()
    Set-Content (Join-Path $work "serve\$($sums.Name)") "$hash  $archive"
    $d = Join-Path $work 'fresh'
    $before = (Get-FileHash "$d\ca.exe").Hash
    if (Invoke-Install $d) { Die 'reported success for a ca.exe that does not run' }
    if (-not (Output-Has 'does not run; nothing was installed')) { Die "a broken ca.exe said: $(Output)" }
    if ((Get-FileHash "$d\ca.exe").Hash -ne $before -or -not (Reports-Version "$d\cloudafrica.exe")) {
        Die 'a broken ca.exe replaced a working install'
    }
    Check 'a downloaded ca.exe that does not run is refused and the install kept'

    Write-Host "$script:pass passed"
} catch {
    Write-Host "FAIL - $($_.Exception.Message)"
    $failed = $true
} finally {
    if ($null -eq $savedPath) {
        Remove-ItemProperty -Path 'HKCU:\Environment' -Name 'Path' -ErrorAction SilentlyContinue
    } else {
        Set-ItemProperty -Path 'HKCU:\Environment' -Name 'Path' -Value $savedPath -Type $savedKind
    }
    # Broadcast the restored PATH, as install.ps1 broadcast the test's.
    [Environment]::SetEnvironmentVariable('CA_INSTALL_PATH_REFRESH', '1', 'User')
    [Environment]::SetEnvironmentVariable('CA_INSTALL_PATH_REFRESH', $null, 'User')
    if ($env:CI -and -not $defaultExisted)
        { Remove-Item $defaultDir -Recurse -Force -ErrorAction SilentlyContinue }
    Remove-Item $work -Recurse -Force -ErrorAction SilentlyContinue
}
if ($failed)
    { exit 1 }
