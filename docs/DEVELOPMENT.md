# Development Guide

This guide is for contributors working on `innate-aiswitcher`.

## Project Structure

```
├── cmd/aisw/          # CLI main entrypoint
├── cmd/mock-provider/ # Local mock LLM server for smoke tests
├── internal/
│   ├── adapter/       # Agent launch plan builders (claude, codex, gemini, openai_env)
│   ├── app/           # PocketBase app bootstrap and custom routes
│   ├── configfile/    # TOML config export/import
│   ├── httpcheck/     # Provider connectivity and model listing
│   ├── safefile/      # Atomic file writes
│   ├── store/         # Typed PocketBase collection wrappers
│   ├── templates/     # Embedded config/preset templates
│   ├── tui/           # Interactive terminal UI
│   └── webui/         # Embedded Web UI (static HTML/CSS/JS via go:embed)
├── docs/              # Markdown 文档源（docmd 发布到 GitHub Pages）
├── docmd.config.json  # docmd 站点配置（summer 模板）
├── package.json       # docmd 依赖与 npm scripts
├── migrations/        # PocketBase collection migrations
└── Taskfile.yml       # Build tasks
```

## Prerequisites

- Go 1.26+
- [Task](https://taskfile.dev/) (optional but recommended)
- Node.js 18+ (only for building/previewing the documentation site)

## Build

```bash
# Build the CLI binary
task build

# Cross-compile for a single target
task build:windows
task build:linux

# Build then install to ~/.local/bin
task install

# Format code
task fmt

# Run go vet
task vet

# Compile all packages
task compile

# Full verification pipeline
task verify
```

> **Note:** Do not use `go test ./...` or `go build ./...` from the repository root. The repo contains reference projects with unrelated Go modules. Use the scoped tasks in `Taskfile.yml`.

## Testing

### Unit Tests

```bash
task test
```

This runs tests for the scoped packages:

```
go test . ./cmd/aisw ./cmd/mock-provider ./internal/... ./migrations
```

### Coverage Areas

| Package | Test Focus |
|---------|-----------|
| `internal/adapter` | Launch plan projection for each agent adapter, settings file generation, cleanup |
| `internal/httpcheck` | Request construction per protocol, endpoint overrides, model extraction |
| `internal/templates` | Embedded template loading, preset resolution, provider generation |
| `internal/safefile` | Atomic writes, permissions, parent directory creation |
| `internal/configfile` | TOML roundtrip, default path formatting |

### Smoke Tests

```bash
task smoke
```

This runs a full integration test:

1. Creates a temporary PocketBase data directory.
2. Starts the mock provider server.
3. Imports a config template.
4. Adds a provider and tests connectivity.
5. Lists models.
6. Creates a profile.
7. Runs a dry-run launch.
8. Exports config.

## Adding a New Adapter

1. Implement a `Builder` function in `internal/adapter/adapter.go`:

```go
func buildMyAgentPlan(ctx BuildContext) (LaunchPlan, func(), error) {
    // ... projection logic
}
```

2. Register it in the `builders` map (or call `adapter.Register("my_agent", buildMyAgentPlan)`):

```go
var builders = map[string]Builder{
    // ... existing adapters
    "my_agent": buildMyAgentPlan,
}
```

3. Add a unit test in `internal/adapter/adapter_test.go` (use `t.TempDir()` for any filesystem side effects; the existing tests use `--dry-run` plans).

4. Add the agent seed in a migration. For new collections follow `migrations/1780565700_init_aisw.go`; for upserting into an existing collection use the `FindFirstRecordByFilter` pattern in `migrations/1780992004_add_skip_permissions_to_agents.go` or `migrations/1780567400_seed_hermes_openclaw_agents.go`.

## Adding a New Provider Preset

1. Edit `internal/templates/files/provider-presets.toml`.
2. Add a `[[presets]]` entry with `slug`, `name`, and one or more `[[presets.url_options]]` blocks. Each option needs `slug`, `label`, `base_url`, `api_protocol` (`openai_chat` | `anthropic` | `openai_responses`), `default_model`, and protocol-specific `endpoints`.
3. If the option targets Claude Code and needs extra session env (long timeouts, traffic flags), set `capabilities = { claude_extra_env = { ... } }` — see the `volcengine` and `xiaomi` claude options for the pattern. For Codex bearer-token auth, set `capabilities = { codex_auth_mode = "experimental_bearer_token" }`.
4. Slug projection: when an option's `slug` differs from the preset `slug` (or a preset has multiple options), `ProviderFromPreset` projects the provider as `preset-slug`-`option-slug` (e.g. `volcengine-claude`, `minimax-codex`). Single-option presets where they match keep the original slug (e.g. `deepseek`, `openai`).
5. Run `task test` to verify the preset loads correctly, and add a preset-shape test in `internal/templates/templates_test.go` (compare `TestProviderPresetsVolcengineURLOptions`).

## Code Style

- Follow standard Go formatting (`gofmt`).
- Use `t.TempDir()` in tests for filesystem operations.
- Prefer table-driven tests for multiple cases.
- Keep adapter logic free of protocol-specific `switch` statements in the core path.

## Commit Messages

Use conventional commit prefixes:

- `feat:` — new feature
- `fix:` — bug fix
- `test:` — test addition or improvement
- `docs:` — documentation only
- `chore:` — maintenance, build, tooling

## REST & Web UI Development

Start the server for local API and Web UI work:

```bash
task serve
```

Or directly:

```bash
go run ./cmd/aisw serve --http 127.0.0.1:8090
```

Open **http://127.0.0.1:8090/** in a browser. The UI calls `/api/aisw/*` endpoints registered in `internal/app/app.go` (`registerProviderRoutes`, `registerProfileRoutes`, `registerAgentRoutes`, `registerPresetRoutes`). Static assets live in `internal/webui/static/` and are served via `internal/webui/webui.go`.

Enable the PocketBase admin UI (separate from the AISwitcher Web UI):

```bash
go run ./cmd/aisw --admin-ui serve --http 127.0.0.1:8090
```

Use `--quiet` on `serve` to suppress HTTP access logs. API reference: [REST API](/API).

## Documentation Site (docmd)

User-facing docs are built with [docmd](https://docs.docmd.io/) from the `docs/` folder and published to GitHub Pages on every push to `main`.

| URL | Purpose |
| --- | --- |
| https://variableway.github.io/innate-aiswitcher/ | Production docs site |
| `http://localhost:3000` | Local preview (`npm run dev`) |

```bash
npm install          # first time only
task docs:dev        # hot-reload dev server
task docs:build      # output static site to site/
```

Configuration lives in `docmd.config.json` (summer template, navigation, `base: /innate-aiswitcher/`). Deployment workflow: `.github/workflows/docs.yml`.

**One-time GitHub setup:** Settings → Pages → Source → **GitHub Actions**.

---

For usage examples, see [Usage Guide](/USAGE).  
For architecture details, see [Architecture](/SPEC).
