$ErrorActionPreference = 'Stop'
Write-Host "Galactic Racer HOSAS / HOTAS Bridge v1.9 installer" -ForegroundColor Cyan

$sourceDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$appDir = Join-Path $env:LOCALAPPDATA 'GalacticRacerHOSAS'
New-Item -ItemType Directory -Force -Path $appDir | Out-Null

# Close the previous build before replacing the executable. The configuration file is not removed.
Get-Process -Name 'GalacticRacerHOSAS' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 250
Copy-Item (Join-Path $sourceDir 'GalacticRacerHOSAS.exe') (Join-Path $appDir 'GalacticRacerHOSAS.exe') -Force
if (Test-Path (Join-Path $sourceDir 'GalacticRacerHOSAS.ico')) {
    Copy-Item (Join-Path $sourceDir 'GalacticRacerHOSAS.ico') (Join-Path $appDir 'GalacticRacerHOSAS.ico') -Force
}

# Keep license/help documents beside the installed app.
foreach ($doc in @('LICENSE.txt','THIRD_PARTY_NOTICES.txt','QUICKSTART.txt','README.txt')) {
    $docPath = Join-Path $sourceDir $doc
    if (Test-Path $docPath) { Copy-Item $docPath (Join-Path $appDir $doc) -Force }
}

# ViGEmClient.dll is MIT-licensed. Fetch the copy shipped in the vgamepad 0.1.0 source package.
$dllPath = Join-Path $appDir 'ViGEmClient.dll'
if (-not (Test-Path $dllPath)) {
    Write-Host 'Downloading ViGEmClient.dll dependency...'
    $meta = Invoke-RestMethod 'https://pypi.org/pypi/vgamepad/0.1.0/json'
    $sdist = $meta.urls | Where-Object { $_.packagetype -eq 'sdist' } | Select-Object -First 1
    if (-not $sdist) { throw 'Could not find the vgamepad source package on PyPI.' }
    $tmp = Join-Path $env:TEMP ('gr-hosas-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmp | Out-Null
    $tgz = Join-Path $tmp 'vgamepad.tar.gz'
    Invoke-WebRequest -UseBasicParsing $sdist.url -OutFile $tgz
    $hash = (Get-FileHash $tgz -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($hash -ne '57f6bd01aec0c172947517fb782d150ef9b285f7f4d524c317374fa5c24a89de') {
        throw "vgamepad package checksum mismatch. Got $hash"
    }
    tar -xzf $tgz -C $tmp
    $found = Get-ChildItem -Path $tmp -Recurse -Filter 'ViGEmClient.dll' | Where-Object { $_.FullName -match '\x64' } | Select-Object -First 1 # Removed trailing '\' from '\x64\' Which was causing installation of ViGEmClient to fail.
    if (-not $found) { throw 'Could not extract the x64 ViGEmClient.dll.' }
    Copy-Item $found.FullName $dllPath -Force
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

# Install the virtual gamepad driver if needed.
$vigemService = Get-Service -Name 'ViGEmBus' -ErrorAction SilentlyContinue
if (-not $vigemService) {
    Write-Host 'Installing ViGEmBus virtual-controller driver (Windows may request permission)...'
    if (Get-Command winget -ErrorAction SilentlyContinue) {
        winget install --id ViGEm.ViGEmBus -e --accept-package-agreements --accept-source-agreements
    } else {
        Write-Warning 'winget was not found. Install ViGEmBus 1.22.0 manually from the official ViGEmBus GitHub releases page, then run the app.'
    }
}

# Desktop + Start Menu shortcuts.
$ws = New-Object -ComObject WScript.Shell
$desktop = [Environment]::GetFolderPath('Desktop')
$shortcut = $ws.CreateShortcut((Join-Path $desktop 'Galactic Racer HOSAS Bridge.lnk'))
$shortcut.TargetPath = (Join-Path $appDir 'GalacticRacerHOSAS.exe')
$shortcut.WorkingDirectory = $appDir
$iconPath = Join-Path $appDir 'GalacticRacerHOSAS.ico'
if (Test-Path $iconPath) { $shortcut.IconLocation = "$iconPath,0" }
$shortcut.Save()

$startDir = Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs'
$shortcut2 = $ws.CreateShortcut((Join-Path $startDir 'Galactic Racer HOSAS Bridge.lnk'))
$shortcut2.TargetPath = (Join-Path $appDir 'GalacticRacerHOSAS.exe')
$shortcut2.WorkingDirectory = $appDir
if (Test-Path $iconPath) { $shortcut2.IconLocation = "$iconPath,0" }
$shortcut2.Save()

Write-Host ''
Write-Host 'Installed successfully. Existing controller settings were preserved.' -ForegroundColor Green
Write-Host "App folder: $appDir"
Start-Process (Join-Path $appDir 'GalacticRacerHOSAS.exe')
