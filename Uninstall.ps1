$ErrorActionPreference = 'SilentlyContinue'
Get-Process -Name 'GalacticRacerHOSAS' -ErrorAction SilentlyContinue | Stop-Process -Force
$appDir = Join-Path $env:LOCALAPPDATA 'GalacticRacerHOSAS'
Remove-Item (Join-Path ([Environment]::GetFolderPath('Desktop')) 'Galactic Racer HOSAS Bridge.lnk') -Force
Remove-Item (Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Galactic Racer HOSAS Bridge.lnk') -Force
Remove-Item $appDir -Recurse -Force
Write-Host 'Galactic Racer HOSAS Bridge removed. ViGEmBus was left installed because other controller apps may use it.'
