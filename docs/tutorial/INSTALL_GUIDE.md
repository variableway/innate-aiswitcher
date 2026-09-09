---
title: "Install Guide"
description: "Install the focused agents (claude code, codex, opencode) and the aisw switcher."
---

# AI Agent 安装与配置教程

AISwitcher 当前聚焦三个 coding agent：**Claude Code**、**Codex CLI**、**OpenCode**。本文介绍它们的安装方式以及 `aisw` 的安装。

## 1. 安装 AISwitcher

```bash
git clone <repo> && cd innate-aiswitcher
task build        # 构建前端 + Go 单体二进制
task install      # 安装到 ~/.local/bin
```

> `task build` 总是先重新构建 Web 前端，再编译内嵌它的 Go 二进制。

## 2. 安装 Agent（按需）

### Claude Code

```bash
# 方式 1: npm（推荐）
npm install -g @anthropic-ai/claude-code

# 方式 2: Homebrew (macOS)
brew install --cask claude-code

# 验证
claude --version
```

### Codex CLI

```bash
npm install -g @openai/codex

# 验证
codex --version
```

### OpenCode

```bash
# 官方安装脚本（macOS / Linux）
curl -fsSL https://opencode.ai/install | bash

# 或 npm
npm install -g opencode-ai

# 验证
opencode --version
```

## 3. 添加厂商 Provider（一把 API Key）

以 GLM（Volcengine Ark）为例，一条命令即可让三个 Agent 同时可用：

```bash
aisw provider from-preset glm --api-key sk-xxx
# 或从环境变量读取 key
aisw provider from-preset glm --api-key-env GLM_API_KEY
```

新增模型与 GLM-5.2 共享同一把 key，无需重复配置：

```bash
aisw provider model add glm glm-5.3
```

## 4. 启动

```bash
aisw start claude glm                # CLI 启动
aisw start claude glm --model glm-5.3
aisw web                             # 打开 Web 应用（含浏览器终端）
```

Web 页面的 Terminal 标签可开多个本地终端，`Launch agent` 一键运行 `aisw start <agent> <provider>`。

## 5. 验证

```bash
aisw test provider glm               # 连通性测试
aisw start codex glm --dry-run       # 只打印启动计划
```

## 相关

- [Usage Guide](/USAGE) — 完整命令用法
- [aisw provider](/commands/provider) — 厂商与模型管理
- [aisw web](/commands/web) — Web 应用与终端会话
