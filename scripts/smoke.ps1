#Requires -Version 5.1
<#
.SYNOPSIS
    Full CLI integration smoke test: fresh temp pb_data + local mock provider.
.DESCRIPTION
    Flow (mirrors AGENTS.md "Smoke test reference flow"):
      config template -> config import -> provider list -> provider from-preset glm
      -> provider model add glm glm-5.3 -> start claude glm --model glm-5.3 --dry-run
      -> provider add local (mock on 127.0.0.1:18990) -> test provider local
      -> test models local -> profile add codex-local
      -> start codex codex-local --dry-run -> config export --include-secrets

    Assertions match the current lipgloss card output (provider list cards,
    "Launch Plan" dry-run card). Every failed step exits non-zero immediately.

    Usage:
      task smoke            # builds bin/aisw first, then runs this script
      pwsh -NoProfile -File scripts/smoke.ps1

    Env:
      SMOKE_ADDR   mock provider listen address (default 127.0.0.1:18990)
#>

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = (Resolve-Path (Join-Path $scriptDir "..")).Path
Set-Location $repoRoot

$aisw = Join-Path $repoRoot "bin\aisw.exe"
if (-not (Test-Path -LiteralPath $aisw)) {
    throw "bin/aisw.exe not found - run 'task build' first (or use 'task smoke')"
}

$addr = if ($env:SMOKE_ADDR) { $env:SMOKE_ADDR } else { "127.0.0.1:18990" }
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("aisw-smoke-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
$dataDir = Join-Path $tmp "pb_data"
$mockProc = $null

function Step([string]$msg) {
    Write-Host ""
    Write-Host "== $msg =="
}

function Assert-Contains([string]$label, [string]$haystack, [string]$needle) {
    if (-not $haystack.Contains($needle)) {
        Write-Host "ASSERT FAILED [$label]: expected output to contain: $needle"
        Write-Host "--- output ---"
        Write-Host $haystack
        exit 1
    }
    Write-Host "ok: $label"
}

function Invoke-Aisw {
    param([Parameter(ValueFromRemainingArguments = $true)][object[]]$cmdArgs)
    $output = & $script:aisw --dir $script:dataDir @cmdArgs 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) {
        Write-Host $output
        throw "aisw $($cmdArgs -join ' ') failed with exit code $LASTEXITCODE"
    }
    return $output
}

function Test-MockHealth([string]$address, [int]$timeoutSec) {
    try {
        Invoke-RestMethod -Uri "http://$address/health" -TimeoutSec $timeoutSec | Out-Null
        return $true
    } catch {
        return $false
    }
}

try {
    # ── mock provider ──────────────────────────────────────────────────────
    Step "mock provider ($addr)"
    if (Test-MockHealth $addr 2) {
        Write-Host "port already serving a healthy mock - reusing it (not killed on exit)"
    } else {
        $mockBin = Join-Path $tmp "mock-provider.exe"
        go build -o $mockBin ./cmd/mock-provider
        if ($LASTEXITCODE -ne 0) { throw "go build mock-provider failed" }
        $mockProc = Start-Process -FilePath $mockBin -ArgumentList "-http", $addr -PassThru -WindowStyle Hidden
        $ready = $false
        for ($i = 0; $i -lt 50; $i++) {
            if (Test-MockHealth $addr 1) { $ready = $true; break }
            Start-Sleep -Milliseconds 100
        }
        if (-not $ready) { throw "mock provider failed to start on $addr (port occupied?)" }
        Write-Host "mock provider up (pid $($mockProc.Id))"
    }

    # ── smoke flow ─────────────────────────────────────────────────────────
    Step "config template"
    Invoke-Aisw "config" "template" "--path" (Join-Path $tmp "smoke.toml") | Out-Host

    Step "config import"
    $out = Invoke-Aisw "config" "import" "--path" (Join-Path $tmp "smoke.toml") "--no-backup"
    Write-Host $out
    Assert-Contains "config import" $out "imported"

    Step "provider list"
    $out = Invoke-Aisw "provider" "list"
    Assert-Contains "provider list (saved cards)" $out "Saved Providers"
    Assert-Contains "provider list (imported glm)" $out "glm"
    Assert-Contains "provider list (preset cards)" $out "Built-in Provider Presets"

    Step "provider from-preset glm"
    $out = Invoke-Aisw "provider" "from-preset" "glm" "--api-key" "smoke-glm-key"
    Write-Host $out
    Assert-Contains "from-preset glm" $out "saved provider glm"

    Step "provider model add glm glm-5.3"
    $out = Invoke-Aisw "provider" "model" "add" "glm" "glm-5.3"
    Write-Host $out
    Assert-Contains "model add" $out "added model glm-5.3"

    Step "start claude glm --model glm-5.3 --dry-run"
    $out = Invoke-Aisw "start" "claude" "glm" "--model" "glm-5.3" "--dry-run"
    Assert-Contains "claude dry-run (plan card)" $out "Launch Plan"
    Assert-Contains "claude dry-run (model)" $out "glm-5.3"

    Step "provider add local (mock)"
    $out = Invoke-Aisw "provider" "add" "local" "--base-url" "http://$addr/v1" "--api-key" "smoke-local-key" "--protocol" "openai_chat" "--model" "gpt-task"
    Write-Host $out
    Assert-Contains "provider add local" $out "saved provider local"

    Step "test provider local"
    $out = Invoke-Aisw "test" "provider" "local"
    Write-Host $out
    Assert-Contains "test provider (status line)" $out "ok"
    Assert-Contains "test provider (http 200)" $out "200"

    Step "test models local"
    $out = Invoke-Aisw "test" "models" "local"
    Write-Host $out
    Assert-Contains "test models (mock model id)" $out "gpt-task"

    Step "profile add codex-local"
    $out = Invoke-Aisw "profile" "add" "codex-local" "--agent" "codex" "--provider" "local"
    Write-Host $out
    Assert-Contains "profile add" $out "saved profile codex-local"

    Step "start codex codex-local --dry-run"
    $out = Invoke-Aisw "start" "codex" "codex-local" "--dry-run"
    Assert-Contains "codex dry-run (plan card)" $out "Launch Plan"
    Assert-Contains "codex dry-run (provider)" $out "local"

    Step "config export --include-secrets"
    $exportPath = Join-Path $tmp "export.toml"
    $out = Invoke-Aisw "config" "export" "--include-secrets" "--path" $exportPath
    Write-Host $out
    Assert-Contains "config export" $out "exported"
    Assert-Contains "export contains secret" (Get-Content -LiteralPath $exportPath -Raw) "smoke-local-key"

    Step "SMOKE PASSED"
} finally {
    if ($null -ne $mockProc) {
        try { Stop-Process -Id $mockProc.Id -Force -ErrorAction SilentlyContinue } catch {}
    }
    if (Test-Path -LiteralPath $tmp) {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}
