---
title: "Usage Guide"
description: "AISwitcher 日常使用：厂商 Provider、模型管理、启动 Agent、Web 与终端。"
---

# AISwitcher 使用指南

> 在线文档：[variableway.github.io/innate-aiswitcher](https://variableway.github.io/innate-aiswitcher/)

## 核心概念

- **厂商 Provider（LLM Provider Config）**：一个厂商（`glm`、`minimax`）一行数据、**一把 API Key**。`models` 是模型列表（全部共享 key），`variants` 按协议（`anthropic` / `openai_responses` / `openai_chat`）存放端点。
- **Agent**：`claude`（Claude Code）、`codex`（Codex CLI）、`opencode`。启动时各自解析所需协议端点——`claude → anthropic`，`codex → openai_responses`（无则回落 openai_chat），`opencode → openai_chat`。
- **Profile**（可选）：Agent↔Provider 绑定，只放 per-agent 覆盖（模型、参数、默认标记）。

## 快速开始

```bash
task build            # 构建前端 + Go 单体二进制（bin/aisw 内嵌 Web 应用）
aisw web               # 启动并打开 Web 页面（Providers / Profiles / Terminal）
```

一条命令配置 GLM 并启动 Claude Code：

```bash
aisw provider from-preset glm --api-key sk-xxx   # 一把 key 服务三个 Agent
aisw start claude glm
```

## 厂商与模型管理

```bash
# 从内置预设导入（预设：glm、minimax）
aisw provider from-preset minimax --api-key-env MINIMAX_API_KEY
aisw provider from-preset glm --api-key sk-xxx --models glm-air-1   # 追加模型

# 查看已保存 Provider + 内置预设
aisw provider list

# 模型管理：加模型 = 共享已有 key，无需再配
aisw provider model add glm glm-5.3            # 与 glm-5.2 共享同一把 key
aisw provider model add glm glm-5.3 --default  # 同时设为默认
aisw provider model list glm
aisw provider model remove glm glm-5.2

# 手动添加单协议 Provider（一般情况建议 from-preset）
aisw provider add local-mock \
  --base-url http://127.0.0.1:18990/v1 \
  --api-key sk-local --protocol openai_chat \
  --model test-model --models test-model,test-pro
```

## 启动 Agent 会话

```bash
aisw start claude glm                      # 厂商 Provider 直接启动
aisw start claude glm --model glm-5.3      # 切换模型（须在 models 列表内）
aisw start codex codex-glm53               # 用 Profile 启动
aisw start codex                           # 走 .aiswrc 或默认 Profile
aisw start claude glm --dry-run            # 只打印启动计划
```

模型优先级：`--model` > Profile `model` > Provider `default_model`。用了未配置的模型会报错并提示 `aisw provider model add`。

## 项目级默认（.aiswrc）

在项目根手动创建（TOML）：

```toml
# .aiswrc
profile = "codex-glm53"
agent = "codex"
```

`aisw start codex` 会自动套用；`.aiswrc` 的 `agent` 与命令行冲突时拒绝启动；`--ignore-project` 跳过。

## 连通性测试

```bash
aisw test provider glm
aisw test provider glm --model glm-5.3
aisw test models glm        # 调用厂商 models 端点
```

## Web 应用与浏览器终端

```bash
aisw web                   # 127.0.0.1:8090，自动打开浏览器
aisw web --no-browser
task web:dev               # 前端开发模式（/api 代理到 127.0.0.1:8090，配合 task serve）
```

- **Providers 页**：厂商卡片、模型徽标增删（共享 key）、From Preset 导入、Test 连通性
- **Profiles 页**：按 Agent 过滤可用 Provider，设置模型覆盖与默认
- **Terminal 页**：浏览器里的本地 PTY 终端，多 tab 独立会话；`Launch agent` 一键 `aisw start <agent> <provider>`

> ⚠️ Terminal 会话即本地 shell。服务默认绑定 127.0.0.1；绑定非回环地址时会打印暴露警告。

## 交互式 TUI

```bash
aisw          # 选择 Agent → 过滤出支持它的 Provider → 选择模型 → 启动
```

## 配置导入导出

```bash
aisw config template --path ~/.innate-aiswitcher/config.toml   # 写出模板（厂商格式示例）
aisw config import  --path config.toml                         # 导入（默认先备份）
aisw config export  --path config.toml --include-secrets       # 导出（含 key）
aisw config dump                                               # 全量写到 init-config 路径
```

配置文件为厂商格式：一个 `[[providers]]` 块含 `models` 与 `[providers.variants.<protocol>]` 子表，参见模板。

## 命令速查

| 命令 | 说明 |
|------|------|
| `aisw web` | 启动 Web 应用（含终端），自动开浏览器 |
| `aisw serve` | 同一服务，不自动开浏览器 |
| `aisw provider list / add / from-preset / model / delete` | 厂商与模型管理 |
| `aisw profile add / list` | Profile 管理 |
| `aisw start AGENT [SELECTOR] [--model M]` | 启动 Agent 会话 |
| `aisw test provider / models SLUG` | 连通性测试 |
| `aisw config template / import / export / dump` | 配置镜像 |

详细参考：[commands](/commands/provider)
