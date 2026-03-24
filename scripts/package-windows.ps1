Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Resolve-Path (Join-Path $scriptDir "..")
$webDir = Join-Path $projectRoot "web"
$webDistDir = Join-Path $webDir "dist"
$embeddedDistDir = Join-Path $projectRoot "static\web\dist"
$buildDir = Join-Path $projectRoot "build"
$outputExe = Join-Path $buildDir "skylink.exe"

Write-Host "[1/4] Build web assets..." -ForegroundColor Cyan
Push-Location $webDir
try {
    if (-not (Test-Path (Join-Path $webDir "node_modules"))) {
        Write-Host "node_modules not found, running npm install..." -ForegroundColor Yellow
        npm.cmd install
        if ($LASTEXITCODE -ne 0) { throw "npm install failed with exit code $LASTEXITCODE" }
    }
    npm.cmd run build
    if ($LASTEXITCODE -ne 0) { throw "npm run build failed with exit code $LASTEXITCODE" }
}
finally {
    Pop-Location
}

Write-Host "[2/4] Sync embedded static files..." -ForegroundColor Cyan
if (-not (Test-Path $webDistDir)) {
    throw "web dist directory not found: $webDistDir"
}
if (Test-Path $embeddedDistDir) {
    Remove-Item -Path $embeddedDistDir -Recurse -Force
}
New-Item -ItemType Directory -Path (Split-Path -Parent $embeddedDistDir) -Force | Out-Null
Copy-Item -Path $webDistDir -Destination $embeddedDistDir -Recurse

Write-Host "[3/4] Build Windows executable..." -ForegroundColor Cyan
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

Write-Host "[4/4] Done." -ForegroundColor Green
Write-Host "Output: $outputExe" -ForegroundColor Green
