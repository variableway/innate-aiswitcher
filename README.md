# innate-aiswitcher

`innate-aiswitcher` 是一个本地 LLM Provider 切换器，用 Go + PocketBase 管理 SQLite 数据，并在启动 Claude Code、Codex CLI、Gemini CLI、Trae CLI、OpenCode 等 Agent session 时选择本次使用的 Provider/Profile。

核心目标是避免 cc-switch 当前“每个 App 一个 ProviderManager”的重复模型：Provider 是全局共享实体，Agent Adapter 只负责把同一个 Provider 投影成不同 Agent 需要的临时配置或环境变量。

## 架构目标

- `providers`：统一的 LLM Provider 中心表，保存 base URL、API key、协议、默认模型、endpoint overrides、headers/capabilities。
- `agents`：只描述 Agent 本身，例如 `claude`、`codex`、`gemini`、`trae`、`opencode`，以及对应 adapter/binary。
- 内置 Agent 包含 `claude`、`codex`、`gemini`、`kimi`、`trae`、`opencode`。
- `profiles`：Agent + Provider 的轻量绑定，用来保存某个 Agent 的模型覆盖、启动参数、env/config overrides。
- `adapter`：启动时的投影层，不拥有 Provider；同一个 Provider 可被 Claude/Codex/Trae 等 adapter 复用。
- `config.toml`：共享配置镜像，结构是 `[[providers]] + [[profiles]]`，不是按 App 重复 Provider；模板会打包进 `bin/aisw`。
- `PocketBase REST`：SQLite collections 可由 CLI 使用，也可供后续 UI 通过 REST 读取。

## 当前命令

推荐用 Task 管理构建和验证：

```bash
task build      # build bin/aisw
task test       # run scoped unit tests
task verify     # fmt + test + compile + build + go mod verify
task smoke      # run a local CLI smoke test
task serve      # start PocketBase REST server
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
  --base-url https://api.example.com/v1 \
  --api-key-env MINIMAX_API_KEY \
  --protocol openai_chat \
  --model gpt-test \
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
  --model gpt-test
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

启动 PocketBase 服务：

```bash
go run ./cmd/aisw serve --http 127.0.0.1:8090
```

默认 `serve` 不启用 PocketBase admin UI，也不打印 admin install URL。需要后台管理页面时显式开启：

```bash
go run ./cmd/aisw --admin-ui --show-admin-banner serve --http 127.0.0.1:8090
```

可用 REST：

- `GET /api/aisw/health`
- `GET /api/aisw/catalog`
- `GET /api/aisw/providers/{slug}/models`
- `POST /api/aisw/providers/{slug}/test`
- `GET /api/collections/agents/records`
- `GET /api/collections/providers/records`
- `GET /api/collections/profiles/records`

`providers.api_key` 是 PocketBase hidden field，不会从公开只读 API 返回。

## Provider 与 Adapter 解耦

这个项目刻意把 Provider 和 Agent Adapter 解耦：

- Provider 只表达“连接哪个 LLM 服务”：协议、base URL、key、默认模型、endpoint overrides。
- Adapter 只表达“某个 Agent 如何消费 Provider”：Claude 写临时 settings JSON，Codex 写临时 `CODEX_HOME/config.toml` + `auth.json`，其他 OpenAI-compatible Agent 使用 session env。
- Profile 是可选绑定，不复制 Provider；它只保存 Agent 维度的覆盖项。
- 默认模型来自 Provider/template。`test provider` 不会再按协议猜默认模型；没有 `default_model` 时必须显式传 `--model`。
- Adapter 使用 registry map 管理，新增 adapter 时注册 builder，不需要在核心路径追加 `switch` 分支。

## TUI

运行 `go run ./cmd/aisw` 会进入交互界面：

- Start an agent session：选择 Agent 和 Provider/Profile。
- List providers：展示所有 Provider 的 slug、name、protocol、default model、base URL、key 状态。
- Configure provider：从内置模板引导配置 API key、URL format、base URL 和 default model。
- Test provider：用当前 API key/default model 发起最小模型请求；配置后也可以立即测试。

MiniMax 模板提供两个 URL format：OpenAI-compatible 和 Claude Code-compatible。

因此同一个 `minimax` Provider 可以同时用于：

```text
Provider(minimax) -> Claude adapter -> claude --settings /tmp/...
Provider(minimax) -> Codex adapter  -> CODEX_HOME=/tmp/... codex
Provider(minimax) -> Trae adapter   -> OPENAI_API_KEY/OPENAI_BASE_URL trae
```

## 本地验证

当前 MVP 包范围测试：

```bash
task verify
task smoke
```

`task smoke` 会启动本地 mock provider，并执行 provider request test 与 model listing test。

仓库里还有用于参考的外部项目/示例目录，其中部分 Go 示例缺自己的依赖，因此 `go test ./...`、`go build ./...`、`go mod tidy` 会被那些参考目录影响。当前项目包请使用 `Taskfile.yml` 中的 scoped 任务。

完整规格见 [docs/SPEC.md](docs/SPEC.md)。
