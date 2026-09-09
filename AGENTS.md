# AGENTS.md

Guidance for AI coding agents (and humans) working in this repository. `CLAUDE.md` covers the same ground for Claude Code; keep the two in sync when changing conventions.

## Project overview

`innate-aiswitcher` (CLI binary: `aisw`) is a local LLM Provider switcher for AI coding agents. It is a **PocketBase application**: CLI, TUI, Web UI and REST API all live in a single Go binary, backed by SQLite.

It stores a shared, vendor-centric Provider catalog, then projects the selected Provider into the temporary config (settings JSON, `CODEX_HOME`, env vars, etc.) that each coding agent expects at session start.

**Core invariant: Provider and Agent Adapter are decoupled.**

- Providers live in a global table (one row per vendor, one API key, `models` list, per-protocol `variants`).
- Adapters are pure launch-time projections and own no Provider state.
- Profiles are optional Agent+Provider bindings holding only per-agent overrides (model, args, env/config overrides, default flag).
- The same Provider can be projected by many adapters without duplication.

Focused agents today: `claude`, `codex`, `opencode`. Retired agents (gemini/kimi/trae/hermes/openclaw) were removed by migration.

## Tech stack

- Go 1.26.x, module `github.com/variableway/innate-aiswitcher`
- PocketBase v0.39.x (framework + HTTP + DB; SQLite via `modernc.org/sqlite`)
- cobra (CLI), charmbracelet/huh + lipgloss (TUI), BurntSushi/toml (config/presets/`.aiswrc`)
- Ginkgo v2 + Gomega and plain `go test` for tests
- Frontend: Vite + React 19 + TypeScript, TanStack Router/Query, Tailwind CSS 4, shadcn/ui, xterm.js — built from `web/` and `go:embed`-ed into the binary
- Docs site: docmd (`docs/` → `site/`, published to GitHub Pages)

## Build / lint / test — use Taskfile, never `go test ./...`

The repo root contains reference material with broken external deps, so unscoped `go test ./...`, `go build ./...`, or `go mod tidy` **will fail**. Always go through `Taskfile.yml`:

```bash
task build       # web:build + go build -o bin/aisw — full single binary
task test        # scoped: go test . ./cmd/aisw ./cmd/mock-provider ./internal/... ./migrations
task compile     # scoped go build across the same package set
task verify      # fmt + vet + test + compile + build + go mod verify
task smoke       # full CLI integration: temp pb_data + local mock-provider + provider/profile/test/start/export
task serve       # REST API + Web UI on 127.0.0.1:8090
task web         # serve + open browser (web app + browser PTY terminals)
task web:dev     # Vite dev server (/api proxied to 127.0.0.1:8090)
task web:build   # build web/ and sync dist into internal/webui/dist (go:embed)
task run         # launch the interactive TUI
task install     # build + cp bin/aisw to ~/.local/bin
task fmt         # gofmt -w main.go cmd/aisw internal migrations
task docs:dev    # docmd dev server for the documentation site
task docs:build  # build static docs to site/
```

Single test: `go test ./internal/adapter -run TestBuildClaudePlan` — still scoped to a package root from `Taskfile.yml`'s `PKGS` var, never `./...`.

Use `task verify` + `task smoke` as the baseline before handing off any change that touches the data path.

## Repository layout

```
main.go                    # bootstraps cobra CLI; blank-imports migrations for registration
cmd/aisw/                  # real CLI entry (Taskfile build target); calls app.NewCLI()
cmd/mock-provider/         # local OpenAI-compatible server used only by `task smoke`
internal/app/app.go        # PocketBase wiring, cobra command tree, custom REST routes (~1450 lines)
internal/store/            # typed wrappers over PocketBase collections (Agent, Provider, Profile,
                           #   Binding, LaunchHistory); normalize.go merges legacy per-protocol rows
internal/providerconfig/   # LLM Provider Config abstraction: agent-adapter -> protocol-variant resolution
internal/adapter/          # launch-plan builders + the Builder registry map (preview.go = dry-run)
internal/agentconfig/      # read/write real agent config files (claude/codex/opencode), whitelist + atomic writes
internal/configfile/       # TOML/JSON export/import of the shared config mirror (transactional upsert)
internal/httpcheck/        # connectivity check + model listing (anthropic / openai_chat / openai_responses)
internal/templates/        # go:embed of config.example.toml, provider-presets.toml, providers.toml; user presets
internal/tui/              # huh/lipgloss interactive UI (single tui.go)
internal/webui/            # go:embed all:dist — built web app, SPA fallback
internal/terminal/         # WebSocket -> local PTY bridge for browser terminals
internal/projectconfig/    # .aiswrc discovery (walk-up from cwd)
internal/safefile/         # atomic temp-file + fsync + rename writer (0o600)
migrations/                # PocketBase Go migrations + agent seeds (timestamped filenames, Automigrate)
web/                       # frontend source (Vite + React); dist synced into internal/webui/dist
docs/                      # docmd documentation source (USAGE.md, API.md, commands/*)
scripts/                   # install-agents.{sh,ps1}, sync-data-db.{sh,ps1}
tasks/prd/                 # PRD documents
gui-test-screenshots/      # manual GUI verification evidence (gitignored session artifacts, not automated tests)
bin/, site/, assets/       # build output / docs build output / empty leftovers — do not edit
```

## Architecture notes

### CLI command tree (cobra, `internal/app/app.go` `NewCLI()`)

```
aisw                                # no subcommand -> TUI
aisw provider add|list|delete|from-preset|preset ...|model list|add|remove
aisw profile add|list
aisw start AGENT [PROVIDER_OR_PROFILE] [--model M] [--dry-run] [--terminal ...] [--cwd] [--ignore-project]
aisw test provider SLUG | test models SLUG
aisw config export|dump|import|template
aisw serve [domain(s)]              # REST API, default 127.0.0.1:8090
aisw web                            # serve + open browser
```

Note: `NewCLI()` and `NewWithOptions()` register commands separately — keep both in sync when adding commands (`serve`/`web` come from the PocketBase RootCmd in one path).

### Data flow for `aisw start AGENT SELECTOR`

1. Lazy PocketBase bootstrap on first DB-touching command.
2. `store.ResolveSelector(agentSlug, selector)` — empty selector walks up for `.aiswrc` (unless `--ignore-project`), then profile, then provider; falls back to the agent's `is_default` profile.
3. `adapter.BuildPlan(...)` dispatches through the `builders` registry map. No protocol/provider `switch` on the core path.
4. `adapter.Execute(plan, ...)` — dry-run prints JSON; otherwise spawns the binary with merged env, or hands off to Ghostty/Terminal (macOS only).
5. `store.SaveLaunchHistory(...)` records the launch.

Model resolution priority: `--model` > Profile.model > Provider.default_model. If the Provider has a non-empty `models` list, the effective model must be on it.

### Vendor providers (`internal/providerconfig`)

One row per vendor holds one `api_key`, a `models` list, and a `variants` map keyed by wire protocol (`anthropic` / `openai_responses` / `openai_chat`). Adapter → protocol mapping: `claude` → `anthropic`; `codex` → `openai_responses` (falls back to `openai_chat`); `openai_env` (opencode) → `openai_chat`.

Legacy single-protocol rows still work — resolution passes them through when the adapter speaks their protocol. On startup, `normalizeLegacyVendors` (`internal/store/normalize.go`) merges old `xxx-claude`-style rows into vendor rows.

### Data model (PocketBase collections)

- `providers` — slug-indexed; `api_key` is a **hidden** field (never returned by public REST); `models`, `variants`, `endpoints`, `capabilities` are JSON.
- `agents` — seeded by migrations; the `adapter` field keys into `internal/adapter/builders`.
- `profiles` — `agent`/`provider` relations with `CascadeDelete`; `is_default=true` is the implicit selection for selector-less `start`.
- `launch_history`, `bindings`, `settings` — reserved/future-use.
- Read is public for discovery; anonymous write is not enabled.

Migrations live in `migrations/` as timestamped Go files calling `migrations.Register(up, down)` in `init()`; the binary triggers them via blank import + `RunAppMigrations()` (Automigrate).

### REST surface (when `aisw serve` runs)

Custom routes under `/api/aisw/...` (full reference: `docs/API.md`):

- Discovery: `GET /api/aisw/health|catalog|agents|presets`
- Provider CRUD + ops: `GET/POST/PUT/DELETE /api/aisw/providers[/{slug}]`, `POST .../from-preset`, `GET/POST/DELETE .../models[/{model}]`, `POST .../test`
- Profile CRUD: `GET/POST/PUT/DELETE /api/aisw/profiles[/{slug}]`
- Terminal: `GET /api/aisw/terminal` (WebSocket → local PTY)
- `GET /` serves the embedded SPA with fallback; unmatched `/api/*` returns 404
- PocketBase collection reads: `GET /api/collections/{agents|providers|profiles}/records`; admin UI off by default (`--admin-ui --show-admin-banner` to enable)

All provider endpoints mask `api_key`; `PUT` with empty `api_key` preserves the stored key.

## Conventions

- **gofmt-clean only.** `task fmt` runs as part of `task build`/`task verify`.
- **Tests**: plain `go test` unit tests plus Ginkgo+Gomega BDD specs (`*_suite_test.go` bootstraps the suite). New behavioral coverage for provider/vendor/model flows should be Ginkgo specs — see `internal/providerconfig`, `internal/store`, `internal/adapter`, `internal/templates`.
- **Atomic writes**: all secret-bearing or config files (config export, codex `config.toml`/`auth.json`, claude `settings.json`) go through `internal/safefile.Write` (temp + 0o600 + fsync + rename + dir sync). Never `os.WriteFile` these directly.
- **Lazy bootstrap**: help/template/preset commands must not open the data dir. `getPB` is only invoked inside `RunE` of commands that need the DB. Preserve this when adding commands.
- **Default model is never guessed by protocol** — it comes from `default_model` (preset/template/`--model`). Adapters error when neither provider nor profile supplies one.
- **Protocol translation lives in exactly one place**: `internal/httpcheck/check.go` `requestFor` (and `codexWireAPI` in `internal/adapter/adapter.go`). Do not scatter `switch` on adapter/protocol names elsewhere.
- **Adding a new vendor**: add one `[[presets]]` block to `internal/templates/files/provider-presets.toml` — no Go code changes. Adding a model: `aisw provider model add`.
- **Adding a new agent adapter**: (1) write a `Builder` returning `(LaunchPlan, cleanup, error)`; (2) register in the `builders` map; (3) add a unit test asserting the dry-run plan (use `t.TempDir()`); (4) add an agent seed migration (see `1780565700_init_aisw.go` vs `1780567400_seed_hermes_openclaw_agents.go` for the two patterns).
- **Commit messages**: conventional prefixes (`feat:`, `fix:`, `test:`, `docs:`, `chore:`).
- **Frontend**: when editing `web/src`, follow the bundled shadcn skill rules (`.zcode/skills/shadcn`): semantic colors, `gap-*`, `Field` forms, full Card composition. Run oxlint via the web package scripts.

## Security rules

- Never log, print, or return raw API keys. REST output goes through `maskAPIKey`; the `api_key` collection field is `Hidden: true`.
- Agent config file writes go through `internal/agentconfig`'s whitelist + safefile.
- Binding the server to a non-loopback address must keep the terminal-exposure warning.
- `config import` runs inside a transaction and takes a `--backup` (default on) export first — do not bypass either.

## Smoke test reference flow

`task smoke` exercises against a fresh temp `pb_data` and the mock provider on `127.0.0.1:18990`:

`config template` → `config import` → `provider list` → `provider from-preset glm` → `provider model add glm glm-5.3` → `start claude glm --model glm-5.3 --dry-run` → `provider add local` → `test provider local` → `test models local` → `profile add codex-local` → `start codex codex-local --dry-run` → `config export --include-secrets`.

Use it as the integration baseline whenever the data path changes.

## More references

- REST reference: `docs/API.md`
- Per-command docs: `docs/commands/*.md`
- User-facing CLI recipes: `README.md`
- Published docs site: https://variableway.github.io/innate-aiswitcher/
