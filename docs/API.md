# REST API Reference

`aisw serve` 启动 PocketBase 后端，并挂载自定义 `/api/aisw/*` 路由与嵌入式 Web UI。

## Base URL

```
http://127.0.0.1:8090
```

默认仅监听本机。Web UI 入口：`GET /`

### Config Preview

```http
GET /api/aisw/config-preview?agent=claude&provider=glm&model=glm-5.3
```

Projects agent + provider + model through the real launch pipeline (dry-run) and returns the exact config files and env vars the session would use — no process starts. Files carry a `name` key that maps onto the agent-config whitelist for one-click persistence.

### Agent Config Files

```http
GET /api/aisw/agent-configs
```

Returns the whitelisted local config files of claude code (`~/.claude/settings.json`), codex (`~/.codex/config.toml`, `~/.codex/auth.json`) and opencode (`~/.config/opencode/opencode.json`) with their current content.

```http
PUT /api/aisw/agent-configs/{agent}/{name}
```

**Body:** `{"content": "<full file text>"}` — atomically saves to the whitelisted path (0600, parent dirs created). Paths outside the whitelist are rejected. Note `auth.json` contains API keys; the endpoint is meant for the local Web UI.

## Terminal Sessions (WebSocket)

```http
GET /api/aisw/terminal
Connection: Upgrade
Upgrade: websocket
```

Upgrades to a WebSocket bridged to a local PTY shell (`$SHELL -l`). Wire protocol:

- client → server text frames: keystrokes; JSON control frames `{"type":"resize","cols":N,"rows":N}`
- client → server binary frames: raw stdin
- server → client binary frames: raw PTY output
- server → client text frame `{"type":"exit"}`: the shell exited

Each connection is an independent session; closing the socket terminates the shell. The endpoint grants local shell access — keep the server bound to 127.0.0.1 unless you trust the network.

## Authentication

自定义 `/api/aisw/*` 端点**无认证**，面向本地开发使用（默认 `127.0.0.1`）。请勿将未鉴权的 `serve` 暴露到公网。

PocketBase 集合端点为公开只读；匿名 create/update/delete **未启用**。

### API Key 处理

| 端点 | `api_key` 行为 |
|------|----------------|
| `GET /api/aisw/catalog` | 清空为空字符串 |
| `GET/POST/PUT /api/aisw/providers*` | 掩码：`前4位****后4位`（空 key 返回空） |
| `PUT /api/aisw/providers/{slug}` | 请求体 `api_key` 为空时保留数据库中已有 key |
| `GET /api/collections/providers/records` | PocketBase hidden field，不返回 |

---

## Web UI

```http
GET /
GET /style.css
GET /app.js
```

嵌入式静态页面（`internal/webui/static/`），通过 `/api/aisw/*` 完成 Provider/Profile 管理与预设导入。

---

## Discovery

### Health Check

```http
GET /api/aisw/health
```

**Response:**

```json
{
  "ok": true,
  "service": "innate-aiswitcher"
}
```

---

### Catalog

```http
GET /api/aisw/catalog
```

返回 agents 与 providers（`api_key` 已清空）。

---

### List Agents

```http
GET /api/aisw/agents
```

返回全部已 seed 的 agent 记录（只读）。

---

### List Presets

```http
GET /api/aisw/presets
```

返回内置 `provider-presets.toml` 中的预设列表（与 `aisw provider presets` 同源）。

---

## Providers

### List Providers

```http
GET /api/aisw/providers
```

### Get Provider

```http
GET /api/aisw/providers/{slug}
```

### Create Provider

```http
POST /api/aisw/providers
Content-Type: application/json
```

**Body:** `Provider` JSON（`slug`, `name`, `base_url`, `api_key`, `api_protocol`, `default_model`, `headers`, `endpoints`, `capabilities`, `notes`, `active`）

**Response:** `201 Created`，`api_key` 已掩码。

### Update Provider

```http
PUT /api/aisw/providers/{slug}
Content-Type: application/json
```

路径 `{slug}` 与 body 中的 `slug` 应对齐。`api_key` 留空则保留原值。

### Delete Provider

```http
DELETE /api/aisw/providers/{slug}
```

**Response:**

```json
{ "ok": true }
```

### Import from Preset

```http
POST /api/aisw/providers/from-preset
Content-Type: application/json
```

**Body:**

```json
{
  "preset_slug": "glm",
  "api_key": "sk-..."
}
```

根据内置厂商预设生成 vendor provider 并 upsert（与 TUI/CLI 预设投影规则一致）：一把 API key + 按协议划分的 `variants` + 预设 `models` 列表，claude/codex/opencode 均可直接使用。

### Save Provider as Preset

```http
POST /api/aisw/presets
```

**Body:** `{"slug": "volcengine-claude"}` — derives a preset from the stored provider (API key excluded) and writes it to the user presets directory (`~/.innate-aiswitcher/presets/<slug>.toml`).

### Import Presets from TOML Content

```http
POST /api/aisw/presets/import
```

**Body:** `{"content": "<preset TOML text>"}` — parses `[[presets]]` blocks and saves each as a user preset file.

### Delete User Preset

```http
DELETE /api/aisw/presets/{slug}
```

Deletes a user-saved preset file; builtin presets are rejected. `GET /api/aisw/presets` returns builtin and user presets with a `source` field (`builtin` | `user`); user presets override builtin entries of the same slug.

### List Provider Models

```http
GET /api/aisw/providers/{slug}/models
```

与 `aisw test models {slug}` 相同（调用远端 models 端点）。此外可用 `POST /api/aisw/providers/{slug}/models`（body `{"model":"...","default":false}`）向已配置模型列表添加模型、`DELETE /api/aisw/providers/{slug}/models/{model}` 移除模型 —— 模型共享 Provider 已保存的 API key。

**Response:**

```json
{
  "ok": true,
  "status_code": 200,
  "endpoint": "https://api.example.com/v1/models",
  "models": ["model-a", "model-b"],
  "message": ""
}
```

连通失败时 HTTP 状态码为 `502`。

### Test Provider

```http
POST /api/aisw/providers/{slug}/test
Content-Type: application/json
```

与 `aisw test provider {slug}` 相同。

**Body:**

```json
{
  "model": "optional-model-override"
}
```

**Response:**

```json
{
  "ok": true,
  "status_code": 200,
  "endpoint": "https://api.example.com/v1/chat/completions",
  "message": "{\"id\":\"ok\"}"
}
```

连通失败时 HTTP 状态码为 `502`。

---

## Profiles

### List Profiles

```http
GET /api/aisw/profiles
```

### Create Profile

```http
POST /api/aisw/profiles
Content-Type: application/json
```

**Body:** `Profile` JSON（`slug`, `name`, `agent`, `provider`, `model`, `default_args`, `skip_permissions`, `is_default`, `config_overrides`, `env_overrides`）

### Update Profile

```http
PUT /api/aisw/profiles/{slug}
Content-Type: application/json
```

### Delete Profile

```http
DELETE /api/aisw/profiles/{slug}
```

**Response:**

```json
{ "ok": true }
```

---

## PocketBase Collection Endpoints (Read-Only)

标准 PocketBase REST，仅公开读权限：

```http
GET /api/collections/agents/records
GET /api/collections/providers/records
GET /api/collections/profiles/records
```

`providers` 集合的 `api_key` 为 hidden field，响应中不包含。

---

## Error Responses

自定义端点错误体格式：

```json
{
  "ok": false,
  "message": "provider not found"
}
```

| Status | Meaning |
|--------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad request / validation error |
| 404 | Resource not found |
| 502 | Provider connectivity check failed |
| 500 | Internal server error |

PocketBase 集合端点遵循 PocketBase 标准错误格式。

---

## CLI to REST Mapping

| CLI Command | REST Equivalent |
|-------------|-----------------|
| `aisw provider list` | `GET /api/aisw/providers` |
| `aisw provider add` / update | `POST` / `PUT /api/aisw/providers/{slug}` |
| `aisw provider delete` | `DELETE /api/aisw/providers/{slug}` |
| `aisw provider presets` | `GET /api/aisw/presets` |
| `aisw profile list` | `GET /api/aisw/profiles` |
| `aisw profile add` / update | `POST` / `PUT /api/aisw/profiles/{slug}` |
| `aisw test provider {slug}` | `POST /api/aisw/providers/{slug}/test` |
| `aisw test models {slug}` | `GET /api/aisw/providers/{slug}/models` |
| Agents catalog | `GET /api/aisw/agents` 或 `GET /api/collections/agents/records` |
| Full catalog snapshot | `GET /api/aisw/catalog` |
