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
│   └── tui/           # Interactive terminal UI
├── migrations/        # PocketBase collection migrations
├── docs/              # Documentation
└── Taskfile.yml       # Build tasks
```

## Prerequisites

- Go 1.26+
- [Task](https://taskfile.dev/) (optional but recommended)

## Build

```bash
# Build the CLI binary
task build

# Format code
task fmt

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

2. Register it in the `builders` map:

```go
var builders = map[string]Builder{
    // ... existing adapters
    "my_agent": buildMyAgentPlan,
}
```

3. Add a unit test in `internal/adapter/adapter_test.go`.

4. Add the agent seed in migrations (or ensure `agents` collection has the entry).

## Adding a New Provider Preset

1. Edit `internal/templates/files/provider-presets.toml`.
2. Add a `[[presets]]` entry with `slug`, `name`, and `[[presets.url_options]]`.
3. Run `task test` to verify the preset loads correctly.

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

## REST Development

Start the server for local API work:

```bash
task serve
```

Or directly:

```bash
go run ./cmd/aisw serve --http 127.0.0.1:8090
```

Enable the admin UI:

```bash
go run ./cmd/aisw --admin-ui serve --http 127.0.0.1:8090
```

---

For usage examples, see [USAGE.md](USAGE.md).  
For architecture details, see [SPEC.md](SPEC.md).
