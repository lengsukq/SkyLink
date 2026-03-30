# Build web (vite build) and copy dist into static/web/dist for Go //go:embed.
# After copy, touches static/static.go so Air / next go build picks up new embedded files.
param(
    [switch] $SkipNpmInstall
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Resolve-Path (Join-Path $scriptDir "..")
$webDir = Join-Path $projectRoot "web"
$webDistDir = Join-Path $webDir "dist"
$embeddedDistDir = Join-Path $projectRoot "static\web\dist"
$staticGo = Join-Path $projectRoot "static\static.go"

function Test-Cmd($name) {
    return [bool](Get-Command $name -ErrorAction SilentlyContinue)
}

if (-not (Test-Cmd "npm")) {
    Write-Host "npm not found. Install Node.js and add it to PATH." -ForegroundColor Red
    exit 1
}

if (-not $SkipNpmInstall) {
    if (-not (Test-Path (Join-Path $webDir "node_modules"))) {
        Write-Host "[sync-web] node_modules missing, running npm install..." -ForegroundColor Yellow
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

Write-Host "[sync-web] npm run build..." -ForegroundColor Cyan
Push-Location $webDir
try {
    npm.cmd run build
    if ($LASTEXITCODE -ne 0) { throw "npm run build failed, exit code $LASTEXITCODE" }
}
finally {
    Pop-Location
}

Write-Host "[sync-web] copy web/dist -> static/web/dist..." -ForegroundColor Cyan
if (-not (Test-Path $webDistDir)) {
    throw "web dist not found: $webDistDir"
}
if (Test-Path $embeddedDistDir) {
    Remove-Item -Path $embeddedDistDir -Recurse -Force
}
New-Item -ItemType Directory -Path (Split-Path -Parent $embeddedDistDir) -Force | Out-Null
Copy-Item -Path $webDistDir -Destination $embeddedDistDir -Recurse

if (Test-Path $staticGo) {
    (Get-Item $staticGo).LastWriteTime = Get-Date
    Write-Host "[sync-web] touched static/static.go (trigger embed rebuild)" -ForegroundColor DarkGray
}

Write-Host "[sync-web] done." -ForegroundColor Green
