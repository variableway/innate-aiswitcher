# REST API Reference

The `aisw serve` command starts a PocketBase-backed HTTP server with custom endpoints for provider management and agent discovery.

## Base URL

```
http://127.0.0.1:8090
```

## Authentication

Public read access is enabled for discovery endpoints.  
`providers.api_key` is a **hidden field** and is never returned by public REST.

## Endpoints

### Health Check

```http
GET /api/aisw/health
```

Returns service health status.

**Response:**

```json
{
  "status": "ok"
}
```

---

### Catalog

```http
GET /api/aisw/catalog
```

Returns agents and providers with API keys redacted.

**Response:**

```json
{
  "agents": [
    {
      "id": "...",
      "slug": "claude",
      "name": "Claude Code",
      "binary": "claude",
      "adapter": "claude",
      "active": true
    }
  ],
  "providers": [
    {
      "id": "...",
      "slug": "minimax",
      "name": "MiniMax",
      "base_url": "https://api.minimax.ai/v1",
      "api_protocol": "openai_chat",
      "default_model": "abab6.5s",
      "active": true
    }
  ]
}
```

---

### List Provider Models

```http
GET /api/aisw/providers/{slug}/models
```

Lists models available from the given provider using its `models` endpoint.

**Response:**

```json
{
  "ok": true,
  "status_code": 200,
  "endpoint": "https://api.minimax.ai/v1/models",
  "models": ["abab6.5s", "abab6.5s-chat"],
  "message": ""
}
```

---

### Test Provider

```http
POST /api/aisw/providers/{slug}/test
Content-Type: application/json
```

Runs the same connectivity check as `aisw test provider {slug}`.

**Request body:**

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
  "endpoint": "https://api.minimax.ai/v1/chat/completions",
  "message": "{\"id\":\"ok\"}"
}
```

---

### PocketBase Collection Endpoints

These follow the standard PocketBase REST API.

#### List Agents

```http
GET /api/collections/agents/records
```

#### List Providers

```http
GET /api/collections/providers/records
```

`api_key` is hidden in the response.

#### List Profiles

```http
GET /api/collections/profiles/records
```

---

## Error Responses

All endpoints return standard HTTP status codes:

| Status | Meaning |
|--------|---------|
| 200 | Success |
| 400 | Bad request (missing params) |
| 404 | Provider/agent/profile not found |
| 500 | Internal server error |

Error bodies follow PocketBase conventions:

```json
{
  "code": 404,
  "message": "The requested resource wasn't found.",
  "data": {}
}
```

---

## CLI to REST Mapping

| CLI Command | REST Equivalent |
|-------------|-----------------|
| `aisw provider list` | `GET /api/collections/providers/records` |
| `aisw test provider {slug}` | `POST /api/aisw/providers/{slug}/test` |
| `aisw test models {slug}` | `GET /api/aisw/providers/{slug}/models` |
| `aisw provider presets` | `GET /api/aisw/catalog` |
