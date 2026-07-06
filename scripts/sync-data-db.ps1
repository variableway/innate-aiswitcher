#Requires -Version 5.1
<#
.SYNOPSIS
    Copy the bundled data.db into the default innate-aiswitcher PocketBase data dir.
.DESCRIPTION
    Source (default): <repo-root>/data.db
    Target (default): %USERPROFILE%\.innate-aiswitcher\pb_data\data.db

    Backs up an existing target database before overwrite and removes stale
    SQLite sidecar files (journal / wal / shm).
.PARAMETER Source
    Path to the source data.db file.
.PARAMETER DestDir
    PocketBase data directory (data.db is written inside it).
.EXAMPLE
    .\scripts\sync-data-db.ps1
.EXAMPLE
    .\scripts\sync-data-db.ps1 -Source .\data.db
#>

param(
    [string]$Source = "",
    [string]$DestDir = ""
)

$ErrorActionPreference = "Stop"

function Resolve-RepoRoot {
    return (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
}

function Get-DefaultDestDir {
    return Join-Path $env:USERPROFILE ".innate-aiswitcher" "pb_data"
}

if ([string]::IsNullOrWhiteSpace($Source)) {
    $Source = Join-Path (Resolve-RepoRoot) "data.db"
}
if ([string]::IsNullOrWhiteSpace($DestDir)) {
    $DestDir = Get-DefaultDestDir
}

$Source = (Resolve-Path -LiteralPath $Source).Path
$DestDir = [System.IO.Path]::GetFullPath($DestDir)
$DestFile = Join-Path $DestDir "data.db"

if (-not (Test-Path -LiteralPath $Source)) {
    throw "source database not found: $Source"
}

New-Item -ItemType Directory -Force -Path $DestDir | Out-Null

$sidecars = @("data.db-journal", "data.db-wal", "data.db-shm")
foreach ($name in $sidecars) {
    $path = Join-Path $DestDir $name
    if (Test-Path -LiteralPath $path) {
        Remove-Item -LiteralPath $path -Force
        Write-Host "removed stale $name"
    }
}

if (Test-Path -LiteralPath $DestFile) {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $backup = Join-Path $DestDir "data.db.bak-$stamp"
    Copy-Item -LiteralPath $DestFile -Destination $backup -Force
    Write-Host "backed up existing database to $backup"
}

$temp = Join-Path $DestDir "data.db.tmp"
Copy-Item -LiteralPath $Source -Destination $temp -Force
Move-Item -LiteralPath $temp -Destination $DestFile -Force

Write-Host "synced $Source"
Write-Host "    -> $DestFile"
