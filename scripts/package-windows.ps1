Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Resolve-Path (Join-Path $scriptDir "..")
$syncPs1 = Join-Path $scriptDir "sync-web-to-static.ps1"
$buildDir = Join-Path $projectRoot "build"
$outputExe = Join-Path $buildDir "skylink.exe"

Write-Host "[1/2] Build web + sync static (go:embed)..." -ForegroundColor Cyan
& $syncPs1
if (-not $?) { throw "sync-web-to-static failed" }

Write-Host "[2/2] Build Windows executable..." -ForegroundColor Cyan
if (-not (Test-Path $buildDir)) {
    New-Item -ItemType Directory -Path $buildDir | Out-Null
}
Push-Location $projectRoot
try {
    go build -trimpath -o $outputExe ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }
}
finally {
    Pop-Location
}

Write-Host "Done." -ForegroundColor Green
Write-Host "Output: $outputExe" -ForegroundColor Green
