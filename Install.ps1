$ErrorActionPreference = 'Stop'
Write-Host "Galactic Racer HOSAS / HOTAS Bridge v1.91 installer" -ForegroundColor Cyan

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
if (-not [Environment]::Is64BitOperatingSystem) {
    throw 'This release is a 64-bit Windows application and requires 64-bit Windows.'
}

$localAppData = Get-LocalAppDataPath
$appDir = Join-Path $localAppData 'GalacticRacerHOSAS'
New-Item -ItemType Directory -Force -Path $appDir | Out-Null

Write-Host "Install folder: $appDir"
Write-Host 'Checking packaged application...' -ForegroundColor DarkCyan

# Close the previous build before replacing files. The configuration file is deliberately preserved.
Get-Process -Name 'GalacticRacerHOSAS' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 250
Copy-Item -LiteralPath $sourceExe -Destination (Join-Path $appDir 'GalacticRacerHOSAS.exe') -Force

$sourceIcon = Join-Path $sourceDir 'GalacticRacerHOSAS.ico'
if (Test-Path -LiteralPath $sourceIcon) {
    Copy-Item -LiteralPath $sourceIcon -Destination (Join-Path $appDir 'GalacticRacerHOSAS.ico') -Force
}

foreach ($doc in @('LICENSE.txt','THIRD_PARTY_NOTICES.txt','QUICKSTART.txt','README.txt','INSTALLER_AUDIT_v1.91.txt')) {
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

# Install the virtual gamepad driver if it cannot be detected. Detection intentionally checks
# services, system drivers, and PnP devices instead of assuming a single exact service name.
$driverReady = Test-ViGEmBusInstalled
if (-not $driverReady) {
    Write-Host 'ViGEmBus was not detected. Attempting installation (Windows may request permission)...'
    $winget = Get-Command winget.exe -ErrorAction SilentlyContinue
    if ($winget) {
        $wingetPath = $winget.Source
        & $wingetPath install --id ViGEm.ViGEmBus -e --accept-package-agreements --accept-source-agreements
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "winget returned exit code $LASTEXITCODE while installing ViGEmBus."
        }
        Start-Sleep -Milliseconds 500
        $driverReady = Test-ViGEmBusInstalled
    } else {
        Write-Warning 'winget was not found. ViGEmBus must be installed manually before virtual Xbox output can work.'
    }
}

if ($driverReady) {
    Write-Host 'ViGEmBus driver: detected' -ForegroundColor Green
} else {
    Write-Warning 'ViGEmBus could not be confirmed. The application will install, but virtual Xbox output may not work until the driver is installed.'
}

# Desktop + Start Menu shortcuts. Resolve Windows special folders rather than constructing paths.
$installedExe = Join-Path $appDir 'GalacticRacerHOSAS.exe'
$iconPath = Join-Path $appDir 'GalacticRacerHOSAS.ico'
$desktop = [Environment]::GetFolderPath('Desktop')
$programs = [Environment]::GetFolderPath('Programs')
New-AppShortcut $desktop 'Galactic Racer HOSAS Bridge.lnk' $installedExe $appDir $iconPath
New-AppShortcut $programs 'Galactic Racer HOSAS Bridge.lnk' $installedExe $appDir $iconPath

Write-Host ''
Write-Host 'Installed successfully. Existing controller settings were preserved.' -ForegroundColor Green
Write-Host "App folder: $appDir"
if (-not $driverReady) {
    Write-Host 'NOTE: ViGEmBus was not confirmed, so Start Mapping may report an Xbox-output error until the driver is installed.' -ForegroundColor Yellow
}
Start-Process -FilePath $installedExe -WorkingDirectory $appDir
