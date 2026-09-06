---
title: "aisw web"
description: "Start the AISwitcher web app — providers, profiles and browser terminal sessions."
---

# aisw web

启动单体 Web 应用：REST API + Web UI（React 19 + TanStack + shadcn/ui）+ **浏览器终端会话**（本地 PTY）。启动后自动在浏览器打开页面。

```bash
aisw web                  # 默认 127.0.0.1:8090，自动打开浏览器
aisw web --no-browser     # 不自动打开浏览器
aisw web --http 127.0.0.1:9090
```

## Web UI 功能

- **Providers**：厂商卡片（协议/可用 Agent/模型徽标）、从预设导入（一把 key）、模型增删（共享 key）、连通性测试；界面默认中文，侧边栏可切换 English
- **Profiles**：Agent↔Provider 绑定，按 Agent 过滤可用 Provider、模型覆盖、默认 Profile
- **Terminal**：浏览器里的本地终端。每个 tab 一个独立 PTY 会话；`Launch agent` 菜单可一键 `aisw start <agent> <provider>`
- **配置文件（Configs）**：直接查看、编辑并保存 claude code / codex / opencode 的本地配置文件（白名单路径，原子写入）

## Terminal 会话（WebSocket → 本地 PTY）

- 端点：`GET /api/aisw/terminal`（WebSocket upgrade），前端为 xterm.js
- 协议：客户端文本帧 = 键入内容（`{"type":"resize","cols":N,"rows":N}` 为控制帧）；二进制帧 = 原始 stdin；服务端二进制帧 = PTY 输出；`{"type":"exit"}` = 会话结束
- 默认 `$SHELL -l`；会话随浏览器断开自动结束
- ⚠️ 终端等于本地 shell 权限。默认绑定 `127.0.0.1`；绑定非回环地址时启动日志会输出警告

## 开发模式

```bash
task web:dev    # Vite dev server（/api 代理到 127.0.0.1:8090）
task serve      # 另一个终端：起 Go API
```

## 打包为单体二进制

```bash
task build:full   # = task web:build（vite build + 同步产物到 internal/webui/dist）+ go build
```

前端产物通过 `go:embed` 嵌入 `internal/webui`，`bin/aisw` 单文件即可运行完整 Web 应用（离线可用）。

## 相关

- [aisw serve](/commands/serve) - 同一服务的 REST 入口（不自动开浏览器）
- [REST API](/API) - `/api/aisw/*`
