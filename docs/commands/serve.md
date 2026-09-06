---
title: "aisw serve"
description: "Start the local REST API server + embedded Web UI."
---

# aisw serve

启动本地 REST API 服务器，同时在 `GET /` 提供嵌入式 Web UI（light 主题）。CLI 与 Web UI 共享同一个 SQLite 数据库。

```bash
aisw serve [domain(s)]
```

不传 domain 时默认监听 `127.0.0.1:8090`；传了 domain（用于自动 TLS）则默认 `0.0.0.0:80`/`443`。

## 用法

```bash
# 本地（默认 127.0.0.1:8090）
aisw serve
# 或
task serve

# 自定义地址
aisw serve --http 127.0.0.1:8090

# 自动 TLS（传 domain）
aisw serve example.com

# 启用 PocketBase admin UI / 启动 banner
aisw --admin-ui --show-admin-banner serve --http 127.0.0.1:8090
```

启动后：

```
aisw: data dir ~/.innate-aiswitcher/pb_data
aisw: server listening on http://127.0.0.1:8090
aisw: web UI    http://127.0.0.1:8090/
aisw: REST API  http://127.0.0.1:8090/api/aisw/
```

## Flags

| Flag | 说明 |
|------|------|
| `--http` | HTTP 监听地址（默认 `127.0.0.1:8090`） |
| `--https` | HTTPS 监听地址 |
| `--origins` | CORS 允许来源（默认 `*`） |
| `--quiet` | 关闭 HTTP 访问日志 |

全局 flag（写在 `serve` 前）：`--admin-ui` 启用 `/_` 后台、`--show-admin-banner` 显示 PocketBase 启动 banner 与 admin 安装地址。

## Web UI

`http://127.0.0.1:8090/` 提供 light 主题的配置页面：

- **Providers**：查看/添加/编辑/删除；从内置预设一键导入（From Preset）；Test 连通性。
- **Profiles**：创建 Agent + Provider 绑定；设置默认 Profile；覆盖模型与 CLI 参数。

## REST API

`/api/aisw/*` 自定义路由（完整参考见 [REST API](/API)）：

- Discovery：`GET /api/aisw/health`、`GET /api/aisw/catalog`、`GET /api/aisw/agents`、`GET /api/aisw/presets`
- Provider CRUD：`GET/POST /api/aisw/providers`、`GET/PUT/DELETE /api/aisw/providers/{slug}`、`POST /api/aisw/providers/from-preset`
- Provider ops：`GET /api/aisw/providers/{slug}/models`、`POST /api/aisw/providers/{slug}/test`
- Profile CRUD：`GET/POST /api/aisw/profiles`、`PUT/DELETE /api/aisw/profiles/{slug}`

`/api/aisw/providers*` 返回的 `api_key` 为掩码形式；`catalog` 直接置空。`PUT` 时空 `api_key` 会保留已存的 key。标准 PocketBase 集合只读端点：`GET /api/collections/{agents|providers|profiles}/records`。

## 说明

- `serve` 是唯一启动 HTTP server 的命令；其它命令只 lazy bootstrap SQLite。
- 默认不启用 PocketBase admin UI；需要时显式传 `--admin-ui`。

## 相关

- [REST API](/API) - 全部端点与 CLI 映射
- [aisw provider](/commands/provider) / [aisw profile](/commands/profile) - Web UI 管理的对象
