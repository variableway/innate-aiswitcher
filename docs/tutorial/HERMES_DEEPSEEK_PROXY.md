# Hermes + DeepSeek 本地代理操作指南

> 使用 Hermes Agent 作为本地代理，通过 DeepSeek 模型调度其他 AI Agent 进行本地操作，避免直接暴露原始 Provider 被封。

---

## 目录

1. [为什么需要本地代理](#1-为什么需要本地代理)
2. [核心原理](#2-核心原理)
3. [环境准备](#3-环境准备)
4. [配置 Hermes + DeepSeek 作为代理](#4-配置-hermes--deepseek-作为代理)
5. [通过代理调用其他 Agent](#5-通过代理调用其他-agent)
6. [防封策略详解](#6-防封策略详解)
7. [一键配置脚本](#7-一键配置脚本)
8. [故障排查](#8-故障排查)

---

## 1. 为什么需要本地代理

### 风险场景

| 场景 | 风险 |
|------|------|
| 直接调用 OpenAI API | IP 被封、账号被限流 |
| 直接调用 Claude API | 地区限制、请求被拒绝 |
| 频繁切换 Provider | 触发风控、API Key 被吊销 |
| 多 Agent 共享同一 Key | 用量异常、被封号 |

### 本地代理的优势

```
┌─────────────────────────────────────────────────────────┐
│  用户                                                    │
│    │                                                    │
│    ▼                                                    │
│  ┌──────────────┐    ┌──────────────┐                  │
│  │ Hermes Agent │───▶│ DeepSeek     │  ← 唯一对外出口  │
│  │  (本地代理)  │    │ (中转层)     │                  │
│  └──────────────┘    └──────────────┘                  │
│         │                                              │
│    ┌────┴────┬────────┬────────┐                      │
│    ▼         ▼        ▼        ▼                      │
│  Claude    Codex    OpenCode   Kimi  ← 本地 Agent      │
│  (本地)    (本地)   (本地)    (本地)   不直接对外      │
└─────────────────────────────────────────────────────────┘
```

**优势**:
- **单一出口**: 只有 DeepSeek 对外发起请求，其他 Agent 完全本地
- **流量混淆**: DeepSeek 的 OpenAI-compatible 协议与正常请求无异
- **IP 隔离**: 本地 Agent 不直接访问外部 API，避免 IP 被标记
- **Key 隔离**: 每个 Agent 使用独立的本地 Profile，不共享原始 Key

---

## 2. 核心原理

### 2.1 双层代理架构

```
Layer 1: Hermes Agent (调度层)
  └── 使用 DeepSeek 模型进行任务理解和调度
  └── 生成调用其他 Agent 的指令

Layer 2: innate-aiswitcher (适配层)
  └── 将 DeepSeek 的调度指令转换为具体 Agent 的启动命令
  └── 为每个 Agent 创建独立的临时配置

Layer 3: 本地 Agent (执行层)
  └── Claude Code: 临时 settings.json
  └── Codex CLI: 临时 CODEX_HOME
  └── OpenCode: 临时环境变量
  └── Kimi: 临时环境变量
```

### 2.2 数据流

```
用户输入 → Hermes (DeepSeek) → 任务分解 → innate-aiswitcher → 启动本地 Agent → 本地执行 → 结果返回
```

**关键点**: 本地 Agent 的 API 调用通过 innate-aiswitcher 的临时配置完成，不暴露用户的真实 Provider 配置。

---

## 3. 环境准备

### 3.1 安装依赖

```bash
#!/bin/bash
# setup-local-proxy.sh
# 一键配置本地代理环境

set -e

echo "=== 本地代理环境配置 ==="

# 1. 安装 innate-aiswitcher
echo "[1/4] 安装 innate-aiswitcher..."
cd /path/to/innate-aiswitcher
task build || go build -o bin/aisw ./cmd/aisw

# 2. 安装 Hermes Agent
echo "[2/4] 安装 Hermes Agent..."
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
source ~/.bashrc

# 3. 安装其他本地 Agent
echo "[3/4] 安装本地 Agent..."
npm install -g @anthropic-ai/claude-code 2>/dev/null || true
npm install -g @openai/codex 2>/dev/null || true
curl -fsSL https://get.opencode.ai | bash 2>/dev/null || true

# 4. 设置环境变量
echo "[4/4] 设置环境变量..."
cat >> ~/.bashrc << 'EOF'

# innate-aiswitcher 本地代理配置
export DEEPSEEK_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
export PATH="$PATH:/path/to/innate-aiswitcher/bin"
EOF

echo "=== 配置完成 ==="
echo "请编辑 ~/.bashrc 填入真实的 API Key"
echo "然后运行: source ~/.bashrc"
```

### 3.2 环境变量模板

```bash
# ~/.bashrc 或 ~/.zshrc

# DeepSeek 代理层（唯一对外）
export DEEPSEEK_API_KEY="sk-your-deepseek-key"

# 本地 Agent 的原始 Key（不直接使用，通过 aisw 代理）
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
export MINIMAX_API_KEY="sk-..."
export KIMI_API_KEY="sk-..."

# innate-aiswitcher 路径
export PATH="$PATH:$HOME/innate-aiswitcher/bin"
```

---

## 4. 配置 Hermes + DeepSeek 作为代理

### 4.1 添加 DeepSeek Provider

```bash
# 添加 DeepSeek 作为代理层 Provider
aisw provider add deepseek \
  --base-url https://api.deepseek.com/v1 \
  --api-key-env DEEPSEEK_API_KEY \
  --protocol openai_chat \
  --model deepseek-chat \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models
```

### 4.2 添加本地 Agent 的 Provider（用于临时配置）

```bash
# MiniMax OpenAI-compatible（供 Codex/OpenCode/Hermes 使用）
aisw provider add minimax-openai \
  --base-url https://api.minimax.chat/v1 \
  --api-key-env MINIMAX_API_KEY \
  --protocol openai_chat \
  --model MiniMax-M1 \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# MiniMax Claude-compatible（供 Claude Code 使用）
aisw provider add minimax-claude \
  --base-url https://api.minimax.chat/anthropic \
  --api-key-env MINIMAX_API_KEY \
  --protocol anthropic \
  --model claude-3-5-sonnet-20241022 \
  --endpoint messages=/v1/messages \
  --endpoint models=/v1/models

# Kimi OpenAI-compatible（供 Codex/OpenCode/Hermes 使用）
aisw provider add kimi-openai \
  --base-url https://api.moonshot.cn/v1 \
  --api-key-env KIMI_API_KEY \
  --protocol openai_chat \
  --model moonshot-v1-8k \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# Kimi Claude-compatible（供 Claude Code 使用）
aisw provider add kimi-claude \
  --base-url https://api.moonshot.cn/anthropic \
  --api-key-env KIMI_API_KEY \
  --protocol anthropic \
  --model claude-3-5-sonnet-20241022 \
  --endpoint messages=/v1/messages \
  --endpoint models=/v1/models

# Xiaomi MiMo（供 Codex/OpenCode/Hermes 使用）
aisw provider add xiaomi-openai \
  --base-url https://api.mimo.xiaomi.com/v1 \
  --api-key-env XIAOMI_API_KEY \
  --protocol openai_chat \
  --model mimo-7b \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models
```

### 4.3 创建 Hermes 代理 Profile

```bash
# Hermes + DeepSeek（主代理）
aisw profile add hermes-deepseek \
  --agent hermes \
  --provider deepseek \
  --model deepseek-chat

# Hermes + MiniMax（备用代理）
aisw profile add hermes-minimax \
  --agent hermes \
  --provider minimax-openai \
  --model MiniMax-M1

# Hermes + Kimi（备用代理）
aisw profile add hermes-kimi \
  --agent hermes \
  --provider kimi-openai \
  --model moonshot-v1-8k

# Hermes + Xiaomi（备用代理）
aisw profile add hermes-xiaomi \
  --agent hermes \
  --provider xiaomi-openai \
  --model mimo-7b
```

### 4.4 创建本地 Agent 的 Profile

```bash
# Claude Code + MiniMax
aisw profile add claude-minimax \
  --agent claude \
  --provider minimax-claude \
  --model claude-3-5-sonnet-20241022

# Claude Code + Kimi
aisw profile add claude-kimi \
  --agent claude \
  --provider kimi-claude \
  --model claude-3-5-sonnet-20241022

# Codex + MiniMax
aisw profile add codex-minimax \
  --agent codex \
  --provider minimax-openai \
  --model MiniMax-M1

# Codex + Kimi
aisw profile add codex-kimi \
  --agent codex \
  --provider kimi-openai \
  --model moonshot-v1-8k

# Codex + DeepSeek
aisw profile add codex-deepseek \
  --agent codex \
  --provider deepseek \
  --model deepseek-chat

# OpenCode + MiniMax
aisw profile add opencode-minimax \
  --agent opencode \
  --provider minimax-openai \
  --model MiniMax-M1

# OpenCode + DeepSeek
aisw profile add opencode-deepseek \
  --agent opencode \
  --provider deepseek \
  --model deepseek-chat

# Kimi CLI + Moonshot
aisw profile add kimi-default \
  --agent kimi \
  --provider kimi-openai \
  --model moonshot-v1-8k
```

---

## 5. 通过代理调用其他 Agent

### 5.1 手动调度模式

```bash
# 1. 启动 Hermes 代理
aisw start hermes hermes-deepseek

# 2. 在 Hermes 中输入任务，例如：
# "帮我用 Claude Code 分析这个项目的代码结构"

# 3. Hermes 会生成调度指令，用户手动执行：
aisw start claude claude-minimax --cwd /path/to/project

# 4. 或者在 Hermes 中输入：
# "用 Codex 帮我写一段 Python 爬虫"

# 5. 手动启动 Codex：
aisw start codex codex-minimax
```

### 5.2 自动调度模式（通过 Hermes Skills）

```bash
# 在 Hermes 中启用本地调度 skill
# 创建 ~/.hermes/skills/local-agent-dispatch/SKILL.md
```

**SKILL.md 内容**:

```markdown
# Local Agent Dispatch

## Description
Dispatch tasks to local AI agents through innate-aiswitcher.

## Procedure

1. Analyze the user's task to determine the best agent:
   - Code analysis → Claude Code
   - Code generation → Codex CLI
   - Quick edits → OpenCode
   - Chinese context → Kimi CLI

2. Generate the aisw command:
   ```bash
   aisw start <agent> <profile> --cwd <path>
   ```

3. Execute the command in a subprocess.

4. Return the results to the user.

## Examples

- "分析代码" → `aisw start claude claude-minimax --cwd .`
- "写代码" → `aisw start codex codex-minimax`
- "快速编辑" → `aisw start opencode opencode-minimax`
```

### 5.3 批量调度脚本

```bash
#!/bin/bash
# dispatch-agent.sh
# 根据任务类型自动调度本地 Agent

TASK="$1"
CWD="${2:-.}"

case "$TASK" in
  "analyze"|"review"|"audit")
    echo "调度 Claude Code 进行代码分析..."
    aisw start claude claude-minimax --cwd "$CWD"
    ;;
  "generate"|"write"|"create")
    echo "调度 Codex CLI 进行代码生成..."
    aisw start codex codex-minimax --cwd "$CWD"
    ;;
  "edit"|"fix"|"patch")
    echo "调度 OpenCode 进行快速编辑..."
    aisw start opencode opencode-minimax --cwd "$CWD"
    ;;
  "chinese"|"cn"|"moonshot")
    echo "调度 Kimi CLI 处理中文任务..."
    aisw start kimi kimi-default --cwd "$CWD"
    ;;
  *)
    echo "未知任务类型: $TASK"
    echo "可用类型: analyze, generate, edit, chinese"
    exit 1
    ;;
esac
```

**使用**:

```bash
chmod +x dispatch-agent.sh
./dispatch-agent.sh analyze /path/to/project
./dispatch-agent.sh generate /path/to/project
./dispatch-agent.sh edit /path/to/project
```

---

## 6. 防封策略详解

### 6.1 策略一：单一出口

```bash
# 只让 DeepSeek 对外，其他 Agent 完全本地
# 配置防火墙规则（可选）

# macOS: 阻止其他 Agent 直接访问外部 API
sudo pfctl -e
sudo cat > /etc/pf.conf << 'EOF'
# 只允许 DeepSeek 出口
pass out on en0 inet proto tcp to api.deepseek.com port 443
block out on en0 inet proto tcp to any port 443
EOF
sudo pfctl -f /etc/pf.conf
```

> ⚠️ 注意：防火墙规则需谨慎配置，避免影响正常网络使用。

### 6.2 策略二：临时配置隔离

```bash
# innate-aiswitcher 自动为每个 Agent 创建临时配置
# 每次启动都是全新的临时目录

# 查看临时配置示例（dry-run）
aisw start claude claude-minimax --dry-run

# 输出示例：
# {
#   "command": "claude --settings /tmp/aisw-claude-xxx.json",
#   "env": {},
#   "files": {
#     "settings": "/tmp/aisw-claude-xxx.json"
#   }
# }

# 临时文件在 Agent 退出后自动清理
```

### 6.3 策略三：Key 轮换

```bash
#!/bin/bash
# rotate-keys.sh
# 定期轮换 API Key（如果支持多个 Key）

# DeepSeek 支持多个 Key 轮换
export DEEPSEEK_API_KEY_1="sk-..."
export DEEPSEEK_API_KEY_2="sk-..."
export DEEPSEEK_API_KEY_3="sk-..."

# 随机选择 Key
KEYS=("$DEEPSEEK_API_KEY_1" "$DEEPSEEK_API_KEY_2" "$DEEPSEEK_API_KEY_3")
RANDOM_KEY="${KEYS[$RANDOM % ${#KEYS[@]}]}"
export DEEPSEEK_API_KEY="$RANDOM_KEY"

# 更新 aisw 配置
aisw provider add deepseek \
  --base-url https://api.deepseek.com/v1 \
  --api-key-env DEEPSEEK_API_KEY \
  --protocol openai_chat \
  --model deepseek-chat
```

### 6.4 策略四：请求频率控制

```bash
# 在 Hermes 配置中限制请求频率
# ~/.hermes/config.yaml

agent:
  max_turns: 50  # 限制单会话轮数

# 使用 sleep 控制批量请求
for task in task1 task2 task3; do
  ./dispatch-agent.sh "$task"
  sleep 5  # 间隔 5 秒
done
```

### 6.5 策略五：本地模型兜底

```bash
# 当外部 API 不可用时，切换到本地模型
# 使用 Ollama 或 LM Studio 作为兜底

# 安装 Ollama
curl -fsSL https://ollama.com/install.sh | bash

# 拉取本地模型
ollama pull qwen2.5:7b

# 添加本地 Provider
aisw provider add local-ollama \
  --base-url http://localhost:11434/v1 \
  --api-key dummy \
  --protocol openai_chat \
  --model qwen2.5:7b

# 创建兜底 Profile
aisw profile add hermes-local \
  --agent hermes \
  --provider local-ollama \
  --model qwen2.5:7b
```

---

## 7. 一键配置脚本

### 7.1 完整一键配置

```bash
#!/bin/bash
# setup-hermes-proxy.sh
# 一键配置 Hermes + DeepSeek 本地代理环境

set -e

echo "=== Hermes + DeepSeek 本地代理配置 ==="

# 检查环境变量
if [ -z "$DEEPSEEK_API_KEY" ]; then
  echo "错误: 请设置 DEEPSEEK_API_KEY"
  exit 1
fi

# 1. 添加 DeepSeek Provider
echo "[1/6] 添加 DeepSeek Provider..."
aisw provider add deepseek \
  --base-url https://api.deepseek.com/v1 \
  --api-key-env DEEPSEEK_API_KEY \
  --protocol openai_chat \
  --model deepseek-chat \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models

# 2. 添加 MiniMax Provider（如果配置了 Key）
if [ -n "$MINIMAX_API_KEY" ]; then
  echo "[2/6] 添加 MiniMax Provider..."
  aisw provider add minimax-openai \
    --base-url https://api.minimax.chat/v1 \
    --api-key-env MINIMAX_API_KEY \
    --protocol openai_chat \
    --model MiniMax-M1 \
    --endpoint chat_completions=/chat/completions \
    --endpoint models=/models

  aisw provider add minimax-claude \
    --base-url https://api.minimax.chat/anthropic \
    --api-key-env MINIMAX_API_KEY \
    --protocol anthropic \
    --model claude-3-5-sonnet-20241022 \
    --endpoint messages=/v1/messages \
    --endpoint models=/v1/models
fi

# 3. 添加 Kimi Provider（如果配置了 Key）
if [ -n "$KIMI_API_KEY" ]; then
  echo "[3/6] 添加 Kimi Provider..."
  aisw provider add kimi-openai \
    --base-url https://api.moonshot.cn/v1 \
    --api-key-env KIMI_API_KEY \
    --protocol openai_chat \
    --model moonshot-v1-8k \
    --endpoint chat_completions=/chat/completions \
    --endpoint models=/models

  aisw provider add kimi-claude \
    --base-url https://api.moonshot.cn/anthropic \
    --api-key-env KIMI_API_KEY \
    --protocol anthropic \
    --model claude-3-5-sonnet-20241022 \
    --endpoint messages=/v1/messages \
    --endpoint models=/v1/models
fi

# 4. 添加 Xiaomi Provider（如果配置了 Key）
if [ -n "$XIAOMI_API_KEY" ]; then
  echo "[4/6] 添加 Xiaomi Provider..."
  aisw provider add xiaomi-openai \
    --base-url https://api.mimo.xiaomi.com/v1 \
    --api-key-env XIAOMI_API_KEY \
    --protocol openai_chat \
    --model mimo-7b \
    --endpoint chat_completions=/chat/completions \
    --endpoint models=/models
fi

# 5. 创建 Hermes 代理 Profile
echo "[5/6] 创建 Hermes 代理 Profile..."
aisw profile add hermes-deepseek \
  --agent hermes \
  --provider deepseek \
  --model deepseek-chat

# 6. 创建本地 Agent Profile
echo "[6/6] 创建本地 Agent Profile..."
[ -n "$MINIMAX_API_KEY" ] && aisw profile add claude-minimax \
  --agent claude --provider minimax-claude --model claude-3-5-sonnet-20241022

[ -n "$MINIMAX_API_KEY" ] && aisw profile add codex-minimax \
  --agent codex --provider minimax-openai --model MiniMax-M1

[ -n "$KIMI_API_KEY" ] && aisw profile add claude-kimi \
  --agent claude --provider kimi-claude --model claude-3-5-sonnet-20241022

[ -n "$KIMI_API_KEY" ] && aisw profile add codex-kimi \
  --agent codex --provider kimi-openai --model moonshot-v1-8k

[ -n "$XIAOMI_API_KEY" ] && aisw profile add hermes-xiaomi \
  --agent hermes --provider xiaomi-openai --model mimo-7b

echo ""
echo "=== 配置完成 ==="
echo ""
echo "启动代理: aisw start hermes hermes-deepseek"
echo ""
echo "可用 Profile:"
aisw profile list
```

### 7.2 使用方法

```bash
# 1. 设置环境变量
export DEEPSEEK_API_KEY="sk-..."
export MINIMAX_API_KEY="sk-..."
export KIMI_API_KEY="sk-..."
export XIAOMI_API_KEY="sk-..."

# 2. 运行一键配置
chmod +x setup-hermes-proxy.sh
./setup-hermes-proxy.sh

# 3. 启动 Hermes 代理
aisw start hermes hermes-deepseek

# 4. 在 Hermes 中调度其他 Agent
# 输入: "用 Claude 分析代码"
# 然后手动执行: aisw start claude claude-minimax
```

---

## 8. 故障排查

### 8.1 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| DeepSeek 返回 401 | API Key 无效 | 检查 `DEEPSEEK_API_KEY` |
| Hermes 无法启动 | `HERMES_HOME` 冲突 | 确认无其他 Hermes 实例运行 |
| Claude 启动失败 | settings.json 格式错误 | 检查 `--dry-run` 输出 |
| Codex 启动失败 | `CODEX_HOME` 权限问题 | 检查临时目录权限 |
| 本地 Agent 无法访问 API | 网络隔离 | 检查防火墙规则 |

### 8.2 调试命令

```bash
# 查看所有 Provider
aisw provider list

# 查看所有 Profile
aisw profile list

# 测试 DeepSeek 连通性
aisw test provider deepseek

# Dry-run 查看启动计划
aisw start hermes hermes-deepseek --dry-run
aisw start claude claude-minimax --dry-run
aisw start codex codex-minimax --dry-run

# 查看临时文件（启动后）
ls /tmp/aisw-*

# 查看 Hermes 日志
hermes doctor
cat ~/.hermes/logs/agent.log
```

### 8.3 恢复默认配置

```bash
# 重置所有配置
rm -rf ~/.innate-aiswitcher/pb_data
rm -f ~/.innate-aiswitcher/config.toml

# 重新初始化
aisw config template --path ~/.innate-aiswitcher/config.toml
aisw config import --path ~/.innate-aiswitcher/config.toml --no-backup
```

---

## 附录：完整 Profile 矩阵

| Agent \ Provider | DeepSeek | MiniMax | Kimi | Xiaomi | OpenAI |
|-----------------|----------|---------|------|--------|--------|
| **Hermes** | ✅ hermes-deepseek | ✅ hermes-minimax | ✅ hermes-kimi | ✅ hermes-xiaomi | ✅ hermes-openai |
| **Claude Code** | ❌ 不支持 | ✅ claude-minimax | ✅ claude-kimi | ❌ 不支持 | ✅ claude-openai |
| **Codex CLI** | ✅ codex-deepseek | ✅ codex-minimax | ✅ codex-kimi | ❌ 不支持 | ✅ codex-openai |
| **OpenCode** | ✅ opencode-deepseek | ✅ opencode-minimax | ❌ 未配置 | ❌ 不支持 | ✅ opencode-openai |
| **Kimi CLI** | ❌ 不支持 | ❌ 不支持 | ✅ kimi-default | ❌ 不支持 | ❌ 不支持 |
| **Gemini CLI** | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 |
| **Trae CLI** | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 |

> 注：✅ 表示已配置，❌ 表示协议不兼容或未配置。Claude Code 需要 `anthropic` 协议，Codex/OpenCode/Hermes 需要 `openai_chat` 协议。

---

*本指南对应 innate-aiswitcher 当前代码版本。如有更新，请以官方文档为准。*
