import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'

export type Lang = 'zh' | 'en'

const dict = {
  // shell
  'app.tagline': { zh: '厂商切换器', en: 'provider switcher' },
  'app.menu': { zh: '打开导航菜单', en: 'Open navigation menu' },
  'theme.toDark': { zh: '切换到暗色模式', en: 'Switch to dark mode' },
  'theme.toLight': { zh: '切换到亮色模式', en: 'Switch to light mode' },
  'nav.agents': { zh: '智能体', en: 'Agents' },
  'nav.providers': { zh: '服务商', en: 'Providers' },
  'nav.profiles': { zh: '配置档', en: 'Profiles' },
  'nav.terminal': { zh: '终端', en: 'Terminal' },
  'nav.configs': { zh: '配置文件', en: 'Configs' },
  'nav.market': { zh: '模型市场', en: 'Model Market' },
  'nav.rankings': { zh: '模型排名', en: 'Model Rankings' },

  // market page
  'market.title': { zh: '模型市场', en: 'Model Market' },
  'market.subtitle': {
    zh: '从 models.dev 开放目录拉取模型参考数据（本地备份，离线可用）；选中模型可导入厂商 Provider，共享其 API Key。',
    en: 'Pull reference data from the open models.dev catalog (backed up locally, works offline); import selected models into a vendor provider sharing its API key.',
  },
  'market.fetch': { zh: '拉取目录', en: 'Fetch Catalog' },
  'market.fetching': { zh: '拉取中…', en: 'Fetching…' },
  'market.fetchDone': { zh: '已拉取 {count} 个模型', en: 'Fetched {count} models' },
  'market.fetchDoneDesc': {
    zh: '{categories} 个厂商分类 · 存储：{storages}',
    en: '{categories} vendor categories · stored to {storages}',
  },
  'market.fetchFailed': { zh: '拉取失败：{msg}', en: 'Fetch failed: {msg}' },
  'market.settings': { zh: '存储设置', en: 'Storage Settings' },
  'market.settingsDesc': {
    zh: '选择拉取的目录数据保存位置；读取始终从所选后端加载。',
    en: 'Choose where fetched catalog data is persisted; reads always load from the selected backend.',
  },
  'market.settingsSaved': { zh: '存储设置已保存', en: 'Storage settings saved' },
  'market.storage': { zh: '存储后端', en: 'Storage backend' },
  'market.storageSqlite': { zh: 'SQLite（PocketBase）', en: 'SQLite (PocketBase)' },
  'market.storageFile': { zh: '文件（~/.innate-aiswitcher/market）', en: 'File (~/.innate-aiswitcher/market)' },
  'market.storageBoth': { zh: '两者同时保存', en: 'Both' },
  'market.save': { zh: '保存', en: 'Save' },
  'market.cancel': { zh: '取消', en: 'Cancel' },
  'market.fetchedAt': { zh: '上次拉取：{time}', en: 'Last fetch: {time}' },
  'market.neverFetched': { zh: '尚未拉取', en: 'Never fetched' },
  'market.allCategories': { zh: '全部分类', en: 'All categories' },
  'market.searchPlaceholder': { zh: '搜索模型…', en: 'Search models…' },
  'market.total': { zh: '{count} 个模型', en: '{count} models' },
  'market.selected': { zh: '已选 {count} 个', en: '{count} selected' },
  'market.import': { zh: '导入到 Provider', en: 'Import to Provider' },
  'market.importPickProvider': { zh: '选择目标厂商 Provider', en: 'Pick the target vendor provider' },
  'market.importHint': {
    zh: '导入的模型加入厂商的模型列表，自动共享该厂商已存的 API Key。',
    en: 'Imported models join the vendor model list and share its stored API key automatically.',
  },
  'market.importDone': { zh: '已导入 {count} 个模型到 {name}', en: 'Imported {count} models into {name}' },
  'market.importDoneDesc': {
    zh: '模型共享厂商 API Key，无需重复配置。',
    en: 'Models share the vendor API key — no extra key setup.',
  },
  'market.importDoneDefault': { zh: '已将 {model} 设为默认模型', en: '{model} is now the default model' },
  'market.importDefault': { zh: '导入后设为默认', en: 'Set as default after import' },
  'market.importNoDefault': { zh: '不设置', en: "Don't set" },
  'market.importDefaultHint': {
    zh: '从导入的模型中选一个作为该 Provider 的默认模型。',
    en: 'Promote one of the imported models to the provider default.',
  },
  'market.importSubmit': { zh: '导入', en: 'Import' },
  'market.empty.title': { zh: '暂无目录数据', en: 'No catalog data yet' },
  'market.empty.desc': {
    zh: '从 models.dev 开放目录拉取模型参考数据后，即可浏览并导入（数据会备份到本地，离线也能浏览）。',
    en: 'Fetch the open models.dev catalog to browse and import models (snapshots are backed up locally and work offline).',
  },
  'market.loadFailed': { zh: '目录加载失败', en: 'Failed to load catalog' },
  'market.colModel': { zh: '模型', en: 'Model' },
  'market.colVendor': { zh: '厂商 / 网关', en: 'Vendor / Gateways' },
  'market.colContext': { zh: '上下文', en: 'Context' },
  'market.colPrice': { zh: '价格', en: 'Price' },
  'market.priceHint': { zh: '输入 / 输出，美元每百万 token', en: 'USD per 1M tokens (input / output)' },
  'market.colAbilities': { zh: '能力', en: 'Abilities' },
  'market.disabled': { zh: '已下线', en: 'offline' },
  'market.noProviders': { zh: '还没有可导入的厂商 Provider', en: 'No vendor providers to import into yet' },

  // rankings page
  'rankings.title': { zh: '模型排名', en: 'Model Rankings' },
  'rankings.subtitle': {
    zh: 'models.dev 开放目录 × Artificial Analysis 独立评测指数（经 OpenRouter 免费转发），按指标排序；同一来源内分数才可比较。',
    en: 'The open models.dev catalog crossed with independent Artificial Analysis indexes (redistributed by OpenRouter), ranked per metric; scores only compare within their own source.',
  },
  'rankings.metricsInfo': { zh: '指标说明', en: 'About metrics' },
  'rankings.metricsInfoDesc': {
    zh: '每个指标的来源、排序方向与可信度说明。',
    en: 'Source, sort direction and reliability notes for every metric.',
  },
  'rankings.searchPlaceholder': { zh: '搜索模型 / 厂商…', en: 'Search models / vendors…' },
  'rankings.total': { zh: '{count} 个条目', en: '{count} entries' },
  'rankings.showing': { zh: '显示前 {count} 个', en: 'showing top {count}' },
  'rankings.fetchedAt': { zh: '数据时间：{time}', en: 'Data as of: {time}' },
  'rankings.sourceDegraded': { zh: '{name} 不可用，分数缺失', en: '{name} unavailable, scores missing' },
  'rankings.loadFailed': { zh: '排名加载失败', en: 'Failed to load rankings' },
  'rankings.empty.title': { zh: '没有匹配的条目', en: 'No matching entries' },
  'rankings.empty.desc': {
    zh: '换个关键词，或切换到其他指标试试。',
    en: 'Try another keyword or switch to a different metric.',
  },
  'rankings.empty.reset': { zh: '清除搜索', en: 'Clear search' },
  'rankings.colModel': { zh: '模型', en: 'Model' },
  'rankings.colVendor': { zh: '厂商', en: 'Vendor' },
  'rankings.colPriceIn': { zh: '输入价', en: 'In $/1M' },
  'rankings.colPriceOut': { zh: '输出价', en: 'Out $/1M' },
  'rankings.colAbilities': { zh: '能力', en: 'Abilities' },
  'rankings.colReleased': { zh: '发布', en: 'Released' },
  'rankings.knowledge': { zh: '知识截止', en: 'Knowledge cutoff' },

  'rankings.metric.coding': { zh: '编码指数', en: 'Coding' },
  'rankings.metric.intelligence': { zh: '智能指数', en: 'Intelligence' },
  'rankings.metric.agentic': { zh: 'Agentic', en: 'Agentic' },
  'rankings.metric.price_input': { zh: '输入价格', en: 'Input price' },
  'rankings.metric.price_output': { zh: '输出价格', en: 'Output price' },
  'rankings.metric.context': { zh: '上下文窗口', en: 'Context' },
  'rankings.metric.output_limit': { zh: '最大输出', en: 'Max output' },
  'rankings.metric.released': { zh: '最新发布', en: 'Newest' },

  'rankings.desc.coding': {
    zh: 'Artificial Analysis 的编码能力指数：独立第三方用统一评测集跑出的分数，经 OpenRouter 模型 API 免费转发。可信度高，适合编码场景选型。',
    en: 'Artificial Analysis coding index: independent third-party scores from a unified evaluation harness, redistributed free via the OpenRouter models API. High reliability for coding-oriented choices.',
  },
  'rankings.desc.intelligence': {
    zh: 'Artificial Analysis 的综合智能指数：跨任务的独立统一评测，反映模型整体能力，经 OpenRouter 免费转发。',
    en: 'Artificial Analysis intelligence index: an independent cross-task unified evaluation of overall model capability, redistributed via OpenRouter.',
  },
  'rankings.desc.agentic': {
    zh: 'Artificial Analysis 的 Agentic/工具调用指数：衡量工具使用与多步任务能力，经 OpenRouter 免费转发。',
    en: 'Artificial Analysis agentic index: tool-use and multi-step task ability, redistributed via OpenRouter.',
  },
  'rankings.desc.price_input': {
    zh: 'models.dev 目录收录的输入价格（美元 / 百万 token），为厂商牌价，非某个网关的转售价。',
    en: 'Listed input price per 1M tokens (USD) from the models.dev directory — vendor list prices, not gateway resale prices.',
  },
  'rankings.desc.price_output': {
    zh: 'models.dev 目录收录的输出价格（美元 / 百万 token），同样为厂商牌价。',
    en: 'Listed output price per 1M tokens (USD) from the models.dev directory.',
  },
  'rankings.desc.context': {
    zh: '上下文窗口大小（models.dev 目录数据）；长上下文任务的硬性规格。',
    en: 'Context window size in tokens (models.dev directory); a hard spec for long-context work.',
  },
  'rankings.desc.output_limit': {
    zh: '单次最大输出 token 数（models.dev 目录数据）。',
    en: 'Maximum output tokens per request (models.dev directory).',
  },
  'rankings.desc.released': {
    zh: '模型发布日期（models.dev 目录数据），新模型通常能力更强但生态验证少。',
    en: 'Model release date (models.dev directory); newer models tend to be stronger but less battle-tested.',
  },
  'rankings.source.artificial_analysis': {
    zh: '独立评测（Artificial Analysis）',
    en: 'Independent evaluation (Artificial Analysis)',
  },
  'rankings.source.models_dev': { zh: '目录数据（models.dev）', en: 'Directory data (models.dev)' },
  'rankings.dir.asc': { zh: '越低越好', en: 'lower is better' },
  'rankings.dir.desc': { zh: '越高越好', en: 'higher is better' },
  'rankings.compareRule': {
    zh: '注意：不同来源的分数不可横向比较或混合平均；排名仅在所选指标内部有效。',
    en: 'Note: scores from different sources are not comparable and must not be averaged together; each ranking is valid only within its own metric.',
  },

  'rankings.ability.reasoning': { zh: '推理', en: 'reasoning' },
  'rankings.ability.tools': { zh: '工具', en: 'tools' },
  'rankings.ability.vision': { zh: '视觉', en: 'vision' },
  'rankings.ability.openWeights': { zh: '开放权重', en: 'open weights' },

  'rankings.vendorBenchTitle': { zh: '厂商自报评测', en: 'Vendor-reported benchmarks' },
  'rankings.vendorBenchDisclaimer': {
    zh: '以下分数来自厂商自己的发布页（非独立评测），各家测试方法与提示词不同，不可横向比较，仅供参考。',
    en: 'These scores come from the vendor\'s own announcements (not independent evaluation); harnesses and prompts differ per vendor, so they are not cross-comparable.',
  },
  'rankings.noBenchmarks': { zh: '该模型暂无厂商自报评测数据', en: 'No vendor-reported benchmarks for this model' },
  'rankings.bench.colName': { zh: '基准', en: 'Benchmark' },
  'rankings.bench.colScore': { zh: '分数', en: 'Score' },
  'rankings.bench.colSource': { zh: '来源', en: 'Source' },

  // common
  'common.cancel': { zh: '取消', en: 'Cancel' },
  'common.confirm': { zh: '确认', en: 'Confirm' },
  'common.delete': { zh: '删除', en: 'Delete' },

  // agents page
  'agents.title': { zh: 'AI Agent', en: 'AI Agents' },
  'agents.subtitle': {
    zh: '自动检测本机安装的 Agent 工具；每个 Agent 附带配置文件模板，可复制或直接写入磁盘。',
    en: 'Detects installed agent tools; each agent ships settings templates you can copy or write to disk.',
  },
  'agents.installed': { zh: '已安装', en: 'installed' },
  'agents.loadFailed': { zh: '加载 Agent 失败', en: 'Failed to load agents' },
  'agents.notInstalled': { zh: '未安装', en: 'not installed' },
  'agents.installHint': { zh: '安装：', en: 'Install:' },
  'agents.installHintUnknown': {
    zh: '请参考该工具的官方文档安装',
    en: 'see the official docs of this tool to install',
  },
  'agents.templates': { zh: '设置模板', en: 'Settings templates' },
  'agents.copy': { zh: '复制', en: 'Copy' },
  'agents.copied': { zh: '已复制到剪贴板', en: 'Copied to clipboard' },
  'agents.writeDisk': { zh: '写入磁盘', en: 'Write to disk' },
  'agents.writeConfirm.title': { zh: '写入配置模板', en: 'Write template to disk' },
  'agents.writeConfirm.desc': {
    zh: '将覆盖磁盘上的以下文件：',
    en: 'This will overwrite the following file on disk:',
  },
  'agents.writeConfirm.confirm': { zh: '写入', en: 'Write' },
  'agents.written': { zh: '{file} 模板已写入磁盘', en: '{file} template written to disk' },
  'agents.templateHint': {
    zh: '模板中的 <占位符> 需替换为厂商的 Base URL、API Key 与模型；「配置文件」页的生成器可自动填充。',
    en: 'Replace <PLACEHOLDERS> with your vendor base URL, API key and model — or use the Configs page builder to fill them automatically.',
  },

  // agent page builder link
  'agents.builder': { zh: '生成配置', en: 'Build config' },
  'agents.builderHint': {
    zh: '选择该 Agent 可用的 Provider 与模型（可从官方拉取清单），一键生成配置文件。',
    en: 'Pick a provider and model available to this agent (fetch the official list), then jump to the config builder.',
  },

  // providers page
  'providers.title': { zh: '模型服务商', en: 'LLM Providers' },
  'providers.subtitle': {
    zh: '每个厂商（LLM Provider）只维护一把 API Key 和模型列表；与具体 Agent 无关。',
    en: 'Each vendor (LLM provider) is just an API key plus its model list — agent-agnostic.',
  },
  'providers.fromPreset': { zh: '从预设导入', en: 'From Preset' },
  'providers.add': { zh: '添加 Provider', en: 'Add Provider' },
  'providers.savePreset': { zh: '存为预设', en: 'Save as preset' },
  'providers.loadFailed': { zh: '加载 Provider 失败', en: 'Failed to load providers' },
  'providers.empty.title': { zh: '还没有 Provider', en: 'No providers configured' },
  'providers.empty.desc': {
    zh: '从预设导入一个厂商（如 MiniMax、GLM），只需填入该厂商的 API Key。',
    en: 'Import a vendor (e.g. MiniMax, GLM) from a preset — you only supply its API key.',
  },
  'providers.empty.cta': { zh: '从预设导入', en: 'Import from Preset' },
  'providers.keySet': { zh: '已配置 Key', en: 'key set' },
  'providers.noKey': { zh: '缺少 Key', en: 'no key' },
  'providers.agents': { zh: '可用 Agent', en: 'agents' },
  'providers.noModels': { zh: '未配置模型', en: 'no models configured' },
  'providers.test': { zh: '测试', en: 'Test' },
  'providers.edit': { zh: '编辑', en: 'Edit' },
  'providers.delete': { zh: '删除', en: 'Delete' },
  'providers.deleteConfirm': {
    zh: '删除 Provider「{name}」？此操作不可撤销。',
    en: 'Delete provider "{name}"? This cannot be undone.',
  },

  // model list
  'models.add': { zh: '添加模型（共享 Key）', en: 'Add model (shares key)' },
  'models.placeholder': { zh: '模型名，如 glm-5.3', en: 'model name, e.g. glm-5.3' },
  'models.addBtn': { zh: '添加', en: 'Add' },
  'models.remove': { zh: '移除模型 {model}', en: 'Remove model {model}' },
  'models.removeDefault.title': { zh: '移除默认模型', en: 'Remove default model' },
  'models.removeDefault.desc': {
    zh: '「{model}」是默认模型，移除后将回退到模型列表中的第一个模型。',
    en: '"{model}" is the default model; removing it falls back to the first model in the list.',
  },
  'models.default': { zh: '默认模型：{model}', en: 'default model: {model}' },
  'models.added': { zh: '模型「{model}」已添加', en: 'Model "{model}" added' },
  'models.addedDesc': { zh: '与 {slug} 的 API Key 共享。', en: 'Shares the {slug} API key.' },
  'models.removed': { zh: '模型「{model}」已移除', en: 'Model "{model}" removed' },

  // provider form
  'providerForm.addTitle': { zh: '添加 Provider', en: 'Add Provider' },
  'providerForm.editTitle': { zh: '编辑 Provider', en: 'Edit Provider' },
  'providerForm.vendorNote': {
    zh: '厂商 Provider —— 各协议端点由预设管理，保持不变。',
    en: 'Vendor provider — per-protocol endpoints are managed by the preset and stay unchanged.',
  },
  'providerForm.singleNote': {
    zh: '一般用「从预设导入」添加厂商（自动带端点）；此处手动填写自定义端点。',
    en: 'Prefer "From Preset" for vendors (endpoints come with it); fill these only for custom endpoints.',
  },
  'providerForm.slug': { zh: 'Slug', en: 'Slug' },
  'providerForm.name': { zh: '显示名称', en: 'Display name' },
  'providerForm.apiKey': { zh: 'API Key', en: 'API key' },
  'providerForm.apiKeySaved': {
    zh: '已保存 —— 留空则保持不变',
    en: 'saved — leave empty to keep',
  },
  'providerForm.apiKeyPlaceholder': {
    zh: '所有模型与 Agent 共用一把 Key',
    en: 'one key for all models & agents',
  },
  'providerForm.apiKeyDesc': {
    zh: '保存在本地；此 Provider 上的所有模型共享。',
    en: 'Stored locally; every model on this provider shares it.',
  },
  'providerForm.models': { zh: '模型', en: 'Models' },
  'providerForm.modelsDesc': { zh: '逗号分隔；全部共享同一把 Key。', en: 'Comma-separated; all share the API key.' },
  'providerForm.defaultModel': { zh: '默认模型', en: 'Default model' },
  'providerForm.defaultModelHint': { zh: '留空取第一个模型', en: 'first model when empty' },
  'providerForm.protocol': { zh: '协议（高级）', en: 'Protocol (advanced)' },
  'providerForm.baseUrl': { zh: '端点 Base URL（高级）', en: 'Endpoint base URL (advanced)' },
  'providerForm.notes': { zh: '备注', en: 'Notes' },
  'providerForm.active': { zh: '启用', en: 'Active' },
  'providerForm.activeDesc': {
    zh: '停用后不参与配置生成与 Agent 启动。',
    en: 'Inactive providers are excluded from config building and agent launches.',
  },
  'providerForm.cancel': { zh: '取消', en: 'Cancel' },
  'providerForm.save': { zh: '保存', en: 'Save' },
  'providerForm.created': { zh: 'Provider 已创建', en: 'Provider created' },
  'providerForm.updated': { zh: 'Provider 已更新', en: 'Provider updated' },
  'providerForm.deleted': { zh: 'Provider 已删除', en: 'Provider deleted' },
  'providerForm.deleteFailed': { zh: '删除失败：{msg}', en: 'Delete failed: {msg}' },

  // preset dialog
  'preset.title': { zh: '从预设导入厂商', en: 'Import vendor from preset' },
  'preset.desc': {
    zh: '一把 API Key 加各协议端点 —— claude code、codex、opencode 立即可用。',
    en: 'One API key plus per-protocol endpoints — claude code, codex and opencode all work immediately.',
  },
  'preset.loading': { zh: '正在加载预设…', en: 'Loading presets…' },
  'preset.builtin': { zh: '内置', en: 'builtin' },
  'preset.user': { zh: '自定义', en: 'custom' },
  'preset.delete': { zh: '删除自定义预设', en: 'Delete custom preset' },
  'preset.deleteConfirm.desc': {
    zh: '删除自定义预设「{name}」？此操作不可撤销，已导入的 Provider 不受影响。',
    en: 'Delete custom preset "{name}"? This cannot be undone; imported providers are not affected.',
  },
  'preset.importFile': { zh: '导入 TOML 文件…', en: 'Import TOML file…' },
  'preset.importDone': { zh: '已从文件导入 {count} 个预设', en: 'Imported {count} preset(s) from file' },
  'preset.importFailed': { zh: '导入失败：{msg}', en: 'Import failed: {msg}' },
  'preset.importedVendor': { zh: '厂商「{name}」已导入', en: 'Vendor "{name}" imported' },
  'preset.importedVendorDesc': {
    zh: '一把 API Key 服务 claude code、codex 和 opencode。',
    en: 'One API key serves claude code, codex and opencode.',
  },
  'preset.importFailedShort': { zh: '导入失败：{msg}', en: 'Import failed: {msg}' },
  'preset.importHint': {
    zh: '需先选择预设并填写 API Key',
    en: 'Select a preset and enter an API key first',
  },
  'preset.apiKey': { zh: 'API Key', en: 'API key' },
  'preset.apiKeyPlaceholder': {
    zh: '所有模型与 Agent 共用一把 Key',
    en: 'one key for every model and agent',
  },
  'preset.apiKeyDesc': { zh: '保存在本地共享 Provider 数据库中。', en: 'Stored locally in the shared provider database.' },
  'preset.cancel': { zh: '取消', en: 'Cancel' },
  'preset.import': { zh: '导入', en: 'Import' },
  'preset.saved': { zh: '已存为预设文件', en: 'Saved as preset file' },
  'preset.deleted': { zh: '预设已删除', en: 'Preset deleted' },

  // test dialog
  'test.title': { zh: '测试 Provider', en: 'Test provider' },
  'test.desc': {
    zh: '使用已保存的 API Key 发送一次最小模型请求。',
    en: "Sends a minimal model request using the provider's stored API key.",
  },
  'test.run': { zh: '运行测试', en: 'Run test' },
  'test.running': { zh: '测试中…', en: 'Testing…' },
  'test.passed': { zh: '测试通过（{code}）', en: 'Test passed ({code})' },
  'test.failed': { zh: '测试失败（状态 {code}）', en: 'Test failed (status {code})' },
  'test.close': { zh: '关闭', en: 'Close' },
  'test.default': { zh: '（默认）', en: ' (default)' },

  // profiles page
  'profiles.title': { zh: 'Agent 配置档', en: 'Agent Profiles' },
  'profiles.subtitle': {
    zh: '可选的 per-Agent 绑定 —— 模型覆盖、启动参数、默认标记。',
    en: 'Optional per-agent bindings — model overrides, launch args, defaults.',
  },
  'profiles.add': { zh: '添加 Profile', en: 'Add Profile' },
  'profiles.loadFailed': { zh: '加载 Profile 失败', en: 'Failed to load profiles' },
  'profiles.empty.title': { zh: '还没有 Profile', en: 'No profiles configured' },
  'profiles.empty.desc': {
    zh: '厂商 Provider 已可直接用于所有 Agent —— Profile 只在需要 per-Agent 覆盖时使用。',
    en: 'Vendor providers already work with every agent directly — profiles are only needed for per-agent overrides.',
  },
  'profiles.default': { zh: '默认', en: 'default' },
  'profiles.agent': { zh: 'Agent', en: 'agent' },
  'profiles.provider': { zh: 'Provider', en: 'provider' },
  'profiles.model': { zh: '模型', en: 'model' },
  'profiles.edit': { zh: '编辑', en: 'Edit' },
  'profiles.delete': { zh: '删除', en: 'Delete' },
  'profiles.deleteConfirm': { zh: '删除 Profile「{name}」？', en: 'Delete profile "{name}"?' },
  'profiles.created': { zh: 'Profile 已创建', en: 'Profile created' },
  'profiles.updated': { zh: 'Profile 已更新', en: 'Profile updated' },
  'profiles.deleted': { zh: 'Profile 已删除', en: 'Profile deleted' },

  // profile form
  'profileForm.addTitle': { zh: '添加 Profile', en: 'Add Profile' },
  'profileForm.editTitle': { zh: '编辑 Profile', en: 'Edit Profile' },
  'profileForm.desc': {
    zh: '将 Agent 绑定到 Provider，可加 per-Agent 覆盖。',
    en: 'Bind an agent to a provider with optional per-agent overrides.',
  },
  'profileForm.name': { zh: '显示名称', en: 'Display name' },
  'profileForm.selectAgent': { zh: '选择 Agent', en: 'select agent' },
  'profileForm.selectProvider': { zh: '选择 Provider', en: 'select provider' },
  'profileForm.providerNote': {
    zh: '仅列出支持 Agent「{agent}」的 Provider。',
    en: 'Only providers serving agent "{agent}" are listed.',
  },
  'profileForm.modelOverride': { zh: '模型覆盖', en: 'Model override' },
  'profileForm.modelPlaceholder': {
    zh: '留空使用 Provider 默认（{model}）',
    en: 'provider default ({model})',
  },
  'profileForm.modelEmpty': { zh: '留空使用 Provider 默认', en: 'leave empty for provider default' },
  'profileForm.args': { zh: '默认参数', en: 'Default args' },
  'profileForm.isDefault': { zh: '设为该 Agent 的默认', en: 'Default for this agent' },
  'profileForm.isDefaultDesc': {
    zh: '`aisw start AGENT` 不带 selector 时使用。',
    en: 'Used when `aisw start AGENT` runs without a selector.',
  },
  'profileForm.cancel': { zh: '取消', en: 'Cancel' },
  'profileForm.save': { zh: '保存', en: 'Save' },

  // agent configs page
  'configs.title': { zh: 'Agent 配置文件', en: 'Agent Config Files' },
  'configs.subtitle': {
    zh: '直接查看、编辑并保存 claude code / codex / opencode 的本地配置文件。',
    en: 'View, edit and save the local config files of claude code / codex / opencode.',
  },
  'configs.tabBuilder': { zh: '生成器', en: 'Builder' },
  'configs.tabDisk': { zh: '磁盘文件', en: 'On-disk files' },
  'configs.notExists': { zh: '文件尚不存在，保存后将创建。', en: 'File does not exist yet; saving creates it.' },
  'configs.save': { zh: '保存', en: 'Save' },
  'configs.saved': { zh: '{file} 已保存', en: '{file} saved' },
  'configs.reload': { zh: '重载', en: 'Reload' },
  'configs.reloadConfirm.title': { zh: '放弃未保存的修改？', en: 'Discard unsaved changes?' },
  'configs.reloadConfirm.desc': {
    zh: '重载将丢弃你对 {file} 的未保存修改。',
    en: 'Reloading discards your unsaved edits to {file}.',
  },
  'configs.loadFailed': { zh: '加载配置失败', en: 'Failed to load configs' },
  'configs.hint': {
    zh: '编辑会直接写入磁盘（原子写入，权限 0600）。auth.json 中包含 API Key，请谨慎分享。',
    en: 'Edits write straight to disk (atomic, mode 0600). auth.json contains API keys — share with care.',
  },
  'configs.showSecrets': { zh: '显示密钥', en: 'Show secrets' },
  'configs.hideSecrets': { zh: '隐藏密钥', en: 'Hide secrets' },
  'configs.maskedHint': {
    zh: '密钥已遮罩（只读），点击眼睛图标显示原文后再编辑。',
    en: 'Secrets are masked (read-only); click the eye icon to reveal before editing.',
  },

  // config preview
  'configs.previewTitle': { zh: '配置生成器', en: 'Config Builder' },
  'configs.previewDesc': {
    zh: '选择 Agent、LLM Provider 与模型，实时生成该组合将要使用的配置文件；可一键写入磁盘。',
    en: 'Pick an agent, provider and model to see the exact config files the session would use — and optionally write them to disk.',
  },
  'configs.agent': { zh: 'Agent', en: 'Agent' },
  'configs.apply': { zh: '写入磁盘', en: 'Write to disk' },
  'configs.applyConfirm.title': { zh: '写入配置到磁盘', en: 'Write config to disk' },
  'configs.applyConfirm.desc': {
    zh: '将覆盖以下文件（原子写入，权限 0600）：',
    en: 'The following files will be overwritten (atomic, mode 0600):',
  },
  'configs.applied': { zh: '配置已写入磁盘', en: 'Config written to disk' },
  'configs.applyFailed': { zh: '写入失败：{msg}', en: 'Write failed: {msg}' },
  'configs.envOnly': { zh: '环境变量（启动时注入，无配置文件）', en: 'Env vars (injected at launch, no config file)' },
  'configs.pickProvider': { zh: '选择 Provider', en: 'select provider' },
  'configs.pickModel': { zh: '选择模型', en: 'select model' },
  'configs.noUsableProviders': { zh: '没有支持该 Agent 的 Provider', en: 'No providers serve this agent' },
  'configs.previewError': { zh: '生成失败：{msg}', en: 'Preview failed: {msg}' },
  'configs.diskTitle': { zh: '磁盘上的当前配置', en: 'Current on-disk configs' },

  // model meta editor
  'meta.title': { zh: '模型信息', en: 'Model info' },
  'meta.inputPrice': { zh: '输入价格', en: 'Input price' },
  'meta.outputPrice': { zh: '输出价格', en: 'Output price' },
  'meta.priceHint': { zh: '自由填写，如 ¥4 / 1M tokens', en: 'free-form, e.g. $2.5 / 1M tokens' },
  'meta.multimodal': { zh: '多模态（图片/视觉）', en: 'Multimodal (vision)' },
  'meta.note': { zh: '备注', en: 'Note' },
  'meta.save': { zh: '保存', en: 'Save' },
  'meta.saved': { zh: '模型信息已保存', en: 'Model info saved' },
  'meta.makeDefault': { zh: '设为默认', en: 'Set as default' },
  'meta.madeDefault': { zh: '已将 {model} 设为默认模型', en: '{model} is now the default model' },
  'meta.confirmDelete': { zh: '确认删除', en: 'Confirm' },
  'meta.deleteHint': { zh: '再点一次删除 {model}', en: 'Click again to remove {model}' },
  'meta.editHint': { zh: '点击模型名编辑价格 / 多模态等信息', en: 'Click a model to edit pricing / modality' },

  // model catalog enrichment
  'models.enrich': { zh: '从模型库补全', en: 'Enrich from catalog' },
  'models.enrichDone': {
    zh: '已补全 {updated}/{matched} 个模型的信息（价格 / 多模态 / 上下文）',
    en: 'Enriched {updated}/{matched} models (pricing / modality / context)',
  },
  'models.enrichMissed': {
    zh: '未收录：{missed}',
    en: 'Not in catalog: {missed}',
  },

  // model combobox
  'models.fetch': { zh: '从 API 获取模型列表', en: 'Fetch model list from API' },
  'models.fetchFailed': { zh: '获取模型列表失败', en: 'Failed to fetch models' },
  'models.usedLocalCatalog': { zh: '已使用本地模型目录', en: 'Using the local model catalog' },
  'models.usedLocalCatalogDesc': {
    zh: '厂商接口不可用或为空，已从本地 models.dev 快照读取。',
    en: 'Vendor API unreachable or empty; served from the local models.dev snapshot.',
  },
  'models.typeOrPick': { zh: '输入或选择模型…', en: 'Type or pick a model…' },
  'models.typeHint': {
    zh: '输入模型名后回车即可使用（自动注册到 Provider）',
    en: 'Type a model name to use it (auto-registers on the provider)',
  },
  'models.configuredGroup': { zh: '已配置', en: 'Configured' },
  'models.remoteGroup': { zh: '可添加（厂商 / 目录）', en: 'Available (vendor / catalog)' },
  'models.useAndAdd': { zh: '使用并添加', en: 'use & add' },

  // terminal page
  'terminal.title': { zh: '终端会话', en: 'Terminal Sessions' },
  'terminal.subtitle': {
    zh: '本地 PTY 终端 —— 用任意 Provider 启动任意 Agent。',
    en: 'Local PTY shells — launch any agent against any provider.',
  },
  'terminal.launch': { zh: '启动 Agent', en: 'Launch agent' },
  'terminal.new': { zh: '新建终端', en: 'New Terminal' },
  'terminal.plainShell': { zh: '纯 Shell', en: 'Plain shell' },
  'terminal.noProviders': { zh: '尚未配置 Provider', en: 'No providers configured' },
  'terminal.empty.desc': {
    zh: '还没有终端会话。打开一个 Shell，或针对某个 Provider 启动 Agent 会话 —— 每个 tab 都是独立的本地 PTY。',
    en: 'No terminal sessions yet. Open a shell or launch an agent session against a provider — each tab runs a separate local PTY.',
  },
  'terminal.open': { zh: '打开终端', en: 'Open terminal' },
  'terminal.close': { zh: '关闭 {name}', en: 'Close {name}' },
  'terminal.closeConfirm.title': { zh: '关闭终端会话', en: 'Close terminal session' },
  'terminal.closeConfirm.desc': {
    zh: '「{name}」中的进程仍在运行，关闭将终止该进程。',
    en: 'A process in "{name}" is still running; closing will terminate it.',
  },
  'terminal.closeConfirm.confirm': { zh: '关闭并终止', en: 'Close & kill' },
  'terminal.closed': { zh: '—— 会话已结束 ——', en: '—— session closed ——' },
} as const

export type TranslationKey = keyof typeof dict

const LangContext = createContext<{
  lang: Lang
  setLang: (lang: Lang) => void
  t: (key: TranslationKey, vars?: Record<string, string | number>) => string
}>({ lang: 'zh', setLang: () => {}, t: (key) => String(key) })

function detectLang(): Lang {
  const stored = localStorage.getItem('aisw.lang')
  if (stored === 'zh' || stored === 'en') return stored
  // Chinese is the default; only users of other locales get English.
  return 'zh'
}

export function LangProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(detectLang)

  useEffect(() => {
    document.documentElement.lang = lang === 'zh' ? 'zh-CN' : 'en'
  }, [lang])

  const setLang = (next: Lang) => {
    localStorage.setItem('aisw.lang', next)
    setLangState(next)
  }

  const t = (key: TranslationKey, vars?: Record<string, string | number>) => {
    let text: string = dict[key]?.[lang] ?? String(key)
    if (vars) {
      for (const [name, value] of Object.entries(vars)) {
        text = text.replaceAll(`{${name}}`, String(value))
      }
    }
    return text
  }

  return <LangContext.Provider value={{ lang, setLang, t }}>{children}</LangContext.Provider>
}

export function useI18n() {
  return useContext(LangContext)
}
