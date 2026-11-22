# Install the CloudAfrica CLI (ca) on Windows.
#
#   irm https://github.com/CloudAfrica/cloudafrica-cli/releases/latest/download/install.ps1 | iex
#
# CA_VERSION     a release tag such as v0.1.0 (default: the latest release)
# CA_INSTALL_DIR where to put ca.exe and cloudafrica.exe (default: %LOCALAPPDATA%\Programs\cloudafrica),
#                added to the user PATH when it is not on it
#
# Errors throw rather than exit: under `irm | iex` an exit would close the caller's shell.

# Test-CloudAfricaCli reports whether $Path is this CLI: `ca version` prints "ca <version> (".
# The empty pipeline input is its stdin, so a program that reads one cannot hang the install.
function Test-CloudAfricaCli([string] $Path) {
    try {
        $first = $null | & $Path version 2>$null | Select-Object -First 1
        return "$first" -like 'ca * (*'
    } catch {
        return $false
    }
}

function Add-CloudAfricaUserPath([string] $Dir) {
    # Read and write the raw registry value: [Environment]::SetEnvironmentVariable would store it
    # as REG_SZ with every %VARIABLE% in it expanded.
    $key = Get-Item -Path 'HKCU:\Environment'
    $path = [string] $key.GetValue('Path', '', 'DoNotExpandEnvironmentNames')
    $entries = $path -split ';' | Where-Object { $_ } | ForEach-Object { $_.TrimEnd('\') }
    if ($entries -contains $Dir.TrimEnd('\'))
        { return $false }
    $value = if ($path -and -not $path.EndsWith(';')) { "$path;$Dir" } else { "$path$Dir" }
    Set-ItemProperty -Path 'HKCU:\Environment' -Name 'Path' -Value $value -Type ExpandString
    # Setting and clearing a variable through .NET broadcasts WM_SETTINGCHANGE, so a terminal
    # opened from now on sees the new PATH without a sign-out.
    [Environment]::SetEnvironmentVariable('CA_INSTALL_PATH_REFRESH', '1', 'User')
    [Environment]::SetEnvironmentVariable('CA_INSTALL_PATH_REFRESH', $null, 'User')
    return $true
}

# The body is one function called on the last line, so a download cut short by `irm | iex`
# runs nothing.
function Install-CloudAfricaCli {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'
    $repo = 'https://github.com/CloudAfrica/cloudafrica-cli'

    if ($PSVersionTable.PSEdition -eq 'Core' -and -not $IsWindows)
        { throw 'install.ps1: this installs on Windows; on Linux or macOS use install.sh' }

    $machine = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    $arch = switch ($machine) {
        'AMD64' { 'amd64' }
        'ARM64' { 'arm64' }
        default { throw "install.ps1: unsupported architecture $machine" }
    }

    # Windows PowerShell 5.1 on an older .NET offers only TLS 1.0, which GitHub refuses.
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor
        [Net.SecurityProtocolType]::Tls12

    if ($env:CA_VERSION) {
        $tag = $env:CA_VERSION
    } else {
        try {
            $response = Invoke-WebRequest -Uri "$repo/releases/latest" -UseBasicParsing
        } catch {
            throw "install.ps1: cannot reach ${repo}: $($_.Exception.Message)"
        }
        # Where releases/latest redirected to: ResponseUri on Windows PowerShell, RequestUri on PowerShell 7.
        $latest = $response.BaseResponse.ResponseUri
        if (-not $latest)
            { $latest = $response.BaseResponse.RequestMessage.RequestUri }
        $tag = ("$latest" -split '/')[-1]
    }
    if ($tag -notmatch '^v[0-9]')
        { throw "install.ps1: no release found (got '$tag')" }
    $version = $tag.Substring(1)

    $dir = if ($env:CA_INSTALL_DIR) { $env:CA_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\cloudafrica' }
    # Relative to the PowerShell location, which [IO.Path]::GetFullPath would not honour.
    $dir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($dir)
    New-Item -ItemType Directory -Force -Path $dir | Out-Null

    $tmp = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
        $archive = "cloudafrica-cli_${version}_windows_${arch}.zip"
        $sums = "cloudafrica-cli_${version}_SHA256SUMS"
        Write-Host "Downloading ca $tag for windows/$arch"
        foreach ($file in $archive, $sums) {
            try {
                Invoke-WebRequest -Uri "$repo/releases/download/$tag/$file" -OutFile (Join-Path $tmp $file) -UseBasicParsing
            } catch {
                throw "install.ps1: download failed: ${file}: $($_.Exception.Message)"
            }
        }

        $want = Get-Content -Path (Join-Path $tmp $sums) |
            ForEach-Object { $f = -split $_; if ($f.Count -eq 2 -and $f[1] -eq $archive) { $f[0] } } |
            Select-Object -First 1
        if (-not $want)
            { throw "install.ps1: $archive is not listed in $sums" }
        $got = (Get-FileHash -Path (Join-Path $tmp $archive) -Algorithm SHA256).Hash
        if ($got -ne $want)
            { throw "install.ps1: checksum mismatch for $archive" }

        Expand-Archive -Path (Join-Path $tmp $archive) -DestinationPath (Join-Path $tmp 'x')
        $new = Join-Path $tmp 'x\ca.exe'
        $ca = Join-Path $dir 'ca.exe'
        $alias = Join-Path $dir 'cloudafrica.exe'
        if ((Test-Path -LiteralPath $ca) -and -not (Test-CloudAfricaCli $ca)) {
            # The name clash the cloudafrica alias exists for: keep the other ca.
            if ((Test-Path -LiteralPath $alias) -and -not (Test-CloudAfricaCli $alias))
                { throw "install.ps1: $ca and $alias are other programs; set CA_INSTALL_DIR to install elsewhere" }
            Copy-Item -LiteralPath $new -Destination $alias -Force
            Write-Warning "$ca is another program, so this installed only $alias"
            $bin = $alias
        } else {
            Copy-Item -LiteralPath $new -Destination $ca -Force
            if ((Test-Path -LiteralPath $alias) -and -not (Test-CloudAfricaCli $alias)) {
                Write-Warning "Left $alias alone: it is another program"
            } else {
                Copy-Item -LiteralPath $new -Destination $alias -Force
            }
            $bin = $ca
        }
    } finally {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }

    Write-Host "Installed $bin"
    if (Add-CloudAfricaUserPath $dir) {
        Write-Host "Added $dir to your user PATH; open a new terminal to run $([IO.Path]::GetFileNameWithoutExtension($bin))."
    }
    if (($env:Path -split ';' | ForEach-Object { $_.TrimEnd('\') }) -notcontains $dir.TrimEnd('\'))
        { $env:Path = "$env:Path;$dir" }
    & $bin version
    if ($LASTEXITCODE -ne 0)
        { throw "install.ps1: $bin version exited $LASTEXITCODE; the installed binary does not run" }
}

Install-CloudAfricaCli
