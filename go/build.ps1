#Requires -Version 5.0
<#
.SYNOPSIS
    Build script for keep-screen-awake (Windows alternative to make).

.EXAMPLE
    .\build.ps1             # Windows binaries (default)
    .\build.ps1 -Darwin     # macOS arm64 binaries
    .\build.ps1 -Clean      # remove bin\
    .\build.ps1 -Test       # run tests
#>
param(
    [switch]$Darwin,
    [switch]$Clean,
    [switch]$Test
)

Set-Location $PSScriptRoot
$ErrorActionPreference = "Stop"

if ($Clean) {
    if (Test-Path bin) { Remove-Item -Recurse -Force bin }
    Write-Host "Cleaned." -ForegroundColor Green
    return
}

if ($Test) {
    go test ./...
    return
}

New-Item -ItemType Directory -Force -Path bin | Out-Null

if ($Darwin) {
    $env:GOOS   = "darwin"
    $env:GOARCH = "arm64"
    go build -o bin/ksad-darwin ./cmd/ksad
    go build -o bin/ksa-darwin  ./cmd/ksa
    Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
    Write-Host "Built: bin/ksad-darwin  bin/ksa-darwin" -ForegroundColor Green
} else {
    go build -o bin/ksad.exe ./cmd/ksad
    go build -o bin/ksa.exe  ./cmd/ksa
    Write-Host "Built: bin\ksad.exe  bin\ksa.exe" -ForegroundColor Green
}
