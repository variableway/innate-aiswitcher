---
title: "aisw start"
description: "Start an agent with a session-only provider or profile."
---

# aisw start

用本次选择的 Provider 或 Profile 启动一个 Agent session。Provider/Profile 解析只在启动时投影成临时配置（settings JSON、`CODEX_HOME`、env vars 等），不写回共享数据库。厂商 Provider（如 `glm`、`minimax`）会按 Agent 自动选择协议端点：`claude` → anthropic，`codex` → openai_responses（无则回落 openai_chat），`opencode` → openai_chat。

```bash
aisw start AGENT [PROVIDER_OR_PROFILE] [--model M] -- [native args]
```

- `AGENT` *(必填)*：Agent slug：`claude`、`codex`、`opencode`。
- `PROVIDER_OR_PROFILE` *(可选)*：selector。先按 slug 查 Profile，找不到再查 Provider。
- `--model` *(可选)*：本次启动的模型覆盖，优先级最高（高于 Profile 的 `model` 与 Provider 的 `default_model`）。Provider 配置了 `models` 列表时，模型必须在列表内。

## selector 解析

`store.ResolveSelector(agentSlug, selector)` 顺序：

1. 若给了 selector：先匹配 Profile slug（必须属于该 Agent），再匹配 Provider slug。
2. 若没给 selector：读取 `.aiswrc`（见下），否则取该 Agent 的默认 Profile（`is_default=true`）。
3. 都没有则报错。

模型解析顺序：`--model` > Profile `model` > Provider `default_model`。

## .aiswrc 项目级默认

未给 selector 时，`start` 会从 `$PWD` 向上查找 `.aiswrc`（TOML，可选 `profile` / `agent` / `provider`）：

```toml
# .aiswrc
profile = "codex-glm53"
agent = "codex"
```

- 命中 `profile` 或 `provider` 即作为本次 selector。
- 若 `.aiswrc` 指定了 `agent` 且与命令行的 `AGENT` 冲突，会拒绝启动。
- `--ignore-project` 跳过 `.aiswrc`。

> 注：`aisw init` 命令已移除。`.aiswrc` 需手动创建（参考 [aisw config](/commands/config) 的 `config template` 导出模板，或直接手写）。

## 用法示例

```bash
# 用厂商 Provider 启动 claude（无需为每个 Agent 重复配置）
aisw start claude glm

# 切换模型：GLM 5.3 与 5.2 共享同一把 key
aisw start claude glm --model glm-5.3

# 用 Profile 启动
aisw start codex codex-glm53

# 只给 Agent，走 .aiswrc 或默认 Profile
aisw start codex

# 跳过 .aiswrc
aisw start codex --ignore-project

# 只打印 launch plan，不真正启动
aisw start claude glm --dry-run

# 透传 native 参数
aisw start codex glm -- -c approval_policy=never
```

## Flags

| Flag | 说明 |
|------|------|
| `--model` | 模型覆盖（须在 Provider 的 `models` 列表内，未配置会提示 `aisw provider model add`） |
| `--dry-run` | 只打印 launch plan（JSON），不启动 Agent |
| `--terminal` | `current`(默认) / `ghostty` / `terminal`（macOS 下可经 Ghostty/Terminal.app 拉起） |
| `--cwd` | 工作目录（默认当前目录） |
| `--ignore-project` | 忽略 `.aiswrc` |

## 启动后

`start` 会写一条 `launch_history` 记录（agent、provider、profile、cwd、command、terminal、status）。`--dry-run` 不实际启动，但仍打印 plan。

## 相关

- [aisw provider](/commands/provider) / [aisw profile](/commands/profile) - selector 的来源
- [aisw test](/commands/test) - 启动前先验证 Provider
