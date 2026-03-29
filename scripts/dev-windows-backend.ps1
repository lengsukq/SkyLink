# SkyLink backend with Air (started by dev-windows.ps1)
$ErrorActionPreference = "Stop"
$projectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location -LiteralPath $projectRoot.Path
Write-Host "SkyLink backend (Air). Ctrl+C to stop. Admin :19080  Proxy :18080" -ForegroundColor Cyan
Write-Host ""
go run github.com/air-verse/air@latest
$code = $LASTEXITCODE
if ($code -ne 0) {
    Write-Host ""
    Write-Host "Backend exited with code $code" -ForegroundColor Red
}
Write-Host ""
Read-Host "Press Enter to close this window"
