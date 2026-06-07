# AI Agent 安装与配置教程

> 一键安装所有支持的 AI Agent，并通过 `innate-aiswitcher` 统一切换 Provider。

---

## 目录

1. [一键安装所有 Agent](#1-一键安装所有-agent)
2. [逐个 Agent 安装指南](#2-逐个-agent-安装指南)
   - [Claude Code](#21-claude-code)
   - [Codex CLI](#22-codex-cli)
   - [Hermes Agent](#23-hermes-agent)
   - [OpenCode](#24-opencode)
   - [DeepSeek 实验](#25-deepseek-实验)
   - [Kimi CLI](#26-kimi-cli)
   - [Gemini CLI](#27-gemini-cli)
   - [Trae CLI](#28-trae-cli)
3. [innate-aiswitcher 统一配置](#3-innate-aiswitcher-统一配置)
4. [快速切换示例](#4-快速切换示例)
5. [故障排查](#5-故障排查)

---

## 1. 一键安装所有 Agent

### macOS / Linux 一键脚本

```bash
#!/bin/bash
# install-all-agents.sh
# 一键安装所有 AI Agent（macOS / Linux）

set -e

echo "=== AI Agent 一键安装脚本 ==="
echo ""

# 检查依赖
echo "[1/8] 检查基础依赖..."
command -v curl >/dev/null 2>&1 || { echo "需要 curl"; exit 1; }
command -v node >/dev/null 2>&1 || echo "⚠️  Node.js 未安装，Codex/OpenCode 需要它"

# 1. Claude Code
echo "[2/8] 安装 Claude Code..."
npm install -g @anthropic-ai/claude-code 2>/dev/null || {
  echo "  尝试备用安装..."
  curl -fsSL https://claude.ai/install.sh | bash
}

# 2. Codex CLI
echo "[3/8] 安装 Codex CLI..."
npm install -g @openai/codex 2>/dev/null || {
  echo "  尝试备用安装..."
  curl -fsSL https://raw.githubusercontent.com/openai/codex/main/codex-cli/scripts/install.sh | bash
}

# 3. Hermes Agent
echo "[4/8] 安装 Hermes Agent..."
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash

# 4. OpenCode
echo "[5/8] 安装 OpenCode..."
curl -fsSL https://get.opencode.ai | bash

# 5. Kimi CLI
echo "[6/8] 安装 Kimi CLI..."
curl -fsSL https://kimi-cli.moonshot.cn/install.sh | bash 2>/dev/null || {
  echo "  ⚠️ Kimi CLI 安装失败，请手动访问 https://kimi-cli.moonshot.cn"
}

# 6. Gemini CLI
echo "[7/8] 安装 Gemini CLI..."
# Gemini CLI 通过 npm 安装
npm install -g @google/gemini-cli 2>/dev/null || {
  echo "  ⚠️ Gemini CLI 安装失败，请手动运行: npm install -g @google/gemini-cli"
}

# 7. Trae CLI
echo "[8/8] Trae CLI 暂无独立 CLI，请下载 Trae IDE: https://trae.ai"

echo ""
echo "=== 安装完成 ==="
echo "已安装: Claude Code, Codex CLI, Hermes Agent, OpenCode"
echo "部分安装: Kimi CLI, Gemini CLI"
echo "未安装: Trae CLI (需下载 IDE)"
echo ""
echo "下一步: 配置 innate-aiswitcher 统一切换 Provider"
echo "  运行: go run ./cmd/aisw"
```

**使用方法：**

```bash
# 下载并执行一键安装脚本
curl -fsSL https://raw.githubusercontent.com/your-repo/innate-aiswitcher/main/docs/tutorial/install-all-agents.sh | bash

# 或本地执行
chmod +x docs/tutorial/install-all-agents.sh
./docs/tutorial/install-all-agents.sh
```

---

## 2. 逐个 Agent 安装指南

### 2.1 Claude Code

**官网**: https://claude.ai/code

**依赖**: Node.js 18+

**一键安装:**

```bash
# 方式 1: npm（推荐）
npm install -g @anthropic-ai/claude-code

# 方式 2: 官方安装脚本
curl -fsSL https://claude.ai/install.sh | bash

# 方式 3: Homebrew (macOS)
brew install claude-code
```

**验证安装:**

```bash
claude --version
```

**innate-aiswitcher 配置:**

```bash
# 添加 Provider
aisw provider add minimax-claude \
  --base-url https://api.minimax.chat/anthropic \
  --api-key-env MINIMAX_API_KEY \
  --protocol anthropic \
  --model claude-3-5-sonnet-20241022 \
  --endpoint messages=/v1/messages \
  --endpoint models=/v1/models

# 创建 Profile
aisw profile add claude-minimax \
  --agent claude \
  --provider minimax-claude \
  --model claude-3-5-sonnet-20241022

# 启动
aisw start claude claude-minimax
```

---

### 2.2 Codex CLI

**官网**: https://github.com/openai/codex

**依赖**: Node.js 18+, OpenAI API Key

**一键安装:**

```bash
# 方式 1: npm（推荐）
npm install -g @openai/codex

# 方式 2: 官方安装脚本
curl -fsSL https://raw.githubusercontent.com/openai/codex/main/codex-cli/scripts/install.sh | bash
```

**验证安装:**

```bash
codex --version
```

**innate-aiswitcher 配置:**

```bash
# 添加 Provider
aisw provider add minimax-openai \
  --base-url https://api.minimax.chat/v1 \
  --api-key-env MINIMAX_API_KEY \
  --protocol openai_chat \
  --model MiniMax-M1 \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 创建 Profile
aisw profile add codex-minimax \
  --agent codex \
  --provider minimax-openai \
  --model MiniMax-M1

# 启动
aisw start codex codex-minimax
```

---

### 2.3 Hermes Agent

**官网**: https://hermes-agent.nousresearch.com

**依赖**: Python 3.11+, uv (Python 包管理器), Node.js 20+ (可选，用于浏览器工具)

**一键安装:**

```bash
# 官方安装脚本（推荐）
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash

# 安装后重新加载 shell
source ~/.bashrc  # 或 source ~/.zshrc
```

**手动安装（无安装脚本时）:**

```bash
# 1. 安装 uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# 2. 克隆仓库
git clone https://github.com/NousResearch/hermes-agent.git
cd hermes-agent

# 3. 创建虚拟环境
uv venv venv --python 3.11
export VIRTUAL_ENV="$(pwd)/venv"

# 4. 安装
uv pip install -e ".[all]"

# 5. 创建符号链接
mkdir -p ~/.local/bin
ln -sf "$(pwd)/venv/bin/hermes" ~/.local/bin/hermes
```

**验证安装:**

```bash
hermes --version
hermes doctor
```

**innate-aiswitcher 配置:**

```bash
# 添加 Provider（OpenAI）
aisw provider add openai \
  --base-url https://api.openai.com/v1 \
  --api-key-env OPENAI_API_KEY \
  --protocol openai_chat \
  --model gpt-4o-mini \
  --endpoint chat_completions=/chat/completions \
  --endpoint responses=/responses \
  --endpoint models=/models

# 添加 Provider（DeepSeek 实验）
aisw provider add deepseek \
  --base-url https://api.deepseek.com/v1 \
  --api-key-env DEEPSEEK_API_KEY \
  --protocol openai_chat \
  --model deepseek-chat \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 创建 Profile
aisw profile add hermes-openai \
  --agent hermes \
  --provider openai \
  --model gpt-4o-mini

aisw profile add hermes-deepseek \
  --agent hermes \
  --provider deepseek \
  --model deepseek-chat

# 启动
aisw start hermes hermes-openai
aisw start hermes hermes-deepseek
```

---

### 2.4 OpenCode

**官网**: https://opencode.ai

**依赖**: Node.js 18+

**一键安装:**

```bash
# 官方安装脚本
curl -fsSL https://get.opencode.ai | bash

# 或 npm
npm install -g opencode
```

**验证安装:**

```bash
opencode --version
```

**innate-aiswitcher 配置:**

```bash
# 添加 Provider
aisw provider add openai \
  --base-url https://api.openai.com/v1 \
  --api-key-env OPENAI_API_KEY \
  --protocol openai_chat \
  --model gpt-4o-mini

# 创建 Profile
aisw profile add opencode-default \
  --agent opencode \
  --provider openai \
  --model gpt-4o-mini

# 启动
aisw start opencode opencode-default
```

---

### 2.5 DeepSeek 实验

**说明**: DeepSeek 不是独立的 Agent，而是通过 Provider 配置供其他 Agent 使用。

**官网**: https://platform.deepseek.com

**依赖**: 无需安装，只需 API Key

**获取 API Key:**

1. 访问 https://platform.deepseek.com
2. 注册账号
3. 创建 API Key
4. 设置环境变量: `export DEEPSEEK_API_KEY="sk-..."`

**innate-aiswitcher 配置:**

```bash
# 添加 DeepSeek Provider
aisw provider add deepseek \
  --base-url https://api.deepseek.com/v1 \
  --api-key-env DEEPSEEK_API_KEY \
  --protocol openai_chat \
  --model deepseek-chat \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 与 Hermes 组合
aisw profile add hermes-deepseek \
  --agent hermes \
  --provider deepseek \
  --model deepseek-chat

# 与 Codex 组合
aisw profile add codex-deepseek \
  --agent codex \
  --provider deepseek \
  --model deepseek-chat

# 启动
aisw start hermes hermes-deepseek
aisw start codex codex-deepseek
```

**可用模型:**

| 模型 | 说明 |
|------|------|
| `deepseek-chat` | 标准对话模型 |
| `deepseek-reasoner` | 推理模型（需 Profile 覆盖） |

**与所有 Agent 的组合:**

```bash
# Hermes + DeepSeek
aisw profile add hermes-deepseek --agent hermes --provider deepseek --model deepseek-chat

# Codex + DeepSeek
aisw profile add codex-deepseek --agent codex --provider deepseek --model deepseek-chat

# OpenCode + DeepSeek
aisw profile add opencode-deepseek --agent opencode --provider deepseek --model deepseek-chat

# 启动
aisw start hermes hermes-deepseek
aisw start codex codex-deepseek
aisw start opencode opencode-deepseek
```

---

### 2.5b MiniMax 适配所有 Agent

**官网**: https://www.minimaxi.com

**依赖**: 无需安装，只需 API Key

**获取 API Key:**

1. 访问 https://www.minimaxi.com
2. 注册开发者账号
3. 创建应用获取 API Key
4. 设置环境变量: `export MINIMAX_API_KEY="..."`

**innate-aiswitcher 配置（适配所有 Agent）:**

```bash
# 添加 MiniMax OpenAI-compatible（供 Codex/Hermes/OpenCode/Kimi 使用）
aisw provider add minimax-openai \
  --base-url https://api.minimax.chat/v1 \
  --api-key-env MINIMAX_API_KEY \
  --protocol openai_chat \
  --model MiniMax-M1 \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 添加 MiniMax Claude-compatible（供 Claude Code 使用）
aisw provider add minimax-claude \
  --base-url https://api.minimax.chat/anthropic \
  --api-key-env MINIMAX_API_KEY \
  --protocol anthropic \
  --model claude-3-5-sonnet-20241022 \
  --endpoint messages=/v1/messages \
  --endpoint models=/v1/models

# 与所有 Agent 组合
aisw profile add hermes-minimax --agent hermes --provider minimax-openai --model MiniMax-M1
aisw profile add codex-minimax --agent codex --provider minimax-openai --model MiniMax-M1
aisw profile add claude-minimax --agent claude --provider minimax-claude --model claude-3-5-sonnet-20241022
aisw profile add opencode-minimax --agent opencode --provider minimax-openai --model MiniMax-M1

# 启动
aisw start hermes hermes-minimax
aisw start codex codex-minimax
aisw start claude claude-minimax
aisw start opencode opencode-minimax
```

---

### 2.5c Kimi 适配所有 Agent

**官网**: https://platform.moonshot.cn

**依赖**: 无需安装，只需 API Key

**获取 API Key:**

1. 访问 https://platform.moonshot.cn
2. 注册开发者账号
3. 创建应用获取 API Key
4. 设置环境变量: `export KIMI_API_KEY="sk-..."`

**innate-aiswitcher 配置（适配所有 Agent）:**

```bash
# 添加 Kimi OpenAI-compatible（供 Codex/Hermes/OpenCode 使用）
aisw provider add kimi-openai \
  --base-url https://api.moonshot.cn/v1 \
  --api-key-env KIMI_API_KEY \
  --protocol openai_chat \
  --model moonshot-v1-8k \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 添加 Kimi Claude-compatible（供 Claude Code 使用）
aisw provider add kimi-claude \
  --base-url https://api.moonshot.cn/anthropic \
  --api-key-env KIMI_API_KEY \
  --protocol anthropic \
  --model claude-3-5-sonnet-20241022 \
  --endpoint messages=/v1/messages \
  --endpoint models=/v1/models

# 与所有 Agent 组合
aisw profile add hermes-kimi --agent hermes --provider kimi-openai --model moonshot-v1-8k
aisw profile add codex-kimi --agent codex --provider kimi-openai --model moonshot-v1-8k
aisw profile add claude-kimi --agent claude --provider kimi-claude --model claude-3-5-sonnet-20241022
aisw profile add opencode-kimi --agent opencode --provider kimi-openai --model moonshot-v1-8k

# 启动
aisw start hermes hermes-kimi
aisw start codex codex-kimi
aisw start claude claude-kimi
aisw start opencode opencode-kimi
```

---

### 2.5d Xiaomi MiMo 适配所有 Agent

**官网**: https://mimo.xiaomi.com

**依赖**: 无需安装，只需 API Key

**获取 API Key:**

1. 访问 https://mimo.xiaomi.com
2. 注册开发者账号
3. 创建应用获取 API Key
4. 设置环境变量: `export XIAOMI_API_KEY="sk-..."`

**innate-aiswitcher 配置（适配所有 Agent）:**

```bash
# 添加 Xiaomi MiMo OpenAI-compatible
aisw provider add xiaomi-openai \
  --base-url https://api.mimo.xiaomi.com/v1 \
  --api-key-env XIAOMI_API_KEY \
  --protocol openai_chat \
  --model mimo-7b \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 与 Hermes 组合（Xiaomi 目前主要支持 openai_chat 协议）
aisw profile add hermes-xiaomi --agent hermes --provider xiaomi-openai --model mimo-7b
aisw profile add codex-xiaomi --agent codex --provider xiaomi-openai --model mimo-7b

# 启动
aisw start hermes hermes-xiaomi
aisw start codex codex-xiaomi
```

---

### 2.5e 防封代理模式（Hermes + DeepSeek 调度其他 Agent）

> 详细指南见 [HERMES_DEEPSEEK_PROXY.md](HERMES_DEEPSEEK_PROXY.md)

**核心思路**: 使用 Hermes + DeepSeek 作为唯一对外出口，其他 Agent 完全本地运行，避免直接暴露原始 Provider。

```bash
# 1. 配置 DeepSeek 作为代理层
aisw provider add deepseek \
  --base-url https://api.deepseek.com/v1 \
  --api-key-env DEEPSEEK_API_KEY \
  --protocol openai_chat \
  --model deepseek-chat

# 2. 配置本地 Agent 的 Provider（MiniMax/Kimi/Xiaomi）
# ...（见上文各 Provider 配置）

# 3. 创建 Hermes 代理 Profile
aisw profile add hermes-deepseek --agent hermes --provider deepseek --model deepseek-chat

# 4. 启动 Hermes 代理
aisw start hermes hermes-deepseek

# 5. 在 Hermes 中调度其他本地 Agent
# 输入: "用 Claude 分析代码"
# 然后执行: aisw start claude claude-minimax
```

**防封策略**:
- 单一出口: 只有 DeepSeek 对外发起请求
- 临时配置: 每次启动都是全新的临时目录
- Key 隔离: 每个 Agent 使用独立的本地 Profile
- 本地兜底: 可配置 Ollama 本地模型作为备用

---

### 2.6 Kimi CLI

**官网**: https://kimi-cli.moonshot.cn

**依赖**: Node.js 18+

**一键安装:**

```bash
# 官方安装脚本
curl -fsSL https://kimi-cli.moonshot.cn/install.sh | bash

# 或 npm
npm install -g kimi-cli
```

**验证安装:**

```bash
kimi --version
```

**innate-aiswitcher 配置:**

```bash
# 添加 Provider
aisw provider add kimi-openai \
  --base-url https://api.moonshot.cn/v1 \
  --api-key-env KIMI_API_KEY \
  --protocol openai_chat \
  --model moonshot-v1-8k \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 创建 Profile
aisw profile add kimi-default \
  --agent kimi \
  --provider kimi-openai \
  --model moonshot-v1-8k

# 启动
aisw start kimi kimi-default
```

---

### 2.7 Gemini CLI

**官网**: https://ai.google.dev/gemini-cli

**依赖**: Node.js 18+

**一键安装:**

```bash
# npm
npm install -g @google/gemini-cli
```

**验证安装:**

```bash
gemini --version
```

**innate-aiswitcher 配置:**

```bash
# 添加 Provider
aisw provider add gemini \
  --base-url https://generativelanguage.googleapis.com \
  --api-key-env GEMINI_API_KEY \
  --protocol gemini_native \
  --model gemini-2.0

# 创建 Profile
aisw profile add gemini-default \
  --agent gemini \
  --provider gemini \
  --model gemini-2.0

# 启动
aisw start gemini gemini-default
```

---

### 2.8 Trae CLI

**说明**: Trae 目前主要是 IDE，暂无独立 CLI。

**官网**: https://trae.ai

**下载地址:**

```bash
# macOS
curl -L "https://trae.ai/download/macos" -o Trae.dmg
open Trae.dmg

# Windows
curl -L "https://trae.ai/download/windows" -o Trae.exe
# 运行安装程序

# Linux
curl -L "https://trae.ai/download/linux" -o Trae.AppImage
chmod +x Trae.AppImage
./Trae.AppImage
```

**innate-aiswitcher 配置:**

```bash
# 添加 Provider
aisw provider add openai \
  --base-url https://api.openai.com/v1 \
  --api-key-env OPENAI_API_KEY \
  --protocol openai_chat \
  --model gpt-4o-mini

# 创建 Profile
aisw profile add trae-default \
  --agent trae \
  --provider openai \
  --model gpt-4o-mini

# 启动（Trae 使用 openai_env adapter）
aisw start trae trae-default
```

---

## 3. innate-aiswitcher 统一配置

### 3.1 安装 innate-aiswitcher

```bash
# 克隆仓库
git clone https://github.com/your-repo/innate-aiswitcher.git
cd innate-aiswitcher

# 构建
task build

# 或直接使用 Go
go build -o bin/aisw ./cmd/aisw
```

### 3.2 配置流程

```bash
# 1. 启动 TUI 交互配置
./bin/aisw

# 2. 或命令行配置
# 查看内置预设
./bin/aisw provider presets

# 3. 添加 Provider
./bin/aisw provider add deepseek \
  --base-url https://api.deepseek.com/v1 \
  --api-key-env DEEPSEEK_API_KEY \
  --protocol openai_chat \
  --model deepseek-chat

# 4. 创建 Profile
./bin/aisw profile add hermes-deepseek \
  --agent hermes \
  --provider deepseek \
  --model deepseek-chat

# 5. 测试连通性
./bin/aisw test provider deepseek

# 6. 启动 Agent
./bin/aisw start hermes hermes-deepseek
```

### 3.3 环境变量准备

```bash
# 在 ~/.bashrc 或 ~/.zshrc 中添加
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
export DEEPSEEK_API_KEY="sk-..."
export MINIMAX_API_KEY="sk-..."
export KIMI_API_KEY="sk-..."
export GEMINI_API_KEY="..."
export MOONSHOT_API_KEY="sk-..."
```

---

## 4. 快速切换示例

### 场景 1: 同一 Agent，切换不同 Provider

```bash
# Claude + MiniMax
aisw start claude claude-minimax

# Claude + Anthropic
aisw start claude claude-anthropic
```

### 场景 2: 同一 Provider，切换不同 Agent

```bash
# DeepSeek + Hermes
aisw start hermes hermes-deepseek

# DeepSeek + Codex
aisw start codex codex-deepseek
```

### 场景 3: TUI 交互切换

```bash
# 启动交互界面
aisw

# 选择:
# 1. Start an agent session
# 2. 选择 Agent (Claude / Codex / Hermes / ...)
# 3. 选择 Provider/Profile
# 4. 选择 Dry-run 或 Start now
```

### 场景 4: 一键切换脚本

```bash
#!/bin/bash
# switch-to-deepseek.sh

echo "切换到 DeepSeek + Hermes..."
aisw start hermes hermes-deepseek
```

```bash
#!/bin/bash
# switch-to-openai.sh

echo "切换到 OpenAI + Claude..."
aisw start claude claude-openai
```

---

## 5. 故障排查

### 5.1 安装问题

| 问题 | 解决 |
|------|------|
| `npm: command not found` | 安装 Node.js: `brew install node` 或访问 https://nodejs.org |
| `uv: command not found` | 安装 uv: `curl -LsSf https://astral.sh/uv/install.sh | sh` |
| `python3: command not found` | 安装 Python: `brew install python@3.11` |
| Hermes 安装失败 | 检查 Python 版本 >= 3.11，或尝试手动安装 |

### 5.2 配置问题

| 问题 | 解决 |
|------|------|
| `provider not found` | 先运行 `aisw provider add ...` |
| `agent not found` | 确认 Agent 已安装，运行 `aisw provider presets` 查看 |
| `unsupported adapter` | 检查 `agents` 表中的 adapter 字段 |
| `API key missing` | 设置环境变量，如 `export DEEPSEEK_API_KEY="sk-..."` |

### 5.3 连通性问题

```bash
# 测试 Provider
aisw test provider deepseek

# 查看详细错误
aisw test provider deepseek --model deepseek-chat

# Dry-run 查看启动计划
aisw start hermes hermes-deepseek --dry-run
```

---

## 附录: 所有 Agent 速查表

| Agent | 安装命令 | 官网 | Adapter | 重点 |
|-------|---------|------|---------|------|
| Claude Code | `npm i -g @anthropic-ai/claude-code` | https://claude.ai/code | `claude` | ⭐ |
| Codex CLI | `npm i -g @openai/codex` | https://github.com/openai/codex | `codex` | ⭐ |
| Hermes | `curl -fsSL https://hermes-agent.nousresearch.com/install.sh \| bash` | https://hermes-agent.nousresearch.com | `hermes` | ⭐⭐ |
| OpenCode | `curl -fsSL https://get.opencode.ai \| bash` | https://opencode.ai | `openai_env` | ⭐ |
| DeepSeek | 无需安装，只需 API Key | https://platform.deepseek.com | - | ⭐⭐ |
| Kimi | `npm i -g kimi-cli` | https://kimi-cli.moonshot.cn | `openai_env` | |
| Gemini | `npm i -g @google/gemini-cli` | https://ai.google.dev/gemini-cli | `gemini` | |
| Trae | 下载 IDE | https://trae.ai | `openai_env` | |

---

*本教程对应 innate-aiswitcher 当前代码版本。如有更新，请以官方文档为准。*
