# AI Agent 安装教程（Windows / macOS / Linux）

> 一键安装 `innate-aiswitcher` 支持的 AI Agent：Claude Code、Codex CLI、OpenCode。  
> 本教程使用 **fnm** 管理 Node.js 版本。

---

## 目录

1. [前置条件](#1-前置条件)
2. [一键安装脚本](#2-一键安装脚本)
   - [Windows](#21-windows)
   - [macOS / Linux](#22-macos--linux)
3. [手动安装](#3-手动安装)
4. [验证安装](#4-验证安装)
5. [与 innate-aiswitcher 配合使用](#5-与-innate-aiswitcher-配合使用)
6. [故障排查](#6-故障排查)

---

## 1. 前置条件

Claude Code、Codex CLI、OpenCode 都基于 **Node.js**，本教程使用 **fnm（Fast Node Manager）** 安装并管理 Node 版本。

- 一个可用的终端
- 能访问 npm registry 和 GitHub releases 的网络

---

## 2. 一键安装脚本

### 2.1 Windows

在项目根目录打开 PowerShell，执行：

```powershell
.\scripts\install-agents.ps1
```

脚本会自动完成：
1. 下载并安装 fnm 到 `~\.local\fnm`
2. 把 fnm 加入用户 PATH
3. 用 fnm 安装 Node.js LTS
4. 全局安装 Claude Code、Codex CLI、OpenCode
5. 把 fnm 初始化写入 PowerShell `$PROFILE`，以后新终端自动生效
6. 验证每个 Agent 的版本

### 2.2 macOS / Linux

在终端执行：

```bash
chmod +x scripts/install-agents.sh
./scripts/install-agents.sh
```

脚本会自动完成：
1. 安装 fnm
2. 把 fnm 初始化写入 `~/.zshrc` 或 `~/.bash_profile`
3. 用 fnm 安装 Node.js LTS
4. 全局安装 Claude Code、Codex CLI、OpenCode
5. 验证每个 Agent 的版本

---

## 3. 手动安装

### 3.1 安装 fnm

**Windows：**

```powershell
winget install Schniz.fnm
```

或使用脚本：

```powershell
$fnmDir = "$env:USERPROFILE\.local\fnm"
New-Item -ItemType Directory -Force -Path $fnmDir | Out-Null
$zip = "$env:TEMP\fnm-windows.zip"
Invoke-WebRequest -Uri "https://github.com/Schniz/fnm/releases/latest/download/fnm-windows.zip" -OutFile $zip -UseBasicParsing
Expand-Archive -Path $zip -DestinationPath $fnmDir -Force
Remove-Item $zip
$env:PATH = "$fnmDir;$env:PATH"
```

**macOS / Linux：**

```bash
curl -fsSL https://fnm.vercel.app/install | bash
```

或 Homebrew：

```bash
brew install fnm
```

### 3.2 初始化 fnm

**Windows（PowerShell）：**

```powershell
fnm env --use-on-cd | Out-String | Invoke-Expression
```

**macOS / Linux：**

```bash
eval "$(fnm env --use-on-cd)"
```

### 3.3 安装 Node.js LTS

```bash
fnm install --lts
fnm use --lts-if-available
node -v
npm -v
```

### 3.4 安装 Agents

```bash
npm install -g @anthropic-ai/claude-code
npm install -g @openai/codex
npm install -g opencode
```

---

## 4. 验证安装

```bash
claude --version
codex --version
opencode --version
```

如果提示命令找不到，通常是 npm 全局 bin 目录不在 PATH 中。脚本已经会自动处理，手动安装时可以参考下面的故障排查。

---

## 5. 与 innate-aiswitcher 配合使用

安装完成后，通过 `aisw` 配置 Provider 和 Profile：

```bash
# 查看内置 Provider 模板
.\bin\aisw provider presets        # Windows
./bin/aisw provider presets         # macOS/Linux

# 添加一个 OpenAI-compatible Provider（例如 MiniMax）
.\bin\aisw provider add minimax `
  --base-url https://api.minimax.chat/v1 `
  --api-key-env MINIMAX_API_KEY `
  --protocol openai_chat `
  --model MiniMax-M3

# 为 Codex 创建 Profile
.\bin\aisw profile add codex-minimax `
  --agent codex `
  --provider minimax `
  --model MiniMax-M3

# 启动 Codex
.\bin\aisw start codex codex-minimax
```

---

## 6. 故障排查

### 6.1 Windows 下提示命令找不到

npm 全局二进制文件通常在 `C:\Users\<用户名>\AppData\Roaming\npm`。检查是否加入 PATH：

```powershell
$npmBin = "$env:APPDATA\npm"
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$npmBin*") {
    [Environment]::SetEnvironmentVariable("Path", "$npmBin;$currentPath", "User")
    Write-Host "已添加 PATH，请重启终端"
}
```

### 6.2 macOS / Linux 下提示命令找不到

通常是 fnm 没有初始化。确认 shell profile 中有：

```bash
eval "$(fnm env --use-on-cd)"
```

然后执行：

```bash
source ~/.zshrc   # 或 ~/.bash_profile
```

### 6.3 fnm 安装失败

如果 winget / Homebrew 不可用，可以直接从 GitHub Releases 下载 fnm 二进制文件：

- Windows: `https://github.com/Schniz/fnm/releases/latest/download/fnm-windows.zip`
- macOS (Intel): `https://github.com/Schniz/fnm/releases/latest/download/fnm-macos.zip`
- macOS (Apple Silicon): `https://github.com/Schniz/fnm/releases/latest/download/fnm-macos-arm64.zip`
- Linux: `https://github.com/Schniz/fnm/releases/latest/download/fnm-linux.zip`

---

## 参考

| Agent | 官网 | npm 包 |
|-------|------|--------|
| Claude Code | https://claude.ai | `@anthropic-ai/claude-code` |
| Codex CLI | https://github.com/openai/codex | `@openai/codex` |
| OpenCode | https://opencode.ai | `opencode` |
| fnm | https://github.com/Schniz/fnm | - |
