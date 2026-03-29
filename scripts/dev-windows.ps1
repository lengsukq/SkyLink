# SkyLink Windows dev: opens two windows (Air + Vite). Run: dev-windows.cmd
# All messages are ASCII-only so default system encoding cannot break parsing.

param(
    [switch] $SkipNpmInstall
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Resolve-Path (Join-Path $scriptDir "..")
$webDir = Join-Path $projectRoot.Path "web"
$backendPs1 = Join-Path $scriptDir "dev-windows-backend.ps1"
$webPs1 = Join-Path $scriptDir "dev-windows-web.ps1"

function Test-Cmd($name) {
    return [bool](Get-Command $name -ErrorAction SilentlyContinue)
}

if (-not (Test-Cmd "go")) {
    Write-Host "go not found. Install Go and add it to PATH." -ForegroundColor Red
    exit 1
}
if (-not (Test-Cmd "npm")) {
    Write-Host "npm not found. Install Node.js and add it to PATH." -ForegroundColor Red
    exit 1
}

if (-not $SkipNpmInstall) {
    if (-not (Test-Path (Join-Path $webDir "node_modules"))) {
        Write-Host "[web] node_modules missing, running npm install..." -ForegroundColor Yellow
        Push-Location $webDir
        try {
            npm.cmd install
            if ($LASTEXITCODE -ne 0) { throw "npm install failed, exit code $LASTEXITCODE" }
        }
        finally {
            Pop-Location
        }
    }
}

Write-Host ""
Write-Host "Opening two windows:" -ForegroundColor Cyan
Write-Host "  1) Backend (Air) - admin :19080, proxy :18080" -ForegroundColor Gray
Write-Host "  2) Web (Vite) - http://localhost:5173 , /api -> :19080" -ForegroundColor Gray
Write-Host ""
Write-Host "Open in browser: http://localhost:5173/" -ForegroundColor Green
Write-Host "Close each window to stop that process." -ForegroundColor DarkGray
Write-Host ""

$psArgs = @(
    "-NoExit"
    "-NoProfile"
    "-ExecutionPolicy"
    "Bypass"
    "-File"
    $backendPs1
)
Start-Process -FilePath "powershell.exe" -WorkingDirectory $projectRoot.Path -ArgumentList $psArgs

Start-Sleep -Milliseconds 500

$psArgsWeb = @(
    "-NoExit"
    "-NoProfile"
    "-ExecutionPolicy"
    "Bypass"
    "-File"
    $webPs1
)
Start-Process -FilePath "powershell.exe" -WorkingDirectory $webDir -ArgumentList $psArgsWeb

Write-Host "Started. If no new windows appear, check security software blocking Start-Process." -ForegroundColor Yellow
Write-Host ""
