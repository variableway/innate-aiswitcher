# innate-aiswitcher SPEC

## 1. Product Goal

`innate-aiswitcher` 是本地 AI Agent Provider switcher。它要让用户在 terminal 启动一个 Agent session 时选择使用哪个 LLM Provider，并用一个共享 Provider 中心避免重复配置 API key、base URL 和模型。

核心约束：Provider 与 Agent Adapter 必须解耦。Provider 是共享连接配置；Adapter 是启动时投影逻辑；Profile 是 Agent + Provider 的可选覆盖绑定。

## 2. Non-Goals For MVP

- 不在 MVP 中实现 launch history 可视化仪表盘。
- 不在 MVP 中承诺 Trae/OpenCode 的最终真实配置协议；当前先用 OpenAI-compatible env adapter。
- 不把 Provider 复制到每个 Agent 自己的 ProviderManager。
- 不把完整 API key 暴露到公开 REST 响应（掩码或 hidden field）。
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
  - Seeded slugs: `claude`, `codex`, `gemini`, `kimi`, `trae`, `opencode`, `hermes`, `openclaw`.
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
  - Generates `config.toml` (and optionally `auth.json`) from the shared Provider.
  - Respects `capabilities.codex_auth_mode`:
    - `requires_openai_auth` (default) → writes `auth.json` with `OPENAI_API_KEY`.
    - `experimental_bearer_token` → embeds the key directly into `config.toml` (used by MiniMax / Xiaomi presets).
  - `capabilities.codex_model_context_window` is appended to `config.toml` when set.
  - Does not mutate the user's real Codex config.
- `gemini`
  - Uses session env: `GEMINI_API_KEY`, `GOOGLE_GEMINI_BASE_URL`.
- `openai_env`
  - Uses session env: `OPENAI_API_KEY`, `OPENAI_BASE_URL`.
  - Used by Kimi, Trae, OpenCode, Hermes, OpenClaw until their exact adapter contracts are finalized.

### 4.1 Agent Skip-Permissions

`agents.skip_permissions_arg` (string, e.g. `--dangerously-skip-permissions` or `--yolo`) and `agents.skip_permissions_default` (bool) seed the agent. `profiles.skip_permissions` (`""` | `"true"` | `"false"`) overrides per profile. The Claude adapter always sets `skipDangerousModePermissionPrompt: true` in its temp settings so the UI confirmation is skipped; the actual permission gate is the CLI flag, applied by `applySkipPermissions` based on the resolved value above.

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
aisw config export --path PATH --format toml|json
aisw config import --path PATH
aisw config import --path PATH --backup-path BACKUP_PATH
aisw config import --path PATH --no-backup
aisw config import --path PATH --format toml|json   # auto-detect by default
aisw config dump [--path PATH] [--format toml|json]  # full dump with secrets to the init-config path
```

`config import` must create a backup by default before applying imported providers/profiles. File writes must be atomic (`temp file + fsync + rename`), and imports must apply providers/profiles inside a SQLite transaction. Wire format defaults to TOML but JSON is supported for both `export` and `import` (auto-detected from the file extension or explicit `--format`).

Project config:

```bash
aisw init --profile SLUG [--agent SLUG] [--provider SLUG] [--force]
aisw start AGENT [SELECTOR] -- [native args] [--ignore-project] [--dry-run] [--cwd DIR] [--terminal current|ghostty|terminal]
```

`aisw init` writes a TOML `.aiswrc` (with `profile`/`agent`/`provider` keys) into the current directory. `aisw start` walks up from `$PWD` for `.aiswrc` and uses the bound `profile` or `provider` unless `--ignore-project` is set. If `.aiswrc.agent` is set and conflicts with the requested agent, startup is rejected.

First-run init:

```bash
aisw --init-config PATH serve   # on empty DB, import this TOML before serving
```

When PocketBase bootstraps against an empty database, `aisw` looks up the file at `configfile.InitConfigPath()` (default `~/.innate-aiswitcher/init-config.toml`, override via `--init-config`) and imports it. If the path does not exist and `--init-config` was not passed, the bootstrap is a no-op.

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

- `aisw serve` starts the API server and embedded Web UI at `/`.
- `aisw serve` does not enable the PocketBase admin UI by default.
- PocketBase startup banner is hidden unless `--show-admin-banner` is passed.
- `--quiet` disables HTTP access logs.
- Default listen address: `127.0.0.1:8090` (override with `--http`).

Example:

```bash
aisw serve --http 127.0.0.1:8090
aisw --admin-ui --show-admin-banner serve --http 127.0.0.1:8090
```

### 6.1 Custom REST Endpoints

Discovery:

- `GET /api/aisw/health` — service health.
- `GET /api/aisw/catalog` — agents + providers (`api_key` blanked).
- `GET /api/aisw/agents` — agent catalog.
- `GET /api/aisw/presets` — bundled provider presets.

Provider CRUD + operations:

- `GET /api/aisw/providers` — list (masked `api_key`).
- `GET /api/aisw/providers/{slug}` — get one.
- `POST /api/aisw/providers` — create/upsert.
- `PUT /api/aisw/providers/{slug}` — update; empty `api_key` preserves existing.
- `DELETE /api/aisw/providers/{slug}` — delete.
- `POST /api/aisw/providers/from-preset` — import from preset (`preset_slug`, `option_slug`, `api_key`).
- `GET /api/aisw/providers/{slug}/models` — model listing (same as CLI).
- `POST /api/aisw/providers/{slug}/test` — connectivity test; body `{ "model": "optional" }`.

Profile CRUD:

- `GET /api/aisw/profiles`
- `POST /api/aisw/profiles`
- `PUT /api/aisw/profiles/{slug}`
- `DELETE /api/aisw/profiles/{slug}`

Custom write endpoints are unauthenticated and intended for localhost use only.

### 6.2 Web UI

- Static files embedded in `internal/webui/static/` via `go:embed`.
- Served at `GET /` (plus `/style.css`, `/app.js`).
- Uses `/api/aisw/*` for Provider/Profile CRUD, preset import, and connectivity tests.
- Shares the same PocketBase SQLite database as the CLI.
- Features: Providers (list / add / edit / delete / preset import / Test), Profiles (list / add / edit / delete / default toggle / per-profile model & CLI args override).

### 6.3 PocketBase Collection REST

Read-only public access:

- `GET /api/collections/agents/records`
- `GET /api/collections/providers/records`
- `GET /api/collections/profiles/records`

Anonymous create/update/delete is not enabled on collections.

## 6.4 Template Contract

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
- `task build:windows`, `task build:linux`
  - Cross-compile for the given OS/arch into `bin/aisw_<os>_<arch>[.exe]`.
- `task install`
  - Builds and copies `bin/aisw` to `~/.local/bin`.
- `task test`
  - Runs scoped tests: `go test . ./cmd/aisw ./cmd/mock-provider ./internal/... ./migrations`.
- `task vet`
  - Runs `go vet` over the same scoped package set.
- `task compile`
  - Compiles scoped packages: `go build . ./cmd/aisw ./cmd/mock-provider ./internal/... ./migrations`.
- `task verify`
  - Runs format, vet, test, build, and `go mod verify`.
- `task smoke`
  - Uses a temporary PocketBase data dir and local mock provider to run the full `config template` → `config import` → `provider presets` → `provider add` → `test provider` → `test models` → `profile add` → `start --dry-run` → `config export --include-secrets` flow.
- `task serve`
  - Starts REST API + embedded Web UI at `http://127.0.0.1:8090/`.
- `task serve:verbose`
  - Same as `serve` but passes `--show-admin-banner`.
- `task dev`
  - Builds binary then runs `serve`.
- `task run`
  - Launches the interactive TUI (`go run ./cmd/aisw`).
- `task docs:dev`
  - Starts the docmd dev server for the documentation site.
- `task docs:build`
  - Builds the static documentation site to `site/` (published via GitHub Actions).
- `task clean`
  - Removes local build artifacts.

Documentation site: https://variableway.github.io/innate-aiswitcher/ (source: `docs/`, config: `docmd.config.json`, deploy: `.github/workflows/docs.yml`).

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

1. Add global/project/session binding tables on top of `.aiswrc` (`bindings` collection is already in place; resolution logic still lives in code).
2. Add exact Trae/OpenCode adapters after confirming their current CLI contracts.
3. Add authentication for write REST when exposing beyond localhost.
4. Expand provider preset catalog and model discovery beyond the current smoke-tested MVP.
5. Add secret storage backends such as macOS Keychain.
6. Add launch history UI and provider health dashboard.
