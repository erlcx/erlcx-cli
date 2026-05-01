param(
    [string]$Output = "dist\erlcx.exe"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$outputPath = Join-Path $repoRoot $Output
$outputDir = Split-Path -Parent $outputPath

New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

$env:GOOS = "windows"
$env:GOARCH = "amd64"
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if (-not $env:GOMODCACHE) {
    $env:GOMODCACHE = Join-Path $repoRoot ".gomodcache"
}

go build -trimpath -o $outputPath .\cmd\erlcx

Write-Host "Built $outputPath"
