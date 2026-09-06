---
title: "aisw provider"
description: "Manage vendor LLM providers: list, add, from-preset, model, delete."
---

# aisw provider

管理厂商级 LLM Provider（LLM Provider Config）。**一个厂商一行数据、一把 API key**：`models` 是该厂商的模型列表（新增模型自动共享同一把 key），`variants` 按协议（`anthropic` / `openai_responses` / `openai_chat`）存放各端点，claude code / codex / opencode 各自解析到所需协议的端点，无需重复配置。

```bash
aisw provider <subcommand>
```

## aisw provider list

列出当前已保存的 Provider，并附带内置预设（presets）。

```bash
aisw provider list
```

输出分两节：

- `# Saved providers`：来自本地 SQLite 的 Provider，字段以 tab 分隔：`slug  name  protocols  models  base_url  key=set|missing`（厂商行显示所有协议和全部模型）。
- `# Built-in presets`：打包进二进制的厂商预设（`glm`、`minimax`），含模型列表与每个协议的 base URL。

## aisw provider from-preset PRESET_SLUG

从内置预设一步创建厂商 Provider —— 只需一把 API key 即可同时支持 claude code、codex、opencode。

```bash
aisw provider from-preset glm --api-key sk-xxx
# 或从环境变量读取 key
aisw provider from-preset minimax --api-key-env MINIMAX_API_KEY
# 追加预设之外的模型
aisw provider from-preset glm --api-key sk-xxx --models glm-air-1
```

| Flag | 说明 |
|------|------|
| `--api-key` | API key（一把 key 服务所有模型和 Agent） |
| `--api-key-env` | 从指定环境变量读取 API key |
| `--models` | 额外模型（逗号分隔），追加到预设模型列表 |

## aisw provider preset

管理可复用的厂商预设。自定义预设保存为 TOML 文件（`~/.innate-aiswitcher/presets/*.toml`，可用 `AISW_PRESETS_DIR` 覆盖），自动出现在 `provider list`、`from-preset` 和 Web UI 的预设列表中；**文件中不含 API Key**，可跨机器分享。

```bash
# 把已配置的 Provider 存为预设文件（Key 排除）
aisw provider preset save volcengine-claude

# 从 TOML 文件导入预设（支持一个文件多个 [[presets]] 块）
aisw provider preset import my-presets.toml

# 列出全部预设（内置 + 自定义）及来源
aisw provider preset list

# 删除自定义预设（内置预设不可删）
aisw provider preset delete volcengine-claude
```

Web UI 中：Provider 卡片的「存为预设」按钮、预设对话框的「导入 TOML 文件…」上传入口与自定义预设的删除按钮对应同样的能力。

## aisw provider model

管理 Provider 的模型列表。列表里的模型全部共享该 Provider 已保存的 API key —— **加模型不需要再配置 key**。

```bash
# 查看已配置的模型（默认模型带标记）
aisw provider model list glm

# 添加 GLM 5.3，与 GLM 5.2 共享同一把 key
aisw provider model add glm glm-5.3

# 添加并把新模型设为默认
aisw provider model add glm glm-5.3 --default

# 移除模型（若移除的是默认模型，默认切换为列表中第一个）
aisw provider model remove glm glm-5.2
```

## aisw provider add SLUG

手动新增或更新一个单协议 Provider（厂商行建议优先用 `from-preset`）。按 slug upsert。

```bash
aisw provider add local-mock \
  --base-url http://127.0.0.1:18990/v1 \
  --api-key sk-xxx \
  --protocol openai_chat \
  --model test-model \
  --models test-model,test-model-pro
```

| Flag | 说明 |
|------|------|
| `--name` | 显示名（默认同 slug） |
| `--base-url` *(必填)* | Provider base URL |
| `--api-key` | API key（明文） |
| `--api-key-env` | 从指定环境变量读取 API key |
| `--protocol` | `openai_chat`(默认) / `anthropic` / `openai_responses` |
| `--model` | 默认模型 |
| `--models` | 模型列表（逗号分隔），共享同一把 key |
| `--endpoint` | endpoint override，`key=path` 或 `key=https://host/path`，可多次指定 |
| `--notes` | 备注 |

说明：

- 默认模型必须由 `--model` 提供（未给时取 `--models` 第一个），adapter 不会按协议猜测；缺失时 `start`/`test` 会报错。
- Provider 的 `models` 列表非空时，`start --model` 只能选用列表中的模型，未配置会报错并提示 `aisw provider model add`。

## aisw provider delete SLUG

按 slug 删除一个 Provider。`profiles` 中引用该 Provider 的记录会因 `CascadeDelete` 一并被删除。

```bash
aisw provider delete glm
```

## 相关

- [aisw profile](/commands/profile) - 把 Provider 绑定到某个 Agent 并加 per-agent 覆盖
- [aisw start](/commands/start) - 用 Provider/Profile 启动 Agent session（支持 `--model` 切换模型）
- [aisw test](/commands/test) - 测试 Provider 连通性与列出模型
- [REST API](/API) - `/api/aisw/providers*`、`/api/aisw/presets`、`POST /api/aisw/providers/from-preset`、`POST /api/aisw/providers/{slug}/models`
