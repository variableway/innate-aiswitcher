# innate-aiswitcher

`innate-aiswitcher` 是一个本地 LLM Provider 切换器，用 Go + PocketBase 管理 SQLite 数据，并在启动 Claude Code、Codex CLI、Gemini CLI、Trae CLI、OpenCode 等 Agent session 时选择本次使用的 Provider/Profile。

核心目标是避免 cc-switch 当前“每个 App 一个 ProviderManager”的重复模型：Provider 是全局共享实体，Agent Adapter 只负责把同一个 Provider 投影成不同 Agent 需要的临时配置或环境变量。

## 架构目标

- `providers`：统一的 LLM Provider 中心表，保存 base URL、API key、协议、默认模型、endpoint overrides、headers/capabilities。
- `agents`：只描述 Agent 本身，例如 `claude`、`codex`、`gemini`、`trae`、`opencode`，以及对应 adapter/binary。
- 内置 Agent 包含 `claude`、`codex`、`gemini`、`kimi`、`trae`、`opencode`、`hermes`、`openclaw`。
- `profiles`：Agent + Provider 的轻量绑定，用来保存某个 Agent 的模型覆盖、启动参数、env/config overrides。
- `adapter`：启动时的投影层，不拥有 Provider；同一个 Provider 可被 Claude/Codex/Trae 等 adapter 复用。
- `config.toml`：共享配置镜像，结构是 `[[providers]] + [[profiles]]`，不是按 App 重复 Provider；模板会打包进 `bin/aisw`。
- `PocketBase REST` + **Web UI**：`aisw serve` 同时提供嵌入式配置页面（`http://127.0.0.1:8090/`）和 `/api/aisw/*` REST API；CLI 与 Web UI 共享同一 SQLite 数据库。

## 当前命令

推荐用 Task 管理构建和验证：

```bash
task build      # build bin/aisw
task test       # run scoped unit tests
task verify     # fmt + test + compile + build + go mod verify
task smoke      # run a local CLI smoke test
task serve      # start REST API + embedded Web UI
```

也可以直接使用 Go 命令运行 CLI，CLI 主入口位于 `cmd/aisw`。

CLI 默认不会因为进程启动就 bootstrap PocketBase。`provider presets`、`config template`、`--help` 这类命令不会创建/open data dir；需要 SQLite collections 的命令才会 lazy 初始化 PocketBase core；只有 `serve` 会启动 HTTP server。

启动交互式选择器：

```bash
go run ./cmd/aisw
```

添加共享 Provider：

```bash
go run ./cmd/aisw provider add minimax \
  --base-url https://api.minimax.chat/v1 \
  --api-key-env MINIMAX_API_KEY \
  --protocol openai_chat \
  --model MiniMax-M3 \
  --endpoint chat_completions=/chat/completions \
  --endpoint models=/models
```

查看内置 Provider 模板：

```bash
go run ./cmd/aisw provider presets
```

写出配置文件模板：

```bash
go run ./cmd/aisw config template --path ~/.innate-aiswitcher/config.toml
```

创建 Agent Profile：

```bash
go run ./cmd/aisw profile add codex-minimax \
  --agent codex \
  --provider minimax \
  --model MiniMax-M3
```

启动一个 session，启动前选择 Provider 或 Profile：

```bash
go run ./cmd/aisw start codex codex-minimax
go run ./cmd/aisw start claude minimax --dry-run
```

测试 Provider API Key：

```bash
go run ./cmd/aisw test provider minimax
go run ./cmd/aisw test models minimax
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

浏览器打开 **http://127.0.0.1:8090/** 可管理 Provider/Profile（添加、编辑、从预设导入、连通性测试）。

默认 `serve` 不启用 PocketBase admin UI。需要 PocketBase 后台管理页面时显式开启：

```bash
go run ./cmd/aisw --admin-ui --show-admin-banner serve --http 127.0.0.1:8090
```

自定义 REST（完整参考见 [docs/API.md](docs/API.md)）：

- `GET /` — 嵌入式 Web UI
- `GET /api/aisw/health`、`GET /api/aisw/catalog`
- `GET/POST/PUT/DELETE /api/aisw/providers[/{slug}]` — Provider CRUD
- `POST /api/aisw/providers/from-preset` — 从内置预设导入
- `GET /api/aisw/providers/{slug}/models`、`POST .../test`
- `GET/POST/PUT/DELETE /api/aisw/profiles[/{slug}]` — Profile CRUD
- `GET /api/aisw/agents`、`GET /api/aisw/presets`
- `GET /api/collections/{agents|providers|profiles}/records` — PocketBase 只读集合

`/api/aisw/providers` 返回的 `api_key` 为掩码形式；PocketBase 集合端点的 `api_key` 为 hidden field，不会返回。

## Provider 与 Adapter 解耦

这个项目刻意把 Provider 和 Agent Adapter 解耦：

- Provider 只表达“连接哪个 LLM 服务”：协议、base URL、key、默认模型、endpoint overrides。
- Adapter 只表达“某个 Agent 如何消费 Provider”：Claude 写临时 settings JSON，Codex 写临时 `CODEX_HOME/config.toml` + `auth.json`，其他 OpenAI-compatible Agent 使用 session env。
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

- Start an agent session：选择 Agent 和 Provider/Profile。
- List providers：展示所有 Provider 的 slug、name、protocol、default model、base URL、key 状态。
- Configure provider：从内置模板引导配置 API key、URL format、base URL 和 default model。
- Test provider：用当前 API key/default model 发起最小模型请求；配置后也可以立即测试。

内置模板为每个 Provider 提供一个或多个 URL format（OpenAI-compatible / Claude Code-compatible / Codex-compatible）。当前内置模板：

| Preset | URL format | Protocol | Base URL | 默认模型 | 投影 slug |
| --- | --- | --- | --- | --- | --- |
| `deepseek` | OpenAI-compatible | `openai_chat` | `https://api.deepseek.com/v1` | `deepseek-v4-flash` | `deepseek` |
| `kimi` | OpenAI-compatible | `openai_chat` | `https://api.moonshot.cn/v1` | `kimi-latest` | `kimi-openai` |
| `minimax` | OpenAI-compatible | `openai_chat` | `https://api.minimaxi.com/v1` | `MiniMax-M3` | `minimax-openai` |
| `minimax` | Claude Code-compatible | `anthropic` | `https://api.minimaxi.com/anthropic` | `MiniMax-M3` | `minimax-claude` |
| `minimax` | Codex-compatible | `openai_responses` | `https://api.minimaxi.com/v1` | `MiniMax-M3` | `minimax-codex` |
| `openai` | OpenAI-compatible | `openai_chat` | `https://api.openai.com/v1` | `gpt-4o` | `openai` |
| `xiaomi` | OpenAI-compatible | `openai_chat` | `https://token-plan-cn.xiaomimimo.com/v1` | `mimo-v2.5-pro` | `xiaomi-openai` |
| `xiaomi` | Claude Code-compatible | `anthropic` | `https://token-plan-cn.xiaomimimo.com/anthropic` | `mimo-v2.5-pro` | `xiaomi-claude` |
| `xiaomi` | Codex-compatible | `openai_responses` | `https://token-plan-cn.xiaomimimo.com/v1` | `mimo-v2.5-pro` | `xiaomi-codex` |
| `anthropic` | Anthropic Messages API | `anthropic` | `https://api.anthropic.com` | `claude-sonnet-4-6-20250715` | `anthropic-claude` |
| `volcengine` | OpenAI-compatible | `openai_chat` | `https://ark.cn-beijing.volces.com/api/v3` | `glm-5.2` | `volcengine-openai` |
| `volcengine` | Claude Code-compatible | `anthropic` | `https://ark.cn-beijing.volces.com/api/plan` | `glm-5.2` | `volcengine-claude` |

> 注意：Kimi 只提供 OpenAI-compatible 接口，不支持 Claude Code / Codex 投影。MiniMax 与 Xiaomi（MiMo）一个 Provider 同时挂多个 URL format，因此同一个 Provider 可以被多个 adapter 复用，例如 `minimax`：

```text
Provider(minimax-claude) -> Claude adapter -> claude --settings /tmp/...
Provider(minimax-codex)  -> Codex adapter  -> CODEX_HOME=/tmp/... codex
Provider(minimax-openai) -> Trae adapter   -> OPENAI_API_KEY/OPENAI_BASE_URL trae
```

## 本地验证

当前 MVP 包范围测试：

```bash
task verify
task smoke
```

`task smoke` 会启动本地 mock provider，并执行 provider request test 与 model listing test。

仓库里还有用于参考的外部项目/示例目录，其中部分 Go 示例缺自己的依赖，因此 `go test ./...`、`go build ./...`、`go mod tidy` 会被那些参考目录影响。当前项目包请使用 `Taskfile.yml` 中的 scoped 任务。

完整规格见 [docs/SPEC.md](docs/SPEC.md)。使用指南见 [docs/USAGE.md](docs/USAGE.md)。

各 Agent（Claude Code、Codex CLI、OpenCode 等）的详细测试指南见 [docs/AGENT_TESTING.md](docs/AGENT_TESTING.md)。
