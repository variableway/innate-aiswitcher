#Requires -Version 5.1
<#
.SYNOPSIS
    在 Windows 上一键安装 fnm、Node.js LTS、Claude Code、Codex CLI、OpenCode。
.DESCRIPTION
    本脚本会自动：
    1. 下载并安装 fnm 到 ~/.local/fnm
    2. 将 fnm 加入用户 PATH
    3. 用 fnm 安装 Node.js LTS
    4. 全局安装 Claude Code、Codex CLI、OpenCode
    5. 将 fnm 初始化写入 PowerShell $PROFILE
.NOTES
    需要 PowerShell 5.1+，以及能访问 GitHub releases 和 npm registry 的网络。
#>

$ErrorActionPreference = "Stop"

# ==============================================================================
# 辅助函数
# ==============================================================================

function Test-Command([string]$Name) {
    return [bool](Get-Command $Name -ErrorAction SilentlyContinue)
}

function Add-ToUserPath([string]$Dir) {
    $current = [Environment]::GetEnvironmentVariable("Path", "User")
    if ([string]::IsNullOrWhiteSpace($Dir)) { return $false }
    if ($current -like "*$Dir*") { return $false }
    [Environment]::SetEnvironmentVariable("Path", "$Dir;$current", "User")
    return $true
}

function Install-Fnm {
    Write-Host ""
    Write-Host "[1/6] 安装 fnm ..."
    if (Test-Command "fnm") {
        Write-Host "   fnm 已安装: $(fnm --version)"
        return
    }

    $fnmDir = "$env:USERPROFILE\.local\fnm"
    New-Item -ItemType Directory -Force -Path $fnmDir | Out-Null

    $zip = "$env:TEMP\fnm-windows.zip"
    $url = "https://github.com/Schniz/fnm/releases/latest/download/fnm-windows.zip"

    Write-Host "   下载: $url"
    Invoke-WebRequest -Uri $url -OutFile $zip -UseBasicParsing

    Write-Host "   解压到: $fnmDir"
    Expand-Archive -Path $zip -DestinationPath $fnmDir -Force
    Remove-Item -Path $zip -Force -ErrorAction SilentlyContinue

    if (Add-ToUserPath -Dir $fnmDir) {
        Write-Host "   已添加 fnm 到用户 PATH"
    }

    $env:PATH = "$fnmDir;$env:PATH"
    Write-Host "   fnm 安装完成: $(fnm --version)"
}

function Initialize-FnmInCurrentSession {
    Write-Host ""
    Write-Host "[2/6] 在当前会话中初始化 fnm ..."
    fnm env --use-on-cd | Out-String | Invoke-Expression
    Write-Host "   fnm 已初始化"
}

function Install-NodeLts {
    Write-Host ""
    Write-Host "[3/6] 安装 Node.js LTS ..."
    if (Test-Command "node") {
        Write-Host "   Node.js 已存在: $(node -v)"
    } else {
        Write-Host "   安装 Node.js LTS ..."
        fnm install --lts
        fnm use --lts-if-available
    }
    Write-Host "   Node.js: $(node -v)"
    Write-Host "   npm: $(npm -v)"
}

function Install-Agent([string]$Name, [string]$Package, [string]$Binary) {
    Write-Host ""
    Write-Host "[安装] $Name ($Package) ..."
    try {
        & npm install -g $Package 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "npm install 返回非零退出码: $LASTEXITCODE"
        }
    } catch {
        Write-Host "   安装失败: $_" -ForegroundColor Red
        return $false
    }

    # npm 安装后刷新 PATH（新加入的 npm bin 目录）
    $env:PATH = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
                [Environment]::GetEnvironmentVariable("Path", "User") + ";" +
                $env:PATH

    if (Test-Command $Binary) {
        Write-Host "   成功: $Binary " -NoNewline
        & $Binary --version 2>&1 | Select-Object -First 1
        return $true
    } else {
        Write-Host "   已安装但命令 $Binary 不在 PATH 中" -ForegroundColor Yellow
        return $false
    }
}

function Install-Agents {
    Write-Host ""
    Write-Host "[4/6] 安装 AI Agents ..."
    $script:claudeOk = Install-Agent -Name "Claude Code" -Package "@anthropic-ai/claude-code" -Binary "claude"
    $script:codexOk = Install-Agent -Name "Codex CLI" -Package "@openai/codex" -Binary "codex"
    $script:opencodeOk = Install-Agent -Name "OpenCode" -Package "opencode" -Binary "opencode"
}

function Add-NpmBinToPath {
    Write-Host ""
    Write-Host "[5/6] 检查 npm 全局目录 ..."
    $npmPrefix = (& npm config get prefix 2>&1 | Select-Object -First 1).Trim()
    $npmBin = Join-Path $npmPrefix ""
    $added = $false

    if (Test-Path $npmBin) {
        if (Add-ToUserPath -Dir $npmBin) {
            Write-Host "   已添加 npm bin 到用户 PATH: $npmBin"
            $added = $true
        } else {
            Write-Host "   npm bin 已在 PATH 中: $npmBin"
        }
    }

    $npmModulesBin = Join-Path $npmPrefix "node_modules\.bin"
    if (Test-Path $npmModulesBin) {
        if (Add-ToUserPath -Dir $npmModulesBin) {
            Write-Host "   已添加 npm modules bin 到用户 PATH: $npmModulesBin"
            $added = $true
        }
    }

    return $added
}

function Write-PowerShellProfile {
    Write-Host ""
    Write-Host "[6/6] 写入 PowerShell Profile ..."
    $profileDir = Split-Path $PROFILE -Parent
    if (-not (Test-Path $profileDir)) {
        New-Item -ItemType Directory -Force -Path $profileDir | Out-Null
    }
    if (-not (Test-Path $PROFILE)) {
        New-Item -ItemType File -Path $PROFILE | Out-Null
    }

    $marker = "# === innate-aiswitcher: fnm init ==="
    $existing = Get-Content $PROFILE -Raw -ErrorAction SilentlyContinue
    if ($existing -and $existing.Contains($marker)) {
        Write-Host "   Profile 中已存在 fnm 初始化，跳过"
        return
    }

    $block = @"

$marker
# 以下内容由 install-agents.ps1 自动生成
fnm env --use-on-cd | Out-String | Invoke-Expression
# === end ===
"@
    Add-Content -Path $PROFILE -Value $block -Encoding UTF8
    Write-Host "   已写入: $PROFILE"
}

function Print-Summary {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "安装结果" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green

    $results = @(
        @{ Name = "Claude Code"; Ok = $script:claudeOk },
        @{ Name = "Codex CLI"; Ok = $script:codexOk },
        @{ Name = "OpenCode"; Ok = $script:opencodeOk }
    )
    foreach ($r in $results) {
        $status = if ($r.Ok) { "OK" } else { "需要检查" }
        Write-Host "$($r.Name): $status"
    }

    Write-Host ""
    Write-Host "请重启终端，然后运行:" -ForegroundColor Yellow
    Write-Host "  claude --version"
    Write-Host "  codex --version"
    Write-Host "  opencode --version"
    Write-Host ""
    Write-Host "下一步: 配置 innate-aiswitcher"
    Write-Host "  .\bin\aisw provider presets"
    Write-Host "  .\bin\aisw provider add ..."
    Write-Host "  .\bin\aisw profile add ..."
}

# ==============================================================================
# 主流程
# ==============================================================================

Write-Host "========================================"
Write-Host "AI Agent Windows 一键安装脚本"
Write-Host "包含: fnm + Node.js LTS + Claude/Codex/OpenCode"
Write-Host "========================================"

Install-Fnm
Initialize-FnmInCurrentSession
Install-NodeLts
Install-Agents
$pathAdded = Add-NpmBinToPath
Write-PowerShellProfile
Print-Summary

if ($pathAdded) {
    Write-Host ""
    Write-Host "已更新用户 PATH，请重启 PowerShell 使更改生效。" -ForegroundColor Yellow
}