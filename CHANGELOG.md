# Changelog

All notable changes to `innate-aiswitcher` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-07-07

### Added

- Embedded Web UI at `GET /` of `aisw serve` (Provider/Profile CRUD, preset import, connectivity tests).
  - New files: `internal/webui/{webui.go, static/{index.html,app.js,style.css}}`.
  - `scripts/sync-data-db.{sh,ps1}` for syncing `pb_data/data.db` into `~/.innate-aiswitcher/`.
- `aisw init` command writes a `.aiswrc` TOML in the current directory so `aisw start AGENT` auto-resolves a default profile / provider / agent per project.
- `aisw start` now consults `.aiswrc` (walks up from `$PWD`) unless `--ignore-project` is set; mismatched `agent` is rejected with an error.
- `--init-config PATH` global flag: when PocketBase bootstraps against an empty database, import the given TOML first (defaults to `~/.innate-aiswitcher/init-config.toml`).
- `aisw config dump` writes a full config (with secrets) to the init-config path; `aisw config import/export` now accept `--format toml|json` and auto-detect on import.
- Codex adapter gains `experimental_bearer_token` auth mode and `model_context_window` capability (preset examples for `minimax-codex`, `xiaomi-codex`).
- `agents.skip_permissions_arg` + `agents.skip_permissions_default` (claude → `--dangerously-skip-permissions`, kimi → `--yolo`, both on by default) plus per-profile `skip_permissions` override; refactored into `applySkipPermissions` / `resolveSkipPermissions`.
- Claude adapter always sets `skipDangerousModePermissionPrompt: true` so the UI confirmation is bypassed; the real permission gate is the resolved CLI flag.
- `adapter.PersistDefault` writes a default-profile Claude settings file (`~/.claude/settings.json`) so `claude` works without going through `aisw start` after `--default` is marked.
- TUI preset cards (lipgloss-styled), provider value prefill when editing, and key-status indicator on `aisw configure`.
- `aisw profile add --skip-permissions true|false` for per-profile override.
- `task install` builds and copies `bin/aisw` to `~/.local/bin`; cross-compile helpers `task build:windows` and `task build:linux`.
- Hermes and OpenClaw agents seeded (`migrations/1780567400_seed_hermes_openclaw_agents.go`).
- Provider preset catalog refresh: MiniMax (OpenAI / Claude / Codex), Xiaomi MiMo (OpenAI / Claude / Codex), Anthropic, Volcengine Ark (OpenAI / Claude) with `claude_extra_env` and `codex_auth_mode` capabilities.
- Documentation site (`task docs:dev`, `task docs:build`) published to GitHub Pages via `.github/workflows/docs.yml`.
- `docs/` site content: SPEC, USAGE, API, AGENT_TESTING, DEVELOPMENT, GOAL, plus tutorials (`INSTALL_GUIDE`, `HERMES_DEEPSEEK_PROXY`).

### Changed

- Default provider templates refreshed to current models: `MiniMax-M3`, `kimi-latest`, `deepseek-v4-flash`, `gpt-4o`, `mimo-v2.5-pro`, `claude-sonnet-4-6-20250715`, `glm-5.2`.
- MiniMax preset URLs updated from `https://api.minimax.chat/...` to `https://api.minimaxi.com/...`; same correction applied across `README.md`, `docs/USAGE.md`, `docs/AGENT_TESTING.md`, `docs/SPEC.md`, `docs/GOAL.md`, `docs/DEVELOPMENT.md`, `docs/tutorial/INSTALL_GUIDE.md`, `docs/tutorial/HERMES_DEEPSEEK_PROXY.md`, `tutorials/tutorial/INSTALL_AGENTS.md`, `internal/templates/files/config.example.toml`, `internal/templates/files/providers.toml`.
- `aisw serve` defaults to `127.0.0.1:8090`; PocketBase admin UI remains off unless `--admin-ui` is passed (along with `--show-admin-banner`); `--quiet` now disables HTTP access logs.
- `aisw config import` always writes an atomic, secret-bearing backup by default; use `--no-backup` to skip; `--backup-path` customises the destination.
- `aisw provider add --api-key-env ENV_NAME` continues to read from env at write time and is now reflected in provider/preset import flows (`POST /api/aisw/providers/from-preset`).
- CLI surface now exposes `aisw init`, `aisw config dump`, `aisw config import/export --format toml|json`, `aisw start --ignore-project --cwd --terminal current|ghostty|terminal --dry-run`.
- REST surface: `GET /api/aisw/catalog`, `GET /api/aisw/agents`, `GET /api/aisw/presets`, plus full Provider/Profile CRUD and `/providers/{slug}/models`, `/providers/{slug}/test`. `api_key` is masked on all custom endpoints and a hidden field on `providers` collection reads.
- Agent adapter registry now includes `hermes` and `openclaw` mapped to the generic `openai_env` builder.
- `aisw serve` access logs are routed through `internal/app/serve_log.go` and can be silenced with `--quiet` or the inverse flag.

### Fixed

- Claude Code confirmation dialog ("enable dangerous mode?") no longer blocks startup — settings JSON now unconditionally sets `skipDangerousModePermissionPrompt: true`.
- Provider update via `PUT /api/aisw/providers/{slug}` preserves the stored API key when the request body leaves `api_key` empty.
- `aisw config import` rolls back provider/profile writes on failure (SQLite transaction wraps the upsert).
- TUI no longer overwrites an existing API key when the field is left empty during `Configure provider`.
- `aisw provider add minimax` examples corrected to use current MiniMax endpoints (`api.minimaxi.com`) instead of the legacy `api.minimax.chat` host.

### Documentation

- `README.md`, `docs/USAGE.md`, `docs/AGENT_TESTING.md`, `docs/SPEC.md`, `docs/GOAL.md`, `docs/DEVELOPMENT.md`, `docs/tutorial/INSTALL_GUIDE.md`, `docs/tutorial/HERMES_DEEPSEEK_PROXY.md` aligned with the current code (provider catalog, command flags, `--ignore-project`, `.aiswrc`, Codex bearer-token mode, `--init-config`, JSON config format).
- Embedded `config.example.toml` template updated to use `api.minimaxi.com` and `MiniMax-M3`.
- This `CHANGELOG.md` introduced.

## [0.1.0] - 2026-07-06

### Added

- Initial public release of `innate-aiswitcher` (`aisw`) — local LLM provider switcher for Claude Code, Codex CLI, Gemini CLI, Kimi, Trae, OpenCode, Hermes, and OpenClaw.
- PocketBase + SQLite data store (`~/.innate-aiswitcher/pb_data/`) with `providers`, `agents`, `profiles`, `bindings`, `launch_history`, `settings` collections.
- Adapter registry with `claude`, `codex`, `gemini`, `openai_env` builders and `BuildPlan` / `Execute` launch path.
- CLI: `provider add/list/delete/presets`, `profile add/list`, `start [--dry-run]`, `test provider`, `test models`, `config template/import/export`.
- Embedded `provider-presets.toml` and `config.example.toml` via `go:embed`.
- `task build`, `task test`, `task verify`, `task smoke`, `task serve`, `task run` Taskfile tasks.
- Cross-platform binary builds for `windows/amd64`, `linux/amd64`, `darwin/amd64`, `darwin/arm64`.

[Unreleased]: https://github.com/variableway/innate-aiswitcher/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/variableway/innate-aiswitcher/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/variableway/innate-aiswitcher/releases/tag/v0.1.0
