# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`innate-aiswitcher` (CLI: `aisw`) is a local LLM Provider switcher for AI coding agents. It stores a shared Provider catalog in PocketBase/SQLite, then projects the selected Provider into the temporary config (settings JSON, `CODEX_HOME`, env vars, etc.) that Claude Code, Codex CLI, Gemini CLI, Kimi, Trae, OpenCode, Hermes, or OpenClaw expects at session start.

The core invariant: **Provider and Agent Adapter are decoupled**. Providers live in a global table; adapters are pure launch-time projections; `profiles` are optional Agent+Provider bindings that hold only per-agent overrides (model, args, env/config overrides, default flag). Same Provider can be projected by many adapters without duplication.

## Build / Lint / Test — use Taskfile, never `go test ./...`

The repo root contains reference projects with broken external deps, so unscoped `go test ./...`, `go build ./...`, or `go mod tidy` will fail. Always go through `Taskfile.yml`:

```bash
task build       # fmt + go build -o bin/aisw ./cmd/aisw
task test        # scoped: go test . ./cmd/aisw ./cmd/mock-provider ./internal/... ./migrations
task compile     # scoped: go build across the same set
task verify      # fmt + test + compile + build + go mod verify
task smoke       # full CLI integration: temp pb_data + local mock-provider + provider/profile/test/start/export
task serve       # pocketbase REST on 127.0.0.1:8090
task run         # go run ./cmd/aisw — launches the TUI
task install     # build + cp bin/aisw to ~/.local/bin
task clean       # rm -rf bin
task fmt         # gofmt -w main.go cmd/aisw internal migrations
```

To run a single test: `go test ./internal/adapter -run TestBuildClaudePlan` (still scoped to one of the package roots listed in `Taskfile.yml`'s `PKGS` var, not `./...`).

## Architecture

```
main.go                    # bootstraps cobra CLI; imports migrations package for side-effect registration
cmd/aisw/main.go           # thin entry; calls app.NewCLI()
cmd/mock-provider/main.go  # local OpenAI-compatible server used only by task smoke
internal/app/app.go        # PocketBase wiring, cobra command tree, custom REST routes
internal/store/            # typed wrappers over PocketBase collections (Agent, Provider, Profile, LaunchHistory)
internal/adapter/          # launch-plan builders + the Builder registry map
internal/configfile/       # TOML export/import of the shared config mirror
internal/httpcheck/        # connectivity check + model listing (CLI + REST)
internal/templates/        # go:embed of config.example.toml and provider-presets.toml
internal/tui/              # huh/bubbletea forms for configure-provider / start-session
internal/projectconfig/    # .aiswrc discovery (walk-up lookup)
internal/safefile/         # atomic temp-file + fsync + rename writer
migrations/                # PocketBase collection migrations + agent seeds
```

### Data flow for `aisw start AGENT SELECTOR`

1. `app.startCommand` lazily bootstraps PocketBase on first DB-touching command (see "Lazy bootstrap" below).
2. `store.ResolveSelector(agentSlug, selector)` — if `selector` is empty, walks up from `$PWD` for `.aiswrc` (unless `--ignore-project`); otherwise tries `GetProfile(selector)` then `GetProvider(selector)`. Falls back to the agent's `is_default=true` profile when no selector is given.
3. `adapter.BuildPlan(agent, provider, profile, opts)` looks up `agent.Adapter` in the `builders` map and dispatches. No protocol/provider `switch` lives on the core path.
4. `adapter.Execute(plan, cleanup, opts)` either dry-runs (prints JSON), spawns the binary in-process with env-merged `cmd.Env`, or hands off to Ghostty/Terminal via `osascript`/`open` (macOS only).
5. `store.SaveLaunchHistory(...)` writes a `launch_history` row.

### Adapter registry (`internal/adapter/adapter.go`)

The `builders` map is the only place a new agent's projection is wired. To add a new agent:

1. Write a `Builder` function returning `(LaunchPlan, cleanup func(), error)`.
2. Add `adapter.Register("my_agent", buildMyAgentPlan)` — or extend the package-level `builders` map.
3. Add a unit test in `internal/adapter/adapter_test.go` asserting the plan (use `t.TempDir()` for any filesystem side effects; the existing tests use `--dry-run` plans).
4. Add an agent seed in `migrations/` (compare `1780565700_init_aisw.go` and `1780567400_seed_hermes_openclaw_agents.go` for the two patterns: full init or `FindFirstRecordByFilter` upsert).

Current adapters: `claude` (temp `--settings` JSON), `codex` (temp `CODEX_HOME` with `config.toml` + `auth.json`, supports `codex_auth_mode=experimental_bearer_token` capability for bearer-token auth), `gemini` (env), and the generic `openai_env` used by `kimi`, `trae`, `opencode`, `hermes`, `openclaw` (just `OPENAI_API_KEY` + `OPENAI_BASE_URL`).

### Data model (PocketBase collections, see `migrations/1780565700_init_aisw.go`)

- `providers` — slug-indexed; `api_key` is a **hidden** field (not returned by public REST), `endpoints` is a JSON map for protocol-specific overrides (`chat_completions`, `responses`, `messages`, `models`), `capabilities` is freeform JSON consumed by adapters (e.g. `claude_extra_env`, `codex_auth_mode`).
- `agents` — seeded with `claude`, `codex`, `gemini`, `kimi`, `trae`, `opencode`, `hermes`, `openclaw`. The `adapter` field keys into `internal/adapter/builders`.
- `profiles` — `agent`/`provider` are `RelationField` with `CascadeDelete`; `is_default=true` makes it the implicit selection when `start` is invoked without a selector.
- `launch_history`, `bindings`, `settings` — reserved/future-use.
- Read is public for discovery; **anonymous write is not enabled**.

### Lazy bootstrap of PocketBase

`internal/app/app.go` is structured so the root process does not open the PocketBase data dir for help/template/preset commands. `getPB` is invoked only inside `RunE` of commands that actually need the DB (`provider add/list/delete`, `profile add/list`, `start`, `test`, `config import/export/dump`, `init`, `serve`). `provider presets`, `config template`, and `--help` work without ever touching the data dir. Preserve this property when adding commands.

On first DB bootstrap with an empty database, `maybeInitConfig` imports the file at `configfile.InitConfigPath()` (default `~/.innate-aiswitcher/init-config.toml`) — only when the file actually exists and providers/profiles are empty.

### Atomic file writes

All disk writes that must not be partially observed (config export, init-config template, codex `config.toml`/`auth.json`, claude `settings.json`) go through `internal/safefile.Write` — temp file in the same dir, `chmod 0o600`, `Sync`, `Close`, `Rename`, then dir `Sync`. Reuse it; do not write these files directly with `os.WriteFile`.

`config import` always runs providers/profiles inside a `Store.RunInTransaction` so partial imports roll back. `--backup` (default on) exports the current state via `configfile.Backup` before applying the import.

### Templates and provider presets

`internal/templates/files/` is embedded with `//go:embed files/*` so `bin/aisw` works offline. To add a new provider preset, edit `provider-presets.toml` (one `[[presets]]` block per provider, with one or more `[[presets.url_options]]` entries; each option must include `api_protocol`, `base_url`, `default_model`, and the protocol-specific endpoint overrides). Preset→provider generation lives in `templates.ProviderFromPreset`.

### `.aiswrc` project config

`internal/projectconfig` walks up from `$PWD` looking for `.aiswrc`. Format: TOML with optional `profile`, `agent`, `provider`. The `aisw init` command writes one (`--force` to overwrite). `aisw start` consults `.aiswrc` automatically unless `--ignore-project` is set, and rejects startup if `.aiswrc.agent` is set and conflicts with the requested agent.

### REST surface (when `aisw serve` is running)

Custom routes in `internal/app/app.go` `registerRoutes`: `GET /api/aisw/health`, `GET /api/aisw/catalog` (returns agents + providers with `api_key` blanked), `GET /api/aisw/providers/{slug}/models`, `POST /api/aisw/providers/{slug}/test` with optional `{"model": "..."}` body. Standard PocketBase collection reads are exposed at `/api/collections/{agents|providers|profiles}/records`. Admin UI is off by default — pass `--admin-ui` and `--show-admin-banner` to enable. Keys are never returned.

## Conventions

- `gofmt`-clean only. `task fmt` runs on every `task build`/`task verify`.
- `internal/safefile` is the only path for writing secret-bearing config files; `0o600` perm is expected.
- Default model for a Provider is not guessed by protocol — must come from `default_model` (set by preset, template, or explicit `--model`). Adapters error out if neither provider nor profile supplies a model.
- `--api-key-env ENV_NAME` reads the key from env at `provider add` time; it is then stored in the hidden `api_key` field and synced across slug-prefix siblings (e.g. `minimax-openai` and `minimax-claude` share the same key when one is set).
- `unsafe`/`switch` on adapter/protocol names lives in `internal/adapter/adapter.go`'s `codexWireAPI` and `internal/httpcheck/check.go`'s `requestFor` — the latter is the only acceptable place to translate `api_protocol` to a request shape.
- Commit messages use conventional prefixes (`feat:`, `fix:`, `test:`, `docs:`, `chore:`); see `docs/DEVELOPMENT.md`.

## Smoke test reference flow

`task smoke` exercises, in order against a fresh temp `pb_data` and a local mock on `127.0.0.1:18990`:
`config template` → `config import` → `provider presets` → `provider add local` → `test provider local` → `test models local` → `profile add codex-local` → `start codex codex-local --dry-run` → `config export --include-secrets`. Use this as the integration baseline when changing anything that touches the data path.

## More

- Architecture spec: `docs/SPEC.md`
- REST reference: `docs/API.md`
- Per-agent testing walkthroughs: `docs/AGENT_TESTING.md`
- User-facing CLI recipes: `README.md`, `docs/USAGE.md`
