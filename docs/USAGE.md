# Usage Guide

This guide covers day-to-day usage of `innate-aiswitcher` (aisw).

## Table of Contents

- [Quick Start](#quick-start)
- [Interactive Mode (TUI)](#interactive-mode-tui)
- [Managing Providers](#managing-providers)
- [Managing Profiles](#managing-profiles)
- [Starting a Session](#starting-a-session)
- [Testing Providers](#testing-providers)
- [Config Import / Export](#config-import--export)
- [REST Server](#rest-server)

## Quick Start

Build the CLI:

```bash
task build
```

Run the interactive TUI:

```bash
./bin/aisw
# or
go run ./cmd/aisw
```

## Interactive Mode (TUI)

The default entrypoint opens an interactive menu:

```bash
go run ./cmd/aisw
```

Options include:

1. **Start an agent session** — pick an Agent and a Provider/Profile.
2. **List providers** — view all configured providers with key status.
3. **Configure provider** — add a provider using built-in presets.
4. **Test provider** — send a minimal request to verify connectivity.

## Managing Providers

### List built-in presets

```bash
aisw provider presets
```

### Add a provider manually

```bash
aisw provider add minimax \
  --base-url https://api.minimax.ai/v1 \
  --api-key sk-xxxxxxxx \
  --protocol openai_chat \
  --model abab6.5s \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models
```

### Add a provider using an environment variable for the key

```bash
aisw provider add openai \
  --base-url https://api.openai.com/v1 \
  --api-key-env OPENAI_API_KEY \
  --protocol openai_chat \
  --model gpt-4o
```

### List configured providers

```bash
aisw provider list
```

### Delete a provider

```bash
aisw provider delete minimax
```

## Managing Profiles

A **Profile** binds an Agent to a Provider with optional overrides (model, args, env).

### Add a profile

```bash
aisw profile add claude-minimax \
  --agent claude \
  --provider minimax \
  --model abab6.5s
```

### List profiles

```bash
aisw profile list
```

## Starting a Session

### Start with a provider directly

```bash
aisw start claude minimax
```

### Start with a profile

```bash
aisw start claude claude-minimax
```

### Dry-run to preview the launch plan

```bash
aisw start claude minimax --dry-run
```

Dry-run prints the JSON launch plan without executing it:

```json
{
  "command": "claude --settings /tmp/aisw-claude-...",
  "cwd": "/current/project",
  "env": {},
  "files": {
    "settings": "/tmp/aisw-claude-..."
  }
}
```

### Pass extra arguments to the agent

```bash
aisw start codex codex-minimax -- --debug
```

## Testing Providers

### Test provider connectivity

Sends a minimal API request using the provider's default model:

```bash
aisw test provider minimax
```

Override the model for the test:

```bash
aisw test provider minimax --model abab6.5s-chat
```

### List available models

```bash
aisw test models minimax
```

## Config Import / Export

### Export config (without secrets)

```bash
aisw config export --path ~/.innate-aiswitcher/config.toml
```

### Export config with secrets

```bash
aisw config export --path ~/.innate-aiswitcher/config.toml --include-secrets
```

### Import config

Imports providers and profiles from a TOML file. A backup is created automatically:

```bash
aisw config import --path ~/.innate-aiswitcher/config.toml
```

Skip the automatic backup:

```bash
aisw config import --path ~/.innate-aiswitcher/config.toml --no-backup
```

### Generate a config template

```bash
aisw config template --path ~/.innate-aiswitcher/config.toml
```

## REST Server

Start the PocketBase-backed REST API:

```bash
aisw serve --http 127.0.0.1:8090
```

Enable the admin UI (available at `/_`):

```bash
aisw --admin-ui serve --http 127.0.0.1:8090
```

### Custom endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/aisw/health` | Service health |
| GET | `/api/aisw/catalog` | Agents and providers (keys redacted) |
| GET | `/api/aisw/providers/{slug}/models` | List models for a provider |
| POST | `/api/aisw/providers/{slug}/test` | Test provider connectivity |

### PocketBase collection endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/collections/agents/records` | List agents |
| GET | `/api/collections/providers/records` | List providers (API key hidden) |
| GET | `/api/collections/profiles/records` | List profiles |

---

For architecture details, see [SPEC.md](SPEC.md).  
For development and contribution guidelines, see [DEVELOPMENT.md](DEVELOPMENT.md).
