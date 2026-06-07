# innate-aiswitcher SPEC

## 1. Product Goal

`innate-aiswitcher` 是本地 AI Agent Provider switcher。它要让用户在 terminal 启动一个 Agent session 时选择使用哪个 LLM Provider，并用一个共享 Provider 中心避免重复配置 API key、base URL 和模型。

核心约束：Provider 与 Agent Adapter 必须解耦。Provider 是共享连接配置；Adapter 是启动时投影逻辑；Profile 是 Agent + Provider 的可选覆盖绑定。

## 2. Non-Goals For MVP

- 不在 MVP 中实现完整 Web UI。
- 不在 MVP 中承诺 Trae/OpenCode 的最终真实配置协议；当前先用 OpenAI-compatible env adapter。
- 不把 Provider 复制到每个 Agent 自己的 ProviderManager。
- 不把真实 API key 暴露到公开 REST 响应。
- 不按协议猜测 default model；default model 必须来自 Provider/template 或用户显式输入。

## 3. Data Model

PocketBase/SQLite collections:

- `providers`
  - Global LLM Provider center.
  - Fields: `slug`, `name`, `base_url`, hidden `api_key`, `api_protocol`, `default_model`, `headers`, `endpoints`, `capabilities`, `notes`, `active`.
  - `endpoints` is a JSON map for optional provider-specific endpoint overrides such as `chat_completions`, `responses`, `messages`, and `models`.
  - Public REST read is allowed; hidden `api_key` is not returned.
- `agents`
  - Agent catalog.
  - Fields: `slug`, `name`, `binary`, `adapter`, `env_map`, `active`.
  - Seeded slugs: `claude`, `codex`, `gemini`, `kimi`, `trae`, `opencode`.
- `profiles`
  - Agent-specific binding to a shared Provider.
  - Fields: `slug`, `name`, relation `agent`, relation `provider`, `model`, `config_overrides`, `env_overrides`, `default_args`, `is_default`.
  - Must not duplicate Provider credentials.
- `bindings`
  - Reserved for global/project/session default resolution.
- `launch_history`
  - Session launch audit trail.
- `settings`
  - Reserved local settings collection.

Shared config file mirror:

```toml
version = 1

[[providers]]
slug = "minimax"
base_url = "https://api.example.com/v1"
api_protocol = "openai_chat"
default_model = "gpt-test"
endpoints = { chat_completions = "/chat/completions", models = "/models" }

[[profiles]]
slug = "codex-minimax"
agent = "codex"
provider = "minimax"
model = "gpt-test"
```

## 4. Adapter Contract

Adapters are loaded through a registry map keyed by `agents.adapter`. The core launch path must call the registered builder instead of adding protocol/provider-specific switch branches.

Adapters receive `Agent + Provider + optional Profile + LaunchOptions` and return a `LaunchPlan`:

```json
{
  "command": "codex",
  "cwd": "/path/to/project",
  "env": { "CODEX_HOME": "/tmp/aisw-codex-..." },
  "files": { "config": "/tmp/aisw-codex-.../config.toml" }
}
```

Current adapters:

- `claude`
  - Writes a temporary settings JSON.
  - Sets Anthropic-compatible env keys inside settings.
  - Uses `claude --settings <temp-file>`.
- `codex`
  - Writes a temporary `CODEX_HOME`.
  - Generates `config.toml` and `auth.json` from the shared Provider.
  - Does not mutate the user's real Codex config.
- `gemini`
  - Uses session env: `GEMINI_API_KEY`, `GOOGLE_GEMINI_BASE_URL`.
- `openai_env`
  - Uses session env: `OPENAI_API_KEY`, `OPENAI_BASE_URL`.
  - Used by Kimi/Trae/OpenCode until their exact adapter contracts are finalized.

## 5. CLI Contract

Canonical Go CLI entrypoint:

```bash
go run ./cmd/aisw
go install ./cmd/aisw
```

The CLI root must not bootstrap PocketBase just because the process started. Backend initialization is lazy:

- Commands such as `provider presets`, `config template`, `--help`, and shell completion must not create or open the PocketBase data dir.
- Commands that read/write SQLite collections may bootstrap PocketBase core in-process.
- Only `serve` may start the HTTP server.

Provider management:

```bash
aisw provider add SLUG --base-url URL --api-key KEY --protocol openai_chat --model MODEL
aisw provider add SLUG --base-url URL --api-key-env ENV_NAME
aisw provider add SLUG --base-url URL --api-key-env ENV_NAME --endpoint chat_completions=/chat/completions --endpoint models=/models
aisw provider list
aisw provider delete SLUG
aisw provider presets
```

Profile management:

```bash
aisw profile add SLUG --agent AGENT --provider PROVIDER --model MODEL
aisw profile list
```

Session startup:

```bash
aisw start AGENT PROVIDER_OR_PROFILE --dry-run
aisw start AGENT PROVIDER_OR_PROFILE -- [native args]
```

Connectivity testing:

```bash
aisw test provider SLUG
aisw test provider SLUG --model MODEL
aisw test models SLUG
```

Config mirror:

```bash
aisw config template --path PATH
aisw config export --path PATH
aisw config export --path PATH --include-secrets
aisw config import --path PATH
aisw config import --path PATH --backup-path BACKUP_PATH
aisw config import --path PATH --no-backup
```

`config import` must create a backup by default before applying imported providers/profiles. File writes must be atomic (`temp file + fsync + rename`), and imports must apply providers/profiles inside a SQLite transaction.

## 5.1 TUI Contract

The default interactive TUI must provide:

- Start an agent session.
- List providers with slug, display name, API protocol, default model, base URL, and key status.
- Configure provider from bundled templates.
- Test provider connectivity from the TUI.

Provider configuration must guide API key, URL format, base URL, and default model. The MiniMax template must expose at least these URL format choices:

- OpenAI-compatible URL/protocol.
- Claude Code-compatible URL/protocol.

## 6. REST Contract

Default serve behavior:

- `aisw serve` starts the API server without enabling the PocketBase admin UI.
- `aisw serve` hides the PocketBase startup banner and admin install URL by default.
- To enable the admin UI at `/_`, start with `--admin-ui`.
- To show the PocketBase startup banner/admin install URL, start with `--show-admin-banner`.

Example:

```bash
aisw --admin-ui --show-admin-banner serve --http 127.0.0.1:8090
```

Custom REST endpoints:

- `GET /api/aisw/health`
  - Returns service health.
- `GET /api/aisw/catalog`
  - Returns agents and providers with provider API keys redacted.
- `GET /api/aisw/providers/{slug}/models`
  - Lists provider models using the same endpoint/auth rules as CLI.
- `POST /api/aisw/providers/{slug}/test`
  - Runs the same provider connectivity test as CLI.
  - Body: `{ "model": "optional-model" }`.

PocketBase collection REST:

- `GET /api/collections/agents/records`
- `GET /api/collections/providers/records`
- `GET /api/collections/profiles/records`

Collections are public read for UI discovery. Anonymous create/update/delete is not enabled.

## 6.1 Template Contract

Config and provider templates must be embedded into the binary with `go:embed` so `bin/aisw` can run template commands without external files.

Embedded templates:

- `config.example.toml`
  - Full shared config template with `[[providers]]` and `[[profiles]]`.
- `provider-presets.toml`
  - TUI/CLI provider preset source.
  - Each URL option must include `api_protocol`, `base_url`, `default_model`, and endpoint overrides for the protocol-specific request/model endpoints.

## 7. Build And Task Contract

The root `Taskfile.yml` is the canonical local build entrypoint because this repository also contains reference projects with unrelated Go packages.

Tasks:

- `task build`
  - Formats source and builds `bin/aisw` from `./cmd/aisw`.
- `task test`
  - Runs scoped tests: `go test . ./cmd/aisw ./cmd/mock-provider ./internal/... ./migrations`.
- `task compile`
  - Compiles scoped packages: `go build . ./cmd/aisw ./cmd/mock-provider ./internal/... ./migrations`.
- `task verify`
  - Runs format, test, compile, build, and `go mod verify`.
- `task smoke`
  - Uses a temporary PocketBase data dir and local mock provider to test provider add, provider API test, model listing, profile add, dry-run launch, and config export.
- `task serve`
  - Starts PocketBase REST server for local UI/API work.
- `task clean`
  - Removes local build artifacts.

Do not use `go test ./...` as the primary verification command until reference projects are moved outside the root Go module or isolated with their own modules/build tags.

## 8. Validation Requirements

Required before handoff:

```bash
task verify
task smoke
```

Expected coverage:

- Unit tests validate OpenAI-compatible and Anthropic-compatible API key request construction.
- Unit tests validate endpoint overrides and OpenAI Responses API request construction.
- Unit tests validate that one shared Provider is projected differently by Claude and Codex adapters.
- Smoke test validates automatic migrations, provider/profile persistence, API key testing against a mock provider, model listing, dry-run startup, and config export.

## 9. Future Work

1. Add project/global default binding resolution.
2. Add exact Trae/OpenCode adapters after confirming their current CLI contracts.
3. Add authenticated write REST for UI operations.
4. Expand provider preset catalog and model discovery beyond the current smoke-tested MVP.
5. Add secret storage backends such as macOS Keychain.
6. Add launch history UI and provider health dashboard.
