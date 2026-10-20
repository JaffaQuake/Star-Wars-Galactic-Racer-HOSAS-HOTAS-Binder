$ErrorActionPreference = 'Stop'
Write-Host "Galactic Racer HOSAS / HOTAS Bridge v2.2.3 official installer" -ForegroundColor Cyan

function Get-LocalAppDataPath {
    $p = [Environment]::GetFolderPath('LocalApplicationData')
    if ([string]::IsNullOrWhiteSpace($p)) { $p = $env:LOCALAPPDATA }
    if ([string]::IsNullOrWhiteSpace($p)) { throw 'Windows did not provide a Local AppData folder.' }
    return $p
}

function Test-X64PEFile([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $false }
    try {
        $fs = [System.IO.File]::Open($Path, 'Open', 'Read', 'ReadWrite')
        try {
            if ($fs.Length -lt 64) { return $false }
            $br = New-Object System.IO.BinaryReader($fs)
            if ($br.ReadUInt16() -ne 0x5A4D) { return $false } # MZ
            $fs.Position = 0x3C
            $peOffset = $br.ReadInt32()
            if ($peOffset -lt 0 -or ($peOffset + 6) -gt $fs.Length) { return $false }
            $fs.Position = $peOffset
            if ($br.ReadUInt32() -ne 0x00004550) { return $false } # PE\0\0
            $machine = $br.ReadUInt16()
            return $machine -eq 0x8664 # AMD64
        } finally {
            $fs.Dispose()
        }
    } catch {
        return $false
    }
}

function Find-X64ViGEmClient([string]$Root) {
    if (-not (Test-Path -LiteralPath $Root)) { return $null }
    $candidates = @(Get-ChildItem -LiteralPath $Root -Recurse -File -Filter 'ViGEmClient.dll' -ErrorAction SilentlyContinue)
    foreach ($candidate in $candidates) {
        if (Test-X64PEFile $candidate.FullName) { return $candidate.FullName }
    }
    return $null
}

function Test-ViGEmBusInstalled {
    try {
        $svc = Get-Service -ErrorAction SilentlyContinue | Where-Object {
            $_.Name -like '*ViGEm*' -or $_.DisplayName -like '*ViGEm*'
        } | Select-Object -First 1
        if ($svc) { return $true }
    } catch {}

    try {
        $drv = Get-CimInstance Win32_SystemDriver -ErrorAction SilentlyContinue | Where-Object {
            $_.Name -like '*ViGEm*' -or $_.DisplayName -like '*ViGEm*'
        } | Select-Object -First 1
        if ($drv) { return $true }
    } catch {}

    try {
        if (Get-Command Get-PnpDevice -ErrorAction SilentlyContinue) {
            $pnp = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object {
                $_.FriendlyName -like '*ViGEm*' -or $_.InstanceId -like '*ViGEm*'
            } | Select-Object -First 1
            if ($pnp) { return $true }
        }
    } catch {}

    return $false
}

function New-AppShortcut([string]$Directory, [string]$Name, [string]$TargetPath, [string]$WorkingDirectory, [string]$IconPath) {
    if ([string]::IsNullOrWhiteSpace($Directory)) { return }
    try {
        New-Item -ItemType Directory -Force -Path $Directory | Out-Null
        $ws = New-Object -ComObject WScript.Shell
        $shortcut = $ws.CreateShortcut((Join-Path $Directory $Name))
        $shortcut.TargetPath = $TargetPath
        $shortcut.WorkingDirectory = $WorkingDirectory
        if (Test-Path -LiteralPath $IconPath) { $shortcut.IconLocation = "$IconPath,0" }
        $shortcut.Save()
    } catch {
        Write-Warning "Could not create shortcut in '$Directory': $($_.Exception.Message)"
    }
}

$sourceDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$sourceExe = Join-Path $sourceDir 'GalacticRacerHOSAS.exe'
if (-not (Test-Path -LiteralPath $sourceExe -PathType Leaf)) {
    throw "GalacticRacerHOSAS.exe was not found beside Install.ps1. Extract the complete ZIP before installing."
}
$sourceHelper = Join-Path $sourceDir 'GalacticRacerWinMMHelper.exe'
if (-not (Test-Path -LiteralPath $sourceHelper -PathType Leaf)) {
    throw "GalacticRacerWinMMHelper.exe was not found beside Install.ps1. Extract the complete ZIP before installing."
}
if (-not (Test-X64PEFile $sourceExe)) {
    throw 'GalacticRacerHOSAS.exe is missing, corrupt, or not an x64 Windows executable.'
}
if (-not (Test-X64PEFile $sourceHelper)) {
    throw 'GalacticRacerWinMMHelper.exe is missing, corrupt, or not an x64 Windows executable.'
}
if (-not [Environment]::Is64BitOperatingSystem) {
    throw 'This release is a 64-bit Windows application and requires 64-bit Windows.'
}

$localAppData = Get-LocalAppDataPath
$appDir = Join-Path $localAppData 'GalacticRacerHOSAS'
New-Item -ItemType Directory -Force -Path $appDir | Out-Null

Write-Host "Install folder: $appDir"
Write-Host 'Checking packaged application...' -ForegroundColor DarkCyan

# Close the previous build before replacing files. The configuration file is deliberately preserved.
Get-Process -Name 'GalacticRacerHOSAS','GalacticRacerWinMMHelper' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 250
Copy-Item -LiteralPath $sourceExe -Destination (Join-Path $appDir 'GalacticRacerHOSAS.exe') -Force
Copy-Item -LiteralPath $sourceHelper -Destination (Join-Path $appDir 'GalacticRacerWinMMHelper.exe') -Force

$sourceIcon = Join-Path $sourceDir 'GalacticRacerHOSAS.ico'
if (Test-Path -LiteralPath $sourceIcon) {
    Copy-Item -LiteralPath $sourceIcon -Destination (Join-Path $appDir 'GalacticRacerHOSAS.ico') -Force
}

# Keep runtime documentation inside the installation. Release audits, hashes and
# per-version notes belong in the downloadable archive/build workspace, not AppData.
foreach ($doc in @('LICENSE.txt','THIRD_PARTY_NOTICES.txt','QUICKSTART.txt','README.txt','WINMM_COMPATIBILITY.txt','CHANGELOG.md')) {
    $docPath = Join-Path $sourceDir $doc
    if (Test-Path -LiteralPath $docPath) {
        Copy-Item -LiteralPath $docPath -Destination (Join-Path $appDir $doc) -Force
    }
}


# ViGEmClient.dll is MIT-licensed. Prefer a valid packaged/local copy. Otherwise fetch the
# pinned vgamepad 0.1.0 source archive and select a DLL by PE architecture, not by folder name.
$dllPath = Join-Path $appDir 'ViGEmClient.dll'
$dllReady = Test-X64PEFile $dllPath
if (-not $dllReady) {
    $packagedDll = Join-Path $sourceDir 'ViGEmClient.dll'
    if (Test-X64PEFile $packagedDll) {
        Write-Host 'Using packaged x64 ViGEmClient.dll.'
        Copy-Item -LiteralPath $packagedDll -Destination $dllPath -Force
        $dllReady = $true
    }
}

if (-not $dllReady) {
    Write-Host 'Downloading ViGEmClient.dll dependency...'
    if (-not (Get-Command tar.exe -ErrorAction SilentlyContinue)) {
        throw 'Windows tar.exe was not found. Install the current Windows system components or place an x64 ViGEmClient.dll beside Install.ps1 and run the installer again.'
    }

    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    } catch {}

    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ('gr-hosas-' + [guid]::NewGuid().ToString('N'))
    try {
        New-Item -ItemType Directory -Path $tmp | Out-Null
        $meta = Invoke-RestMethod 'https://pypi.org/pypi/vgamepad/0.1.0/json'
        $sdist = $meta.urls | Where-Object { $_.packagetype -eq 'sdist' } | Select-Object -First 1
        if (-not $sdist) { throw 'Could not find the vgamepad 0.1.0 source package on PyPI.' }

        $tgz = Join-Path $tmp 'vgamepad.tar.gz'
        Invoke-WebRequest -UseBasicParsing $sdist.url -OutFile $tgz
        $hash = (Get-FileHash -LiteralPath $tgz -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($hash -ne '57f6bd01aec0c172947517fb782d150ef9b285f7f4d524c317374fa5c24a89de') {
            throw "vgamepad package checksum mismatch. Got $hash"
        }

        & tar.exe -xzf $tgz -C $tmp
        if ($LASTEXITCODE -ne 0) { throw "tar.exe failed to extract the dependency archive (exit code $LASTEXITCODE)." }

        $found = Find-X64ViGEmClient $tmp
        if (-not $found) {
            throw 'ViGEmClient.dll files were found/extracted, but no valid x64 DLL could be identified.'
        }
        Copy-Item -LiteralPath $found -Destination $dllPath -Force
        $dllReady = Test-X64PEFile $dllPath
        if (-not $dllReady) { throw 'The copied ViGEmClient.dll did not validate as an x64 PE file.' }
    } finally {
        if (Test-Path -LiteralPath $tmp) {
            Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Write-Host 'ViGEmClient.dll: x64 OK' -ForegroundColor Green

# Hardware readiness must be tested through ViGEmClient, not inferred from a
# service name alone. The shipped executable exposes a no-UI, no-controller
# preflight that connects and disconnects safely using the installed client DLL.
$installedExe = Join-Path $appDir 'GalacticRacerHOSAS.exe'
function Test-ViGEmBusConnectivity([string]$Exe) {
    try {
        $p = Start-Process -FilePath $Exe -ArgumentList '--check-vigem' -WorkingDirectory $appDir -PassThru -Wait -ErrorAction Stop
        return ($p.ExitCode -eq 0)
    } catch { return $false }
}

$driverReady = Test-ViGEmBusConnectivity $installedExe
$driverPackageExit = $null
if (-not $driverReady) {
    if (Test-ViGEmBusInstalled) {
        Write-Warning 'A ViGEmBus service or device was found, but ViGEmClient could not connect to it.'
    } else {
        Write-Warning 'No working ViGEmBus installation was detected.'
    }
    $driverScript = Join-Path $sourceDir 'Install-Driver.ps1'
    if (-not (Test-Path -LiteralPath $driverScript -PathType Leaf)) {
        Write-Warning 'Install-Driver.ps1 is missing. Re-extract the complete v2.2.3 archive.'
    } else {
        Write-Host 'Requesting administrator approval for ViGEmBus ONLY (the application itself stays unelevated)...' -ForegroundColor Cyan
        try {
            $args = '-NoProfile -ExecutionPolicy Bypass -File "' + $driverScript + '"'
            $elevated = Start-Process -FilePath 'powershell.exe' -ArgumentList $args -Verb RunAs -PassThru -Wait -ErrorAction Stop
            $driverPackageExit = $elevated.ExitCode
            if ($driverPackageExit -ne 0) {
                Write-Warning "Driver setup did not complete (status $driverPackageExit)."
            }
        } catch {
            Write-Warning "Driver elevation or installation was declined or failed: $($_.Exception.Message)"
        }
    }
    # Driver registration can lag behind installer completion; probe the actual
    # ViGEmClient connection after each retry rather than checking only names.
    for ($i = 0; $i -lt 10 -and -not $driverReady; $i++) {
        Start-Sleep -Seconds 1
        $driverReady = Test-ViGEmBusConnectivity $installedExe
    }
}
if ($driverReady) {
    Write-Host 'ViGEmBus: READY - ViGEmClient connected successfully.' -ForegroundColor Green
} else {
    if ($driverPackageExit -eq 0) {
        Write-Warning 'Driver installation returned success, but the bus connection is not ready yet. Restart Windows, then try again.'
    } else {
        Write-Warning 'ViGEmBus is not ready. Xbox output will not work until the driver is installed/repaired and connectivity succeeds.'
    }
    Write-Host 'Official manual driver installer: https://github.com/nefarius/ViGEmBus/releases/tag/v1.22.0' -ForegroundColor Yellow
}

# Desktop + Start Menu shortcuts. Resolve Windows special folders rather than constructing paths.
$installedExe = Join-Path $appDir 'GalacticRacerHOSAS.exe'
$iconPath = Join-Path $appDir 'GalacticRacerHOSAS.ico'
$desktop = [Environment]::GetFolderPath('Desktop')
$programs = [Environment]::GetFolderPath('Programs')
New-AppShortcut $desktop 'Galactic Racer HOSAS Bridge.lnk' $installedExe $appDir $iconPath
New-AppShortcut $programs 'Galactic Racer HOSAS Bridge.lnk' $installedExe $appDir $iconPath


# Explicitly limited, nonrecursive cleanup of obsolete generated installer artifacts.
# DO NOT use a general wipe of AppData: hosas_config.json, user files, logs,
# device assignments and unknown files must survive upgrades. Do not follow
# symlinks/junctions or recurse into user-created folders.
$legacyReleasePattern = '^(?:BUILD_VALIDATION|INSTALLER_AUDIT|SHA256|RELEASE_NOTES)_v[0-9][0-9A-Za-z.\-]*\.txt$'
Get-ChildItem -LiteralPath $appDir -File -Force -ErrorAction SilentlyContinue | Where-Object {
    $_.Name -match $legacyReleasePattern -and -not ($_.Attributes -band [IO.FileAttributes]::ReparsePoint)
} | ForEach-Object {
    try {
        Remove-Item -LiteralPath $_.FullName -Force -ErrorAction Stop
        Write-Host "Removed obsolete generated release artifact: $($_.Name)"
    } catch {
        Write-Warning "Could not clean obsolete file $($_.Name); skipped: $($_.Exception.Message)"
    }
}

Write-Host ''
Write-Host 'GalacticRacerHOSAS application installed. Existing controller settings were preserved.' -ForegroundColor Green
Write-Host "App folder: $appDir"
if (-not $driverReady) {
    Write-Host 'Xbox controller NOT READY. Restart Windows, or install/repair ViGEmBus 1.22.0 manually using the official link above.' -ForegroundColor Yellow
}
Start-Process -FilePath $installedExe -WorkingDirectory $appDir
