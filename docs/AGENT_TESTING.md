# Agent 测试指南

本文档说明如何为 `innate-aiswitcher` 支持的各个 Agent（Claude Code、Codex CLI、OpenCode 等）配置 Provider、运行连通性测试，以及验证启动计划。

---

## 支持的 Agent 与 Adapter 对照

| Agent | Binary | Adapter | 说明 |
|-------|--------|---------|------|
| **Claude Code** | `claude` | `claude` | 生成临时 `settings.json`，通过 `--settings` 传入 |
| **Codex CLI** | `codex` | `codex` | 生成临时 `CODEX_HOME/config.toml` + `auth.json` |
| **Gemini CLI** | `gemini` | `gemini` | 设置 `GEMINI_API_KEY` + `GOOGLE_GEMINI_BASE_URL` |
| **Kimi CLI** | `kimi` | `openai_env` | 设置 `OPENAI_API_KEY` + `OPENAI_BASE_URL` |
| **Trae CLI** | `trae` | `openai_env` | 设置 `OPENAI_API_KEY` + `OPENAI_BASE_URL` |
| **OpenCode** | `opencode` | `openai_env` | 设置 `OPENAI_API_KEY` + `OPENAI_BASE_URL` |

> **注意**：`test provider` 和 `test models` 是 **Provider 级别**的测试，不依赖具体 Agent。任何支持 `anthropic`、`openai_chat` 或 `openai_responses` 协议的 Provider 都可以被测试。

---

## 通用测试命令

### 1. 测试 Provider 连通性

发送一个最小请求验证 API Key 和模型是否可用：

```bash
./bin/aisw test provider <provider-slug>
```

可选覆盖模型：
```bash
./bin/aisw test provider <provider-slug> --model <model-name>
```

### 2. 测试模型列表

拉取 Provider 支持的模型列表：

```bash
./bin/aisw test models <provider-slug>
```

### 3. Dry Run 预览启动计划

在真正启动 Agent 前，预览生成的配置：

```bash
./bin/aisw start <agent> <profile-or-provider> --dry-run
```

---

## Claude Code 测试流程

### 前提
- 已安装 Claude Code：`npm install -g @anthropic-ai/claude-code`
- Claude Code 使用 **Anthropic Messages API** 兼容端点

### 配置步骤

```bash
# 1. 添加 Provider（Anthropic 兼容模式）
./bin/aisw provider add minimax-claude \
  --name "MiniMax (Claude-compatible)" \
  --base-url https://api.minimax.chat/anthropic \
  --api-key-env MINIMAX_API_KEY \
  --protocol anthropic \
  --model MiniMax-M3 \
  --endpoint messages=/v1/messages \
  --endpoint models=/v1/models

# 2. 测试连通性
./bin/aisw test provider minimax-claude

# 3. 查看可用模型
./bin/aisw test models minimax-claude

# 4. 创建 Profile
./bin/aisw profile add claude-minimax \
  --agent claude \
  --provider minimax-claude \
  --model MiniMax-M3 \
  --default

# 5. Dry Run 预览
./bin/aisw start claude claude-minimax --dry-run

# 6. 实际启动
./bin/aisw start claude claude-minimax
```

### 启动原理

Claude Adapter 生成临时 `settings.json`：
```json
{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "<api-key>",
    "ANTHROPIC_API_KEY": "<api-key>",
    "ANTHROPIC_BASE_URL": "https://api.minimax.chat/anthropic",
    "ANTHROPIC_MODEL": "MiniMax-M3",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "MiniMax-M3",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "MiniMax-M3",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "MiniMax-M3"
  }
}
```

启动命令：
```bash
claude --settings /tmp/aisw-claude-xxx.json
```

---

## Codex CLI 测试流程

### 前提
- 已安装 Codex CLI：`npm install -g @openai/codex`
- Codex 使用 **OpenAI Responses API** 或 **Chat Completions API**

### 配置步骤

```bash
# 1. 添加 Provider（OpenAI 兼容模式）
./bin/aisw provider add minimax-codex \
  --name "MiniMax (Codex-compatible)" \
  --base-url https://api.minimax.chat/v1 \
  --api-key-env MINIMAX_API_KEY \
  --protocol openai_chat \
  --model MiniMax-M3 \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 2. 测试连通性
./bin/aisw test provider minimax-codex

# 3. 创建 Profile
./bin/aisw profile add codex-minimax \
  --agent codex \
  --provider minimax-codex \
  --model MiniMax-M3 \
  --default

# 4. Dry Run 预览
./bin/aisw start codex codex-minimax --dry-run

# 5. 实际启动
./bin/aisw start codex codex-minimax
```

### 启动原理

Codex Adapter 创建临时目录 `CODEX_HOME`：
```toml
# CODEX_HOME/config.toml
model_provider = "minimax-codex"
model = "MiniMax-M3"

[model_providers.minimax-codex]
name = "minimax-codex"
base_url = "https://api.minimax.chat/v1"
wire_api = "chat"
requires_openai_auth = true
```

```json
// CODEX_HOME/auth.json
{"OPENAI_API_KEY": "<api-key>"}
```

启动命令：
```bash
CODEX_HOME=/tmp/aisw-codex-xxx codex
```

---

## OpenCode 测试流程

### 前提
- 已安装 OpenCode CLI
- OpenCode 使用 **OpenAI Chat Completions API** 兼容端点

### 配置步骤

```bash
# 1. 添加 Provider（OpenAI 兼容模式）
./bin/aisw provider add minimax-opencode \
  --name "MiniMax (OpenCode-compatible)" \
  --base-url https://api.minimax.chat/v1 \
  --api-key-env MINIMAX_API_KEY \
  --protocol openai_chat \
  --model MiniMax-M3 \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 2. 测试连通性
./bin/aisw test provider minimax-opencode

# 3. 创建 Profile
./bin/aisw profile add opencode-minimax \
  --agent opencode \
  --provider minimax-opencode \
  --model MiniMax-M3 \
  --default

# 4. Dry Run 预览
./bin/aisw start opencode opencode-minimax --dry-run

# 5. 实际启动
./bin/aisw start opencode opencode-minimax
```

### 启动原理

OpenCode 使用 `openai_env` Adapter，设置环境变量：
```bash
OPENAI_API_KEY=<api-key>
OPENAI_BASE_URL=https://api.minimax.chat/v1
```

启动命令：
```bash
OPENAI_API_KEY=<api-key> OPENAI_BASE_URL=https://api.minimax.chat/v1 opencode
```

---

## 项目级配置（.aiswrc）

为不同项目绑定不同 Agent/Profile：

```bash
# 在项目目录创建 .aiswrc
cd ~/work-project
./bin/aisw init --profile claude-minimax

# 文件内容
# profile = "claude-minimax"

# 进入目录直接启动
cd ~/work-project
./bin/aisw start claude
# 自动使用 claude-minimax
```

---

## 测试协议支持矩阵

| Provider 协议 | `test provider` | `test models` | 支持的 Agent |
|--------------|-----------------|---------------|-------------|
| `anthropic` | ✅ 发送 `/v1/messages` | ✅ 发送 `/v1/models` | Claude Code |
| `openai_chat` | ✅ 发送 `/chat/completions` | ✅ 发送 `/models` | Codex, OpenCode, Kimi, Trae |
| `openai_responses` | ✅ 发送 `/responses` | ✅ 发送 `/models` | Codex |
| `gemini_native` | ❌ 未实现 | ❌ 未实现 | Gemini CLI |

---

## 故障排查

### 测试返回 401 Unauthorized
- 检查 API Key 是否正确设置
- 确认环境变量名与 `--api-key-env` 一致

### 测试返回 404 Not Found
- 检查 `base_url` 是否正确
- 确认 endpoint override 路径正确

### Agent 启动失败
- 先运行 `--dry-run` 查看生成的配置
- 检查 Agent binary 是否在 PATH 中：`which claude`、`which codex`、`which opencode`

### 模型不支持
- 使用 `--model` 覆盖默认模型
- 运行 `test models` 查看 Provider 支持的模型列表

---

*最后更新：2025-06-07*
