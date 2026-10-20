# Run only when requested by Install.ps1. This is the one elevation boundary:
# the main application and its user settings remain in the standard user's context.
$ErrorActionPreference = 'Stop'
try {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Error 'ViGEmBus setup requires administrator approval.'
        exit 10
    }
    $winget = Get-Command 'winget.exe' -ErrorAction SilentlyContinue
    if (-not $winget) {
        Write-Error 'winget is unavailable in the elevated session.'
        exit 11
    }
    Write-Host 'Installing/repairing ViGEmBus with elevated permissions...' -ForegroundColor Cyan
    & $winget.Source install --id ViGEm.ViGEmBus -e --silent --accept-package-agreements --accept-source-agreements
    $result = $LASTEXITCODE
    if ($result -ne 0) {
        Write-Error "ViGEmBus winget installation did not complete (exit code $result)."
        exit 12
    }
    Write-Host 'ViGEmBus package setup finished. The main installer will now test the actual bus connection.'
    exit 0
} catch {
    Write-Error "ViGEmBus setup failed: $($_.Exception.Message)"
    exit 13
}
