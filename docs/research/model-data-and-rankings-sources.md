# 调研报告：models.dev 模型数据获取 + 模型 Ranking 维度/来源与可信度评估

- 日期：2026-09-20
- 项目：innate-aiswitcher（aisw）
- 范围：不改代码，实测数据源 + ranking 来源调研 + 集成计划草案
- 背景：`internal/modelcatalog` 已消费 models.dev `api.json` 做价格/能力补全；本篇回答两个问题：(1) models.dev 的模型数据如何完整获取；(2) 模型排名的维度/来源有哪些、哪些可信、如何落地。

---

## 调研一：models.dev 数据面实测（2026-09-20）

### 三个 JSON 端点（均免 key、MIT）

| 端点 | 实测大小 | 内容 | 对 aisw 的用途 |
|---|---|---|---|
| `https://models.dev/api.json` | 4.5MB | 222 个 provider × **7870 条** model 记录（per-provider 视角） | 能力/价格/上下文窗口（`modelcatalog.Enrich` 已在用） |
| `https://models.dev/models.json` | 316KB | **414 个** provider 无关的模型条目（key 形如 `zhipuai/glm-5.3`） | 其中 135 条带 benchmark 分数（见下） |
| `https://models.dev/catalog.json` | 4.8MB | `{"models":…, "providers":…}` 两者合体 | 与前两者重复，取数选一个即可 |

仓库 `github.com/anomalyco/models.dev`（SST 作者维护，schema 有 CI 校验，提交频率极高）。另有 `https://models.dev/logos/{provider}.svg` 取厂商 logo。

### `api.json` 字段频次（7870 条全量统计）

`attachment` / `description` / `id` / `last_updated` / `limit` / `modalities` / `name` / `open_weights` / `reasoning` / `release_date` / `tool_call` 全量 7870；`cost` 7453；`temperature` 7368；`family` 7212；`reasoning_options` 5683；`structured_output` 5462；`knowledge`(截止日) 4098；`interleaved` 1105；`provider` 315；`status` 278；`experimental` 58。

**关键事实：`api.json` 完全没有任何 benchmark/ranking 字段。** 排名数据必须从别处来。

### `models.json` 的 `benchmarks`：可展示，不可排序

135/414 条带 `benchmarks` 数组，元素结构：

```json
{"name": "SWE-Bench Pro", "score": 59.5, "metric": "resolve rate",
 "source": "https://github.com/meituan-longcat/longcat-2.0", "date": "2026-06-30",
 "harness": "Harbor", "version": "2.1"}
```

实测覆盖最多的基准：SWE-Bench Pro（63，另有 `SWE Bench Pro`/`SWE-bench Pro` 拼写变体未归一）、Terminal-Bench（49）、SWE-Bench Verified（40）、Humanity's Last Exam（34）、Aider Polyglot（31）、SciCode（30）、Artificial Analysis Coding Index（27）、SWE-Atlas Codebase QnA（26）、Terminal-Bench Hard（22）、GPQA Diamond（20）、AA Coding Agent Index（18）等。

三个使用限制：

1. **`source` 全部指向厂商自己的发布页/仓库** —— 这是「厂商自报分数」，不是独立评测，prompt/scaffold/版本各不相同；
2. 基准名称未归一化（同一名词 3 种拼法、`Humanity's Last Exam` 存在两种撇号字符），聚合前必须做 normalization 表；
3. 覆盖稀疏（135/414），大部分模型没有分数。

结论：适合在模型详情里展示「官方宣称的成绩（带 source+date）」，**不能作为 aisw 内排序键**。

---

## 调研二：Ranking 的维度与来源

### 维度清单（按 aisw 编码场景的相关度排序）

| 维度 | 代表来源 | 说明 |
|---|---|---|
| 编码能力 | AA Coding Index、SWE-bench Verified/Pro、Aider Polyglot、LiveCodeBench | aisw 用户最关心的维度 |
| 综合智能 | AA Intelligence Index、LMArena | 跨任务综合 |
| Agentic/工具调用 | AA Agentic Index、Terminal-Bench、τ-bench、BFCL | coding agent 场景权重高 |
| 人类偏好 | LMArena（Elo 投票） | 真实但反映「聊天讨喜度」 |
| 真实使用量 | OpenRouter rankings（按 token 量） | 开发者用脚投票，流行度≠质量 |
| 价格 | models.dev、OpenRouter、LiteLLM、AA | 已由调研（2026-09-08）覆盖 |
| 速度/延迟 | AA（median tokens/s、TTFT） | 厂商官方数字不可比 |
| 长上下文 | MRCR、Graphwalks | 小众维度 |
| 开放性 | models.dev `open_weights`/`license`/`weights` 字段 | 数据面已具备 |

### 来源逐个评估（按可信度分层）

分层原则：**① 独立第三方统一评测**（同 harness、同题、同 prompt 自己跑）> **② 固定学术基准**（可复现但易被训练污染）> **③ 大规模人类偏好**（真实但可被风格与私测操纵）> **④ 真实用量**（是另一个维度的真实，不是质量）> **⑤ 厂商自报**（不可横比，仅展示）。

| 来源 | 层级 | 获取方式（实测） | 评语 |
|---|---|---|---|
| **Artificial Analysis** | ① | 官方 API `artificialanalysis.ai/api/v2/data/llms/models` **需 key**（实测无 key 返回 401）；**捷径：OpenRouter `/api/v1/models` 免费内嵌了 AA 三指数**（见调研三） | 独立评测 + 实测速度/价格，方法论公开，Intelligence Index 已到 v4；指数版本会调整，需带版本号 |
| **LiveBench** | ①/② | 网页 livebench.ai；**无官方结果 API**；GitHub 仓库是评测代码，license **NOASSERTION**（实测） | 月度换题抗污染、客观计分；自动化抓取不建议（无 API + license 存疑） |
| **SWE-bench Verified / Pro** | ② | 官网/HF | 编码域事实标准；Verified 500 题经人工校验，Pro（Scale）更难。**跨 harness 分数不可比**（SWE-agent vs 各家自报 scaffold） |
| **Terminal-Bench** | ② | 官网/HF | agentic 终端任务，活跃；**版本敏感**（2.0 vs 2.1 分数不能混） |
| **Aider Polyglot** | ② | aider.chat/docs/leaderboards | 编辑式 diff 编码；题集小（~225）方差大、更新慢 |
| **HELM**（斯坦福） | ② | crfm.stanford.edu/helm | 方法论严谨但更新慢、新模型覆盖滞后 |
| **Epoch AI** | ② | epoch.ai | 独立非营利（FrontierMath 等），覆盖窄、学术向 |
| **LMArena** | ③ | **无官方 API**（实测结论）；半官方途径是 Hugging Face Space；第三方 scraper（如 Apify） | 大规模真人投票 Elo，新模型上线快。重大批评见下 |
| **OpenRouter rankings** | ④ | `/api/v1/rankings` **实测 404**，只有网页（openrouter.ai/rankings，第三方镜像 tokenmaxxing.com） | 真实 token 用量排名；但反映 OR 生态（价格敏感、含 `:free` 补贴模型），≠ 最强模型榜 |
| Scale SEAL / Vellum 等 | 私有 | — | 方法论不公开，不可验证 |
| 厂商自报（models.dev `benchmarks` 字段） | ⑤ | models.json | 只做带来源展示 |

### LMArena 的可信度争议（要点，供展示文案参考）

《The Leaderboard Illusion》（arXiv:2504.20879，Cohere Labs/Stanford/MIT/AI2，NeurIPS 2025）：大厂可**私测大量匿名变体只发布最高分的那个**（Meta 被测出预跑约 27 个 Llama 4 变体）、可撤回差分、数据访问不对等；另有长回答/排版风格偏置的既往研究。LMArena 官方博客有反驳，作者利益相关（Cohere 自家也参赛）。工程结论：LMArena 作辅助参考可以，作主排序键不行；若引用需注明「人类偏好，非能力评测」。

### 横向工程原则（落地时必须遵守）

1. **不同来源的分数绝对值不可互比、不可加权平均**——只能各自来源内部排名后并列展示。
2. 同一基准不同版本（Terminal-Bench 2.0/2.1）、不同 harness（SWE-bench 各 scaffold）分数不可横比。
3. 排名随新模型发布以周为单位变动：缓存 TTL ≥24h，展示必须带 as-of 日期与来源徽章。
4. aisw 默认排序建议：**AA Coding Index（编码场景）→ 缺省回落 AA Intelligence Index → 价格（models.dev cost）**。

---

## 调研三：获取通道实测与推荐组合

### 通道清单

| 通道 | 数据 | 成本 | 实测结果 |
|---|---|---|---|
| A. models.dev `api.json` | 能力/价格/上下文 | 免 key | 222 providers / 7870 models（`modelcatalog` 已消费） |
| B. **OpenRouter `/api/v1/models`** | **AA 三指数 + 实时路由价格** | 免 key | 446 个模型，`benchmarks.artificial_analysis` 含 `intelligence_index`/`coding_index`/`agentic_index`；带指数的 132 条，**去掉 `:batch`/`:free` 等变体后缀后 86 个模型**；国产系覆盖良好（glm-5.3 ii=44.8/ci=74.8、kimi-k3 ii=43.6/ci=76.2、qwen3.8-max ii=45.4 等） |
| C. models.dev `models.json` | 厂商自报 benchmark | 免 key | 135 条，带 source/date，仅展示用 |
| D. AA 官方 API | 全量 AA 数据（更全更及时） | 需注册 key（实测 401） | 可选增强 |
| E. LiveBench / LMArena / OR rankings 网页 | 各自榜单 | 无 API | **不建议自动化抓取**（无官方接口、LiveBench license NOASSERTION、LMArena 需 scraper） |

注意：OpenRouter 的价格是**路由价**（含加价，与厂商直连价不同），且模型 id 是 OR 命名空间（`z-ai/glm-5.3`、`:batch`/`:free` 变体），消费前需归一化；models.dev 上的 AA 指数条目（27+18+6 条）同样存在，可与 OR 通道互为校验。

### 集成计划草案（aisw）

1. **`internal/modelcatalog` 增加第二个 fetcher**：拉 OpenRouter `/api/v1/models`，解析 `benchmarks.artificial_analysis`；模型匹配沿用 `vendorAliases` 思路新增 aisw slug → OR 前缀映射（`glm→z-ai`、`kimi→moonshotai`、`deepseek→deepseek`…），并剥离 `:batch`/`:free` 后缀去重。
2. **ModelMeta 扩展分数字段**（数值型，区别于现有自由文本价格）：`{name, source: "artificial_analysis", value, as_of}`，用户自填值不覆盖（沿用 `Enrich` 的「只填空」策略）。
3. **展示**：CLI `provider list`/模型列表按 coding_index 缺省排序（回落 intelligence_index）；TUI/Web UI 加分数列 + 来源徽章 + as-of 日期；模型详情页可挂 models.json 厂商自报 benchmark（带 source 链接，明确标注「厂商宣称」）。
4. **缓存**：与现有 5 分钟进程内缓存一致即可（数据本身周级变动）；离线时优雅降级为无分数。
5. 验收：`task verify` + `task smoke`；新增 Ginkgo spec 覆盖 OR 归一化与「不覆盖用户自填」两条规则。

---

## 执行验证记录（2026-09-20 本机）

```bash
curl -sL https://models.dev/api.json      # 4.5MB, jq: 222 providers, 7870 models
curl -sL https://models.dev/models.json   # 316KB, 414 entries, 135 with benchmarks
curl -sL https://openrouter.ai/api/v1/models   # 446 models; AA 指数 132 条(去重 86)
curl -sI https://artificialanalysis.ai/api/v2/data/llms/models  # 401 API key required
curl -sI https://openrouter.ai/api/v1/rankings                 # 404（无此端点）
curl -sI https://raw.githubusercontent.com/LiveBench/LiveBench/main/data.csv  # 404
```

## 主要来源

- models.dev 实测（2026-09-20）：https://models.dev/api.json 、https://models.dev/models.json 、https://models.dev/catalog.json ；仓库 https://github.com/anomalyco/models.dev （MIT）
- OpenRouter models API 实测（2026-09-20，含 `benchmarks.artificial_analysis` 三指数）：https://openrouter.ai/api/v1/models ；用量榜（仅网页）：https://openrouter.ai/rankings ，第三方镜像 https://tokenmaxxing.com
- Artificial Analysis：https://artificialanalysis.ai （官方 API 实测需 key）
- 《The Leaderboard Illusion》：https://arxiv.org/abs/2504.20879 （Cohere Labs/Stanford/MIT/AI2，NeurIPS 2025）；LMArena 官方回应：https://blog.lmarena.ai
- LiveBench 仓库（license NOASSERTION，实测）：https://github.com/LiveBench/LiveBench ；榜单 https://livebench.ai
- SWE-bench：https://www.swebench.com ；Terminal-Bench：https://www.tbench.ai ；Aider 榜：https://aider.chat/docs/leaderboards.html ；HELM：https://crfm.stanford.edu/helm
- 上一篇相关调研：`docs/research/vendor-models-pricing-and-token-tracking.md`（2026-09-08，价格/能力数据源对比与 usage 追踪）
