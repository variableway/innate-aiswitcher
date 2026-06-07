# Goal and Implementation Plan

## Overall Goal

`innate-aiswitcher` 的总体目标是做一个本地 AI Agent Provider switcher：用一个共享 Provider 中心管理不同 LLM 服务，并在每次 terminal session 启动 Claude Code、Codex CLI、Gemini、Trae、OpenCode 等 Agent 时选择要使用的 Provider/Profile。

项目必须解决的核心问题是 Provider 与 Agent Adapter 解耦，避免 cc-switch 当前“每个 App 一个 ProviderManager”的重复配置模型。

## Architecture Review

Provider 与 Agent Adapter 解耦是合理且必要的：

- 同一个 Provider 可能同时服务多个 Agent，不应该因为 Claude、Codex、Trae 的配置格式不同而重复保存多份 base URL/API key。
- Agent Adapter 是运行时投影逻辑，只负责把共享 Provider 转成该 Agent 能消费的临时 settings/config/env。
- Profile 可以保存 Agent 相关覆盖项，但它仍引用 Provider，不复制 Provider。
- 配置文件可以共用，结构应保持为 `[[providers]]` 和 `[[profiles]]`，其中 profile 引用 provider slug。

## MVP Scope Implemented

- PocketBase + SQLite 后台数据存储。
- Collections: `providers`、`agents`、`profiles`、`bindings`、`launch_history`、`settings`。
- Seed agents: Claude Code、Codex CLI、Gemini CLI、Kimi CLI、Trae CLI、OpenCode。
- Provider catalog includes bundled presets with base URL、protocol、default model and endpoint overrides。
- CLI commands: `provider`、`profile`、`start`、`test provider`、`test models`、`config template/import/export`。
- TUI default entry for choosing Agent + Provider, testing Provider connectivity, and starting/dry-running a session。
- Adapter projection:
  - Claude: temporary settings JSON。
  - Codex: temporary `CODEX_HOME` with `config.toml` and `auth.json`。
  - Gemini/OpenAI-compatible tools: session environment variables。
- API Key testing through OpenAI-compatible、OpenAI Responses and Anthropic-compatible minimal requests。
- Basic model listing through provider `models` endpoint overrides/fallbacks。
- Config export/template writes are atomic; config import is backed up by default and applied transactionally。
- REST endpoints for health/catalog/provider test/model listing plus public read access to agent/provider/profile collections。

## Implementation Plan For Next Iterations

1. Add project/global bindings so `start AGENT` can infer a default Provider/Profile from current working directory.
2. Expand adapter fidelity for Trae/OpenCode once their exact CLI config expectations are confirmed.
3. Add authenticated write REST endpoints or PocketBase auth flow for a future UI.
4. Add a web UI over PocketBase REST for Provider/Profile management and launch history.
5. Add secret handling options beyond hidden PocketBase fields, such as keychain integration.
6. Expand the provider preset catalog and model discovery coverage for more providers.

## Validation Strategy

- Unit tests cover provider API request construction, endpoint overrides, model listing, and adapter projection.
- CLI smoke tests cover migration, provider/profile CRUD, dry-run launch plans, config export, API key testing, and model listing with a local mock endpoint.
- REST smoke tests cover custom endpoints and built-in PocketBase collection reads.
