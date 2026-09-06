---
title: "aisw profile"
description: "Manage optional agent+provider profiles with per-agent overrides."
---

# aisw profile

管理 Agent Profile。Profile 是**可选**的 Agent + Provider 绑定，只承载 per-agent 覆盖项（模型、启动参数、env/config overrides、是否默认）。它不复制 Provider。

```bash
aisw profile <subcommand>
```

## aisw profile list

列出所有 Profile，字段以 tab 分隔：`slug  agent  provider  model`。

```bash
aisw profile list
```

## aisw profile add SLUG

新增或更新一个 Profile。按 slug upsert；`--agent` 与 `--provider` 必填。

```bash
aisw profile add claude-glm \
  --name "Claude + GLM" \
  --agent claude \
  --provider glm \
  --default
```

```bash
aisw profile add codex-glm53 \
  --agent codex \
  --provider glm \
  --model glm-5.3
```

| Flag | 说明 |
|------|------|
| `--name` | 显示名（默认同 slug） |
| `--agent` *(必填)* | Agent slug，如 `claude`/`codex`/`opencode` |
| `--provider` *(必填)* | Provider slug |
| `--model` | 模型覆盖（留空则用 Provider 的 `default_model`） |
| `--args` | 默认 native 参数 |
| `--skip-permissions` | 覆盖 Agent 的 skip-permissions 默认：`true` / `false`（留空用 Agent 默认） |
| `--default` | 标记为该 Agent 的默认 Profile |

`--default` 会把该 Agent + Provider 的默认配置持久化到 Agent 期望的位置，使 `aisw start AGENT`（不带 selector）可直接命中。

## Profile 应保持最小化（设计建议）

> 这是 Task 1 对「重复 profile」问题的检查结论与建议。

检查 `internal/templates/files/config.example.toml` 时发现两类重复，已修复：

1. **Profile 复述 Provider 的默认模型**：模板里每个 profile 都把 `model` 又写了一遍，而 adapter 在 profile 没给 `model` 时本就会回退到 Provider 的 `default_model`（见 `internal/adapter/adapter.go` 的 `BuildPlan`）。已删除这些冗余 `model` 字段。
2. **Profile slug 与 Provider slug 冲突**：Profile slug 不能与 Provider slug 相同，否则 `start` 的 selector 解析会优先命中 Profile，容易混淆。

建议（写配置/用 CLI 时遵循）：

- **Profile 是可选的**。`aisw start AGENT PROVIDER` 可直接用 Provider 启动，不必先建 Profile。只有需要 per-agent 覆盖（不同 Agent 用不同模型/参数、或想设默认）时才建 Profile。
- **只放覆盖项**。Profile 不要重复 Provider 已有的 `default_model`/`base_url`/`api_key`；只写 `model`/`args`/`env_overrides`/`config_overrides`/`is_default`。
- **slug 不要与 Provider 冲突**。建议用 `<agent>-<vendor>[-<model>]`（如 `codex-glm`、`codex-glm53`），避免和 Provider slug 撞名。

这与项目目标一致：Provider 配好后，对应 Agent 可直接 `aisw start AGENT PROVIDER` 使用；Profile 仅在需要差异化时介入，避免为每个 (agent, provider) 组合都建一条重复记录。

## 相关

- [aisw provider](/commands/provider) - 管理共享 Provider
- [aisw start](/commands/start) - 解析 selector（Profile 或 Provider）启动 session
- [REST API](/API) - `/api/aisw/profiles*`
