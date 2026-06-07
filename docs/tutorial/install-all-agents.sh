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
