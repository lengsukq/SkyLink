# SkyLink Vite dev server (started by dev-windows.ps1)
$ErrorActionPreference = "Stop"
$projectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$webDir = Join-Path $projectRoot.Path "web"
if (-not (Test-Path $webDir)) {
    Write-Host "web folder not found: $webDir" -ForegroundColor Red
    Read-Host "Press Enter to close"
    exit 1
}
Set-Location -LiteralPath $webDir
Write-Host "SkyLink web (Vite). Ctrl+C to stop. http://localhost:5173" -ForegroundColor Cyan
Write-Host ""
npm.cmd run dev
$code = $LASTEXITCODE
if ($code -ne 0) {
    Write-Host ""
    Write-Host "npm exited with code $code" -ForegroundColor Red
}
Write-Host ""
Read-Host "Press Enter to close this window"
