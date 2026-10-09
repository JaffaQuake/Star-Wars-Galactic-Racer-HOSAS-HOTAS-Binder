$ErrorActionPreference = 'SilentlyContinue'
Get-Process -Name 'GalacticRacerHOSAS' -ErrorAction SilentlyContinue | Stop-Process -Force

$localAppData = [Environment]::GetFolderPath('LocalApplicationData')
if ([string]::IsNullOrWhiteSpace($localAppData)) { $localAppData = $env:LOCALAPPDATA }
if (-not [string]::IsNullOrWhiteSpace($localAppData)) {
    $appDir = Join-Path $localAppData 'GalacticRacerHOSAS'
    Remove-Item -LiteralPath $appDir -Recurse -Force
}

$desktop = [Environment]::GetFolderPath('Desktop')
$programs = [Environment]::GetFolderPath('Programs')
if (-not [string]::IsNullOrWhiteSpace($desktop)) {
    Remove-Item -LiteralPath (Join-Path $desktop 'Galactic Racer HOSAS Bridge.lnk') -Force
}
if (-not [string]::IsNullOrWhiteSpace($programs)) {
    Remove-Item -LiteralPath (Join-Path $programs 'Galactic Racer HOSAS Bridge.lnk') -Force
}

Write-Host 'Galactic Racer HOSAS Bridge removed. ViGEmBus was left installed because other controller apps may use it.'
