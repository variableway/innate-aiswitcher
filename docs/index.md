---
title: "AISwitcher"
description: "Local LLM Provider switcher for AI coding agents."
---

# AISwitcher

**innate-aiswitcher** (`aisw`) is a local LLM Provider switcher for AI coding agents. It stores a shared Provider catalog in PocketBase/SQLite, then projects the selected Provider into the temporary config that Claude Code, Codex CLI, Gemini CLI, Kimi, Trae, OpenCode, Hermes, or OpenClaw expects at session start.

::: card "Quick Start"
Install a release binary, or build from source:

```bash
task build
task serve
```

Open **http://127.0.0.1:8090/** for the Web UI, or use the CLI:

```bash
aisw provider list
aisw start codex codex-local
```
:::

## Core Concepts

| Concept | Description |
| --- | --- |
| **Provider** | Global LLM connection config — base URL, API key, protocol, default model |
| **Agent** | A coding agent (`claude`, `codex`, `opencode`) with an adapter |
| **Profile** | Optional Agent + Provider binding with per-agent overrides |
| **Adapter** | Launch-time projection — settings JSON, `CODEX_HOME`, or env vars |

Provider and Agent Adapter are **decoupled**: one shared Provider can be projected by many agents without duplicating credentials.

## Documentation

::: grid
== card "Usage Guide"
Day-to-day CLI, Web UI, and Task workflows.

[Read →](/USAGE)

== card "REST API"
Custom `/api/aisw/*` endpoints for providers, profiles, and presets.

[Read →](/API)

== card "Architecture"
Data model, adapter contract, and design spec.

[Read →](/SPEC)
:::

## Get the CLI

Download pre-built binaries from [GitHub Releases](https://github.com/variableway/innate-aiswitcher/releases/latest):

| Platform | Asset |
| --- | --- |
| Windows amd64 | `aisw_windows_amd64.exe` |
| Linux amd64 | `aisw_linux_amd64` |
| macOS Intel | `aisw_darwin_amd64` |
| macOS Apple Silicon | `aisw_darwin_arm64` |
