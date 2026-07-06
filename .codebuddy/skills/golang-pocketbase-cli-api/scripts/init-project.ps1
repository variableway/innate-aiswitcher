#!/usr/bin/env pwsh
<#
.SYNOPSIS
  Initialize a new Go + PocketBase + CLI + REST API project from template.

.DESCRIPTION
  Copies the project-template assets to the target directory,
  replaces module names, and runs `go mod tidy`.

.PARAMETER TargetDir
  Target directory for the new project (default: current directory).

.PARAMETER ModuleName
  Go module name (default: derived from directory name).

.EXAMPLE
  .\init-project.ps1 -TargetDir ../my-new-app -ModuleName github.com/me/myapp
#>
param(
    [string]$TargetDir = ".",
    [string]$ModuleName = ""
)

$ErrorActionPreference = "Stop"
$TemplateDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$TemplateDir = Join-Path $TemplateDir ".." "assets" "project-template"

if (-not (Test-Path $TemplateDir)) {
    Write-Error "Template directory not found: $TemplateDir"
    exit 1
}

# Resolve target
$TargetDir = Resolve-Path $TargetDir -ErrorAction SilentlyContinue
if (-not $TargetDir) {
    $TargetDir = Join-Path (Get-Location) (Split-Path $TargetDir -Leaf)
    New-Item -ItemType Directory -Force -Path $TargetDir | Out-Null
}
$TargetDir = (Resolve-Path $TargetDir).Path

# Module name
if (-not $ModuleName) {
    $ModuleName = "myapp"
}

Write-Host "Creating project in $TargetDir with module $ModuleName ..."

# Copy template
Copy-Item -Path "$TemplateDir\*" -Destination $TargetDir -Recurse -Force

# Replace module names in .go files
$goFiles = Get-ChildItem -Path $TargetDir -Filter "*.go" -Recurse
foreach ($file in $goFiles) {
    $content = Get-Content $file.FullName -Raw
    $content = $content -replace "myapp", $ModuleName
    Set-Content $file.FullName -Value $content -NoNewline
}

# Update go.mod module name
$modPath = Join-Path $TargetDir "go.mod"
if (Test-Path $modPath) {
    $modContent = Get-Content $modPath -Raw
    $modContent = $modContent -replace "module myapp", "module $ModuleName"
    Set-Content $modPath -Value $modContent -NoNewline
}

Write-Host "Project initialized!"
Write-Host ""
Write-Host "Next steps:"
Write-Host "  cd $TargetDir"
Write-Host "  go mod tidy"
Write-Host "  task serve"
