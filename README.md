# innate-aiswitcher

`innate-aiswitcher` 是一个本地 LLM Provider 切换器，用 Go + PocketBase 管理 SQLite 数据，并在启动 Claude Code、Codex CLI、OpenCode 等 Agent session 时选择本次使用的 Provider/Profile。

📖 **Documentation:** [variableway.github.io/innate-aiswitcher](https://variableway.github.io/innate-aiswitcher/)

核心目标是避免 cc-switch 当前“每个 App 一个 ProviderManager”的重复模型：Provider 是全局共享实体，Agent Adapter 只负责把同一个 Provider 投影成不同 Agent 需要的临时配置或环境变量。

## 架构目标

- `providers`：厂商级 LLM Provider 中心表（LLM Provider Config）。一个厂商（如 `glm`、`minimax`）一行数据、一把 API key：`models` 列表中的所有模型共享该 key，`variants` 按协议（`anthropic` / `openai_responses` / `openai_chat`）保存各自的 base URL 与端点。
- `agents`：只描述 Agent 本身。当前聚焦 `claude`、`codex`、`opencode` 三个 Agent，其余（gemini/kimi/trae/hermes/openclaw）已移除。
- `profiles`：Agent + Provider 的轻量绑定，用来保存某个 Agent 的模型覆盖、启动参数、env/config overrides。
- `adapter`：启动时的投影层，不拥有 Provider；同一个 Provider 可被 Claude/Codex/Trae 等 adapter 复用。
- `config.toml`：共享配置镜像，结构是 `[[providers]] + [[profiles]]`，不是按 App 重复 Provider；模板会打包进 `bin/aisw`。
- `PocketBase REST` + **Web UI**（React 19 + TanStack + shadcn/ui，构建产物 go:embed 进单体二进制）：`aisw web` / `aisw serve` 提供 Web 页面（Providers/Profiles 管理 + 浏览器终端会话，终端为本地 PTY）和 `/api/aisw/*` REST API；CLI 与 Web UI 共享同一 SQLite 数据库。

## 当前命令

推荐用 Task 管理构建和验证：

```bash
task build       # build bin/aisw
task test        # run scoped unit tests
task verify      # fmt + vet + test + build + go mod verify
task smoke       # run a local CLI smoke test (mock provider)
task serve       # start REST API + embedded Web UI
task web         # aisw web — Web UI + browser terminal sessions (opens browser)
task web:build   # build web/ frontend into internal/webui/dist (go:embed)
task build:full  # web:build + go build — full single binary
task run         # launch the interactive TUI
task install     # build + copy bin/aisw to ~/.local/bin
task docs:dev    # docmd dev server for the documentation site
task docs:build  # build static docs to site/
```

也可以直接使用 Go 命令运行 CLI，CLI 主入口位于 `cmd/aisw`。

CLI 默认不会因为进程启动就 bootstrap PocketBase。`provider presets`、`config template`、`--help` 这类命令不会创建/open data dir；需要 SQLite collections 的命令才会 lazy 初始化 PocketBase core；只有 `serve` 会启动 HTTP server。

启动交互式选择器：

```bash
go run ./cmd/aisw
```

从内置预设一键导入厂商 Provider（CLI / TUI / Web UI 都支持）—— 一把 API key 同时服务 claude code、codex、opencode：

```bash
go run ./cmd/aisw provider from-preset minimax --api-key-env MINIMAX_API_KEY
go run ./cmd/aisw provider from-preset glm --api-key sk-xxx
```

新增模型（与已有模型共享同一把 key，无需再配 key）：

```bash
go run ./cmd/aisw provider model add glm glm-5.3
go run ./cmd/aisw start claude glm --model glm-5.3   # 启动时切模型
```

项目级默认 `.aiswrc`（TOML，可选 `profile` / `agent` / `provider`；之后 `aisw start` 自动套用）：

```toml
# .aiswrc
profile = "codex-glm53"
agent = "codex"
```

```bash
go run ./cmd/aisw start codex               # 读取 .aiswrc
go run ./cmd/aisw start codex --ignore-project   # 跳过 .aiswrc
```

写出配置文件模板：

```bash
go run ./cmd/aisw config template --path ~/.innate-aiswitcher/config.toml
```

创建 Agent Profile（可选；厂商 Provider 本身已可直接用于任一 Agent）：

```bash
go run ./cmd/aisw profile add codex-glm53 \
  --agent codex \
  --provider glm \
  --model glm-5.3
```

启动一个 session，启动前选择 Provider 或 Profile：

```bash
go run ./cmd/aisw start codex codex-glm53
go run ./cmd/aisw start claude glm --dry-run
go run ./cmd/aisw start claude glm --model glm-5.3
```

测试 Provider API Key：

```bash
go run ./cmd/aisw test provider glm
go run ./cmd/aisw test models glm
# 覆盖模型：
go run ./cmd/aisw test provider glm --model glm-5.3
```

导出/导入共享配置：

```bash
go run ./cmd/aisw config export --path ~/.innate-aiswitcher/config.toml
go run ./cmd/aisw config export --path ~/.innate-aiswitcher/config.toml --include-secrets
go run ./cmd/aisw config import --path ~/.innate-aiswitcher/config.toml
go run ./cmd/aisw config import --path ~/.innate-aiswitcher/config.toml --backup-path ~/.innate-aiswitcher/backups/before-import.toml
```

`config import` 默认会先导出一份包含 secret 的备份；如确实不需要，可传 `--no-backup`。导出/模板写入使用临时文件 + rename 的原子写入流程，导入 providers/profiles 时使用 SQLite transaction，失败会回滚。

启动 REST API + Web UI：

```bash
go run ./cmd/aisw serve --http 127.0.0.1:8090
# 或
task serve
```

浏览器打开 **http://127.0.0.1:8090/** 可管理 Provider/Profile（厂商卡片、模型增删共享 key、从预设导入、连通性测试），Terminal 页可开多个本地 PTY 终端并一键 `aisw start <agent> <provider>`。

默认 `serve` 不启用 PocketBase admin UI。需要 PocketBase 后台管理页面时显式开启：

```bash
go run ./cmd/aisw --admin-ui --show-admin-banner serve --http 127.0.0.1:8090
```

自定义 REST（完整参考见 [docs/API.md](docs/API.md)）：

- `GET /` — 嵌入式 Web UI
- `GET /api/aisw/health`、`GET /api/aisw/catalog`
- `GET/POST/PUT/DELETE /api/aisw/providers[/{slug}]` — Provider CRUD
- `POST /api/aisw/providers/from-preset` — 从内置厂商预设导入（一把 key + 全部协议端点）
- `GET /api/aisw/providers/{slug}/models`、`POST/DELETE /api/aisw/providers/{slug}/models[/{model}]`（配置的模型列表）、`POST .../test`
- `GET/POST/PUT/DELETE /api/aisw/profiles[/{slug}]` — Profile CRUD
- `GET /api/aisw/agents`、`GET /api/aisw/presets`
- `GET /api/collections/{agents|providers|profiles}/records` — PocketBase 只读集合

`/api/aisw/providers` 返回的 `api_key` 为掩码形式；PocketBase 集合端点的 `api_key` 为 hidden field，不会返回。

## Provider 与 Adapter 解耦

这个项目刻意把 Provider 和 Agent Adapter 解耦：

- Provider 只表达“连接哪个厂商”：一把 API key、模型列表、按协议划分的端点（variants）。
- Adapter 只表达“某个 Agent 如何消费 Provider”：Claude 写临时 settings JSON（anthropic 变体），Codex 写临时 `CODEX_HOME/config.toml` + `auth.json`（responses/chat 变体），OpenCode 使用 session env（openai_chat 变体）。
- Profile 是可选绑定，不复制 Provider；它只保存 Agent 维度的覆盖项。
- 默认模型来自 Provider/template。`test provider` 不会再按协议猜默认模型；没有 `default_model` 时必须显式传 `--model`。
- Adapter 使用 registry map 管理，新增 adapter 时注册 builder，不需要在核心路径追加 `switch` 分支。

## Web UI

`task serve` 或 `aisw serve` 会在 **http://127.0.0.1:8090/** 提供嵌入式配置页面：

- **Providers**：查看、添加、编辑、删除；从内置预设一键导入；Test 连通性
- **Profiles**：创建 Agent + Provider 绑定；设置默认 Profile；覆盖模型与 CLI 参数

Web UI 与 CLI 共享 `~/.innate-aiswitcher/pb_data/` 中的同一数据库，可交替使用。

## TUI

运行 `go run ./cmd/aisw` 会进入交互界面：

- Start an agent session：选择 Agent，再选择支持该 Agent 的 Provider（厂商行自动匹配协议端点），多模型时可选择本次模型。
- List providers：展示所有 Provider 的 slug、name、协议、模型列表、key 状态。
- Configure provider：从内置厂商模板引导配置一把 API key、默认模型和模型列表。
- Test provider：用当前 API key/default model 发起最小模型请求；配置后也可以立即测试。

内置厂商预设（一个预设 = 一把 API key + 全部协议端点 + 模型列表）：

| Preset | 协议端点 (variants) | 模型 | 服务 Agent |
| --- | --- | --- | --- |
| `glm` (Volcengine Ark) | `anthropic`: `https://ark.cn-beijing.volces.com/api/plan` · `openai_chat`: `https://ark.cn-beijing.volces.com/api/v3` | `glm-5.2`, `glm-5.3` | claude / codex / opencode |
| `minimax` | `anthropic`: `https://api.minimaxi.com/anthropic` · `openai_responses` / `openai_chat`: `https://api.minimaxi.com/v1` | `MiniMax-M3` | claude / codex / opencode |

> codex 优先使用 `openai_responses`，厂商没有该端点时自动回落到 `openai_chat`（wire_api = chat）。以 `minimax` 为例，一把 key 同时驱动三个 Agent：

```text
Provider(minimax, 一个 key)
  ├─ claude   -> anthropic 变体  -> claude --settings /tmp/...
  ├─ codex    -> responses 变体 -> CODEX_HOME=/tmp/... codex
  └─ opencode -> chat 变体      -> OPENAI_API_KEY/OPENAI_BASE_URL opencode
```

新增厂商只需在 `internal/templates/files/provider-presets.toml` 加一个 `[[presets]]` 块，新增模型用 `aisw provider model add`，均无需改代码。

## 本地验证

当前 MVP 包范围测试：

```bash
task verify
task smoke
```

`task smoke` 会启动本地 mock provider，并执行 `config template` → `config import` → `provider presets` → `provider add` → `test provider` → `test models` → `profile add` → `start --dry-run` → `config export --include-secrets` 的端到端流程。

仓库里还有用于参考的外部项目/示例目录，其中部分 Go 示例缺自己的依赖，因此 `go test ./...`、`go build ./...`、`go mod tidy` 会被那些参考目录影响。当前项目包请使用 `Taskfile.yml` 中的 scoped 任务。

完整规格见 [docs/SPEC.md](docs/SPEC.md)（[在线版](https://variableway.github.io/innate-aiswitcher/SPEC/)）。使用指南见 [docs/USAGE.md](docs/USAGE.md)（[在线版](https://variableway.github.io/innate-aiswitcher/USAGE/)）。

各 Agent（Claude Code、Codex CLI、OpenCode 等）的详细测试指南见 [docs/AGENT_TESTING.md](docs/AGENT_TESTING.md)。
