# SkyLink Windows dev: optional embed sync, then backend (Air) + frontend (Vite, default).
#
# Run: dev-windows.cmd
#   (default)         npm run build + sync static/web + Air + Vite on :5173
#   -SkipWebBuild      skip npm build/sync (faster; use Vite-only UI at http://localhost:5173)
#   -SkipNpmInstall    skip npm ci in sync script when node_modules exists; Vite window still ensures deps for dev
#   -NoVite             only Air (single window); open http://127.0.0.1:19080/ for embedded UI

param(
    [switch] $SkipNpmInstall,
    [switch] $SkipWebBuild,
    [switch] $NoVite
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Resolve-Path (Join-Path $scriptDir "..")
$webDir = Join-Path $projectRoot.Path "web"
$backendPs1 = Join-Path $scriptDir "dev-windows-backend.ps1"
$webPs1 = Join-Path $scriptDir "dev-windows-web.ps1"
$syncPs1 = Join-Path $scriptDir "sync-web-to-static.ps1"

function Test-Cmd($name) {
    return [bool](Get-Command $name -ErrorAction SilentlyContinue)
}

if (-not (Test-Cmd "go")) {
    Write-Host "go not found. Install Go and add it to PATH." -ForegroundColor Red
    exit 1
}

$needNpm = (-not $SkipWebBuild) -or (-not $NoVite)
if ($needNpm -and -not (Test-Cmd "npm")) {
    Write-Host "npm not found. Install Node.js and add it to PATH." -ForegroundColor Red
    exit 1
}

if (-not $SkipWebBuild) {
    Write-Host ""
    Write-Host "Packaging latest web into static/web/dist (go:embed)..." -ForegroundColor Cyan
    if ($SkipNpmInstall) {
        & $syncPs1 -SkipNpmInstall
    }
    else {
        & $syncPs1
    }
    if (-not $?) { exit 1 }
    Write-Host ""
}

if (-not $NoVite) {
    if (-not $SkipNpmInstall -and -not (Test-Path (Join-Path $webDir "node_modules"))) {
        Write-Host "[web] node_modules missing, running npm ci..." -ForegroundColor Yellow
        Push-Location $webDir
        try {
            npm.cmd ci
            if ($LASTEXITCODE -ne 0) { throw "npm ci failed, exit code $LASTEXITCODE" }
        }
        finally {
            Pop-Location
        }
    }
}

Write-Host "Opening windows:" -ForegroundColor Cyan
Write-Host "  1) Backend (Air) - admin :19080, proxy :18080" -ForegroundColor Gray
if (-not $NoVite) {
    Write-Host "  2) Web (Vite)    - http://localhost:5173  (/api -> :19080)" -ForegroundColor Gray
}
Write-Host ""
Write-Host "Use the Vite URL for daily dev (HMR). Embedded UI: http://127.0.0.1:19080/" -ForegroundColor Green
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

if (-not $NoVite) {
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
}

Write-Host "Started." -ForegroundColor Yellow
Write-Host ""
