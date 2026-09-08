# 调研报告：Vendor 模型列表/价格自动同步 + Agent Session Token 追踪

- 日期：2026-09-08
- 项目：innate-aiswitcher（aisw）
- 范围：不改代码，仅调研与设计草案

---

## 调研一：自动获取最新模型列表与价格

### aisw 现状（代码确认）

- `internal/httpcheck/check.go` 的 `ListModels()` 已能对 OpenAI 兼容（`GET /models`）与 Anthropic（`GET /v1/models`）协议拉取在线模型 ID 列表，但**只有 ID，没有价格/上下文/模态信息**。
- `internal/store/models.go`：`Provider.Models []string` 手工维护；`Provider.ModelMeta map[string]ModelMeta`（`input_price`/`output_price`/`multimodal`/`note`，价格目前是自由文本字符串）为本地手工标注。
- 内置预设（`internal/templates/files/provider-presets.toml`）：`glm`（智谱 bigmodel + 火山方舟 ark 双 variant）、`minimax`、`kimi`（api.moonshot.cn）、`deepseek`、`openai`、`anthropic`、`xiaomi`（mimo token-plan）。

### 候选数据源对比

| 数据源 | API / 获取方式 | License | 更新频率 | aisw 相关厂商覆盖 | 可靠性评估 |
|---|---|---|---|---|---|
| **models.dev** | `GET https://models.dev/api.json`，单文件 JSON（约 4.5MB，213 个 provider），无需 key | **MIT**（GitHub API 确认；仓库已从 sst/models.dev 迁移至 anomalyco/models.dev，旧地址 301 重定向） | 极高：每天多次提交（2026-09-07 当天多次 sync 提交），6.7k stars | ✅ 全部命中：`zhipuai`(15 模型)、`volcengine`(15)、`minimax-cn`(7)、`moonshotai-cn`(4)、`kimi-for-coding`(4)、`deepseek`(3)、`openai`(48)、`anthropic`(14)、`xiaomi`(6) | 社区维护，数据可能滞后官方定价页；但 schema 有 CI 校验，活跃度高 |
| **LiteLLM model_prices_and_context_window.json** | GitHub raw 单文件 JSON（约 2.3MB，3818 条目），key 为 `<provider_prefix>/<model>` | GitHub 标记 **NOASSERTION**（仓库主体 MIT，含企业版目录的特殊条款）；数据文件随主仓库 | 极高：每天多次提交，58k stars | ⚠️ 部分命中：`deepseek/deepseek-chat` 等直连 key 存在（含 `litellm_provider: deepseek`），但国产厂商大量条目以托管平台前缀存在（`azure_ai/FW-GLM-5.2`、`baseten/zai-org/GLM-5` 等 559 条 CN 相关），缺少 `minimax-cn`/`moonshotai-cn` 直连 key 与 `xiaomi/mimo` | 条目字段含 `source` 官方定价页 URL，可追溯；但 key 命名需自行归一化 |
| **OpenRouter** `/api/v1/models` | `GET https://openrouter.ai/api/v1/models`，无需 key，428 个模型 | 专有服务（数据无明确开源 license） | 实时（路由平台一手数据） | ⚠️ 37 个 CN 相关模型（deepseek/moonshotai/minimax/xiaomi 有，zhipu 经 `z-ai` 命名空间）；价格单位是 **per-token USD 字符串**，含 `input_cache_read`/`input_cache_write`/分层 `overrides` | 服务可用性依赖 OpenRouter；价格是 OpenRouter 加价后的路由价，**不等于厂商直连价** |
| **厂商自家 `/v1/models`** | aisw `httpcheck.ListModels` 已实现 | — | 实时一手 | ✅ 所有 OpenAI 兼容厂商 | 只有模型 ID；无价格/上下文；Anthropic 协议厂商端点形状不同；部分厂商需有效 API key 才能调 |
| artificial analysis | 有 API 但需注册 key，非开源 | 专有 | — | 未深入验证 | 未验证 |

### models.dev 数据结构（实测样本）

provider 级：`{id, name, env[], api, doc, npm, models{}}`，如 `zhipuai: {api: "https://open.bigmodel.cn/api/paas/v4", env: ["ZHIPU_API_KEY"], doc: <官方定价页>}`。

model 级（`deepseek/deepseek-v4-flash-vision-exp` 实测）：
```json
{
  "id", "name", "family", "attachment": true,
  "reasoning": true, "tool_call": true, "structured_output": true,
  "release_date": "2026-08-21", "last_updated": "2026-08-21",
  "modalities": {"input": ["text","image"], "output": ["text"]},
  "limit": {"context": 1000000, "output": 384000},
  "status": "beta",
  "cost": {"input": 0.14, "output": 0.28, "reasoning": 0.28, "cache_read": 0.0028}
}
```
`cost` 单位是 **USD / 1M tokens**（与 models.dev 前端表格一致），可直接映射到 `ModelMeta.InputPrice/OutputPrice`。

### 推荐

- **首选：models.dev**。MIT、无需 key、单文件易缓存、schema 稳定且 CI 校验、aisw 全部 7 个内置预设厂商均有覆盖（含火山方舟和小米这两个其他源基本没有的）、`doc` 字段给出官方定价页可作兜底跳转。
- **备选：OpenRouter**（无需 key 的实时 API，适合做「models.dev 里查不到时」的 fallback）；**LiteLLM JSON** 适合作为算钱用的价格参考库（`input_cost_per_token` per-token 数值直接可算），但其 license 标记与 key 命名归一化成本使其不适合作首选模型目录源。
- 厂商自家 `/v1/models`（已有 `httpcheck.ListModels`）作为**账号级事实源**：sync 后用它做 diff（你账号实际可用哪些），与 models.dev 的「目录源」互补。

### 集成设计草案

1. **新包 `internal/modelindex`**：
   - `Fetch(ctx) (Index, error)`：拉取 `https://models.dev/api.json`。
   - 缓存：`~/.innate-aiswitcher/cache/modelsdev.json`，带 `fetched_at`；默认 TTL 24h，`--refresh` 强制。支持 HTTP 条件请求（ETag/If-Modified-Since，models.dev 走 CDN，未验证是否返回 ETag——需实测，不支持则按 TTL）。
   - slug 映射表（aisw preset slug → models.dev provider id，可多个）：
     `glm→[zhipuai, volcengine]`（glm 预设本身是双 variant：bigmodel 与 ark，需按 variant 的 base_url 选源）、`minimax→[minimax-cn]`、`kimi→[moonshotai-cn, kimi-for-coding]`、`deepseek→[deepseek]`、`openai→[openai]`、`anthropic→[anthropic]`、`xiaomi→[xiaomi]`。自定义 provider 按 base_url host 启发式匹配。
2. **CLI 形态**：
   - `aisw provider sync-models [slug]`（缺省全部）：dry-run 默认输出 diff 表（新增/移除/价格变化），`--apply` 写入。
   - 写入策略：新增模型 append 到 `providers.models`；**从不自动删除**现有模型（与近期「safer model removal」方向一致），移除项只在报告中提示；价格写入 `model_meta[model].input_price/output_price`（沿用现有自由文本，格式化如 `$0.14 / 1M`，或后续把 ModelMeta 改为数值字段）；`multimodal = "image" ∈ modalities.input`。
   - `--from-live` 选项：改用 `httpcheck.ListModels` 拉账号真实可用列表做 merge。
3. **Web UI**：providers 页加「Sync models」按钮走同一 store 逻辑。

---

## 调研二：Session Token 消耗追踪

### 各 agent 数据落盘（截至 2026-09-08 可验证的最新状态）

#### 1. Claude Code

- **位置**：`~/.claude/projects/<cwd-编码>/<session-uuid>.jsonl`（本机实测存在，子 agent 在 `<session-uuid>/subagents/agent-*.jsonl`）。
- **格式**：每行一个事件。`type=assistant` 的记录含 `message.usage`（本机实测，version 2.1.220）：
  ```json
  {"input_tokens":1245,"cache_creation_input_tokens":0,"cache_read_input_tokens":22144,"output_tokens":165,"server_tool_use":{...}}
  ```
  另有 `message.model`、`requestId`、`timestamp`、`sessionId`、`cwd`。较新版本还直接在条目上写 `costUSD`（ccusage 文档 2026-05 确认）。
- **⚠️ 重大已知缺陷（2026-02 报告，anthropics/claude-code#28197）**：JSONL 中约 75% 条目的 `input_tokens` 是流式占位值（0 或 1），低估 100-174 倍；`output_tokens` 不含 thinking，低估 10-17 倍；**cache 两个字段是准确的（~1x）**。解析时必须按 `requestId` + `message.id` 去重。此 issue 是否已修复「未验证」——集成时优先用条目上的 `costUSD`（若有），token 数标注为低估风险。
- **官方 statusline 通道（更准确）**：`settings.json` 配 `statusLine.command`，Claude Code 每轮向脚本 stdin 喂 JSON，含 `session_id`、`transcript_path`、`cost.total_cost_usd`、`cost.total_duration_ms`、`context_window.total_input_tokens/total_output_tokens`。statusbar 累计值来自 finalized API 响应，比 JSONL 准确得多——但它是 per-turn 推送，**没有持久化文件**，需要常驻接收方。
- 参考实现：ccusage（ryoppippi/ccusage，MIT）是这个格式的事实标准解析器。

#### 2. Codex CLI

- **位置**：`~/.codex/sessions/YYYY/MM/DD/rollout-<ISO时间>-<uuid>.jsonl`（`CODEX_HOME` 可改根；归档在 `~/.codex/archived_sessions/`；另有 `~/.codex/session_index.jsonl` 索引）。
- **格式**：每行 `{timestamp, type, payload}`。关键事件：
  - `type=session_meta`：`payload.{id, cwd, model_provider, cli_version}`。
  - `type=turn_context`：`payload.model`（当前模型，可能中途切换）。
  - `type=event_msg, payload.type=token_count`：`payload.info.total_token_usage`（**累计值**：`input_tokens/cached_input_tokens/output_tokens/reasoning_output_tokens/total_tokens`）+ `last_token_usage`（单轮增量）+ `model_context_window`。算 session 总量取**最后一条 token_count 的 total_token_usage** 即可；按时间段归因用相邻累计值做 `max(0, cur-prev)` delta。
- **⚠️ 已知缺口**：2026-01 的 openai/codex#9660 报告**交互式 session 不写 token_count**（只有 `codex exec` 非交互写）。但 2026-06 的 issue #27131 引用了当时 rollout 里的 `token_count` 事件与累计值，说明较新版本（0.137.x 时代）交互式已写入。**本机唯一样本（2026-05-24 rollout）确实没有 token_count 事件**，与版本窗口一致。结论：新版可用，旧文件需容忍缺失。
- 参考实现：`ccusage codex`（Beta）已处理多版本格式与 MultiAgent V2 子 agent 前缀回放。

#### 3. OpenCode

- **位置**：`~/.local/share/opencode/opencode.db`（**单一 SQLite**，WAL 模式；`OPENCODE_DATA`/`XDG_DATA_HOME` 可改根；官方命令 `opencode db path` 可查路径）。v1.2.0（2026-02）从旧的 JSON 文件树迁移到 SQLite。
- **格式（本机实测 schema）**：`session` 表直接有列：`cost`、`tokens_input`、`tokens_output`、`tokens_reasoning`、`tokens_cache_read`、`tokens_cache_write`、`model`(JSON，含 `id`/`providerID`)、`directory`、`time_created/updated`。`message`/`part` 表为 `id + session_id + 时间戳 + data(JSON blob)`，assistant message 的 `data.tokens.{input,output,reasoning,cache.read,cache.write}` 与 `data.modelID`。**session 级聚合已被 OpenCode 自己算好，一条只读 SQL 即可**：
  ```sql
  SELECT id, directory, json_extract(model,'$.id'), tokens_input, tokens_output,
         tokens_reasoning, tokens_cache_read, tokens_cache_write, cost
  FROM session ORDER BY time_updated DESC;
  ```
- 读取必须用只读/不可变模式（`file:...?mode=ro&immutable=1`）避免与 WAL 写入冲突。
- **⚠️ 风险**：schema 未公开承诺稳定——2026-07 anomalyco/opencode#36407 记录了一次迁移系统从 drizzle `__drizzle_migrations` 切到自定义 `migration` 表导致旧库直接崩。解析器要按列名存在性做特性探测。

### 统一集成设计草案

1. **新包 `internal/usage`**，每 agent 一个 parser，统一输出：
   ```go
   type SessionUsage struct {
       Agent      string    // claude / codex / opencode
       SessionID  string
       CWD        string
       Model      string
       StartedAt  time.Time
       UpdatedAt  time.Time
       Input, Output, CacheRead, CacheWrite, Reasoning int64
       CostUSD    float64   // 数据源自带则用之，否则按调研一价格表估算
       Source     string    // 文件/DB 路径
       Accuracy   string    // "exact"(opencode/costUSD) / "underestimate"(claude jsonl) 等
   }
   ```
   - claude：扫 `~/.claude/projects/**/*.jsonl`，按 requestId 去重，有 `costUSD` 用 `costUSD`。
   - codex：扫 rollout jsonl，取最后 token_count 累计值；无 token_count 的旧文件跳过并计数。
   - opencode：只读开 sqlite，直读 `session` 表（首选），`message.data` 做 per-turn 明细。
2. **与 launch_history 关联**：`LaunchHistory` 现有 `CWD + Created`（store 层）。agent session 与 launch 记录**没有共享 session id**（aisw 只是拉起终端进程），建议按 `(agent, cwd 归一化, 时间窗口)` 做启发式关联：session 的 StartedAt 落在某次 launch 之后、下一次同 agent+cwd launch 之前即归属。可选增强：launch_history 表加 `session_id` 列，claude 可在启动后扫最新 jsonl 反查（`sessionId` 字段），opencode 可直接记 session row id——但这要求 launch 后回扫，作为二期。
3. **CLI 形态**：
   - `aisw usage [--agent claude|codex|opencode] [--since 7d] [--cwd .]`：表格输出 tokens + 估算成本（成本用调研一的 modelindex 价格或条目自带 costUSD/cost）。
   - `aisw usage --watch`：轮询/tail 数据源（claude jsonl 与 codex rollout 是 append-only，可增量读；opencode 轮询 sqlite），实现准实时跟踪。
4. **算钱**：优先用数据源自带 cost（claude `costUSD`、opencode `session.cost`）；否则用 modelindex/LiteLLM 价格表按四个桶分别计价（cache_read 通常 0.1x input 价，cache_write 1.25x， reasoning 按 output 价——opencode 的 reasoning 单独计数需并入 output 计价）。

### 局限与风险汇总

- 三个 agent 的落盘格式**均无稳定性承诺**，版本漂移是常态（claude 的 usage 占位 bug、codex 的 token_count 时有时无、opencode 的 schema 迁移断裂）。parser 必须特性探测 + 宽容降级，并把 `Accuracy` 暴露给用户。
- claude JSONL 的 input/output token 目前系统性低估（上游 bug），成本数字以 `costUSD`/statusline 为准；statusline 通道虽准确但需注入用户 `~/.claude/settings.json`，侵入性高，建议只做可选项。
- codex 交互式旧 session 无 token 数据，无法回填。
- 订阅制（ChatGPT/Claude Pro）下算出的 USD 是「API 等价名义值」，非真实账单。

---

## 主要来源

调研一：
- models.dev API 实测（2026-09-08）：https://models.dev/api.json ；仓库：https://github.com/anomalyco/models.dev （GitHub API 实测：MIT、pushed 2026-09-07、6769 stars）
- LiteLLM 价格 JSON 实测：https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json ；仓库 https://github.com/BerriAI/litellm （GitHub API 实测：license NOASSERTION、pushed 2026-09-08）
- OpenRouter models API 实测：https://openrouter.ai/api/v1/models
- models.dev 项目介绍：https://www.developersdigest.tech/blog/models-dev-model-routing-infrastructure

调研二：
- Claude Code usage 占位值 bug：https://github.com/anthropics/claude-code/issues/28197 ；分析：https://gille.ai/en/blog/claude-code-jsonl-logs-undercount-tokens/
- ccusage cost modes（costUSD 字段）：https://ccusage.com/guide/cost-modes ；Codex 数据源：https://ccusage.com/guide/codex/
- Claude statusline JSON 字段：https://gist.github.com/AKCodez/ffb420ba6a7662b5c3dda2edce7783de
- Codex token_count 交互式缺口：https://github.com/openai/codex/issues/9660 ；token_count 事件实证：https://github.com/openai/codex/issues/27131
- Codex 会话文件布局：https://inventivehq.com/knowledge-base/openai/how-to-resume-sessions
- OpenCode SQLite 存储与 token 列：https://cheroliv.com/en/blog/2026/0125_audit_conso_opencode_methodologie_comparaison_prix_llm_post.html ；message.data tokens 结构：https://github.com/miiiiiiich/agent-walker/blob/main/docs/opencode.md ；schema 迁移断裂：https://github.com/anomalyco/opencode/issues/36407 ；官方 storage 文档：https://opencode.ai/docs/troubleshooting/
- 本机实测：`~/.claude/projects/**.jsonl`（v2.1.220 usage 字段）、`~/.codex/sessions/2026/05/24/rollout-*.jsonl`（事件类型枚举、无 token_count）、`~/.local/share/opencode/opencode.db`（表与列清单）
