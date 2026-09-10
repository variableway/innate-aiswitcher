import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'

export type Lang = 'zh' | 'en'

const dict = {
  // shell
  'app.tagline': { zh: '厂商切换器', en: 'provider switcher' },
  'app.agents': { zh: 'claude code · codex · opencode', en: 'claude code · codex · opencode' },
  'nav.agents': { zh: 'Agent', en: 'Agents' },
  'nav.providers': { zh: 'Provider', en: 'Providers' },
  'nav.profiles': { zh: 'Profile', en: 'Profiles' },
  'nav.terminal': { zh: '终端', en: 'Terminal' },
  'nav.configs': { zh: '配置文件', en: 'Configs' },
  'nav.market': { zh: '模型市场', en: 'Model Market' },

  // market page
  'market.title': { zh: '模型市场', en: 'Model Market' },
  'market.subtitle': {
    zh: '从 lobehub 公共目录手工拉取模型参考数据；选中模型可导入厂商 Provider，共享其 API Key。',
    en: 'Manually pull reference data from the public lobehub catalog; import selected models into a vendor provider sharing its API key.',
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
  'market.importSubmit': { zh: '导入', en: 'Import' },
  'market.empty.title': { zh: '暂无目录数据', en: 'No catalog data yet' },
  'market.empty.desc': {
    zh: '从 lobehub 公共目录拉取模型参考数据后，即可浏览并导入。',
    en: 'Fetch the public lobehub catalog to browse and import models.',
  },
  'market.loadFailed': { zh: '目录加载失败', en: 'Failed to load catalog' },
  'market.colModel': { zh: '模型', en: 'Model' },
  'market.colVendor': { zh: '厂商 / 网关', en: 'Vendor / Gateways' },
  'market.colContext': { zh: '上下文', en: 'Context' },
  'market.colAbilities': { zh: '能力', en: 'Abilities' },
  'market.disabled': { zh: '已下线', en: 'offline' },
  'market.noProviders': { zh: '还没有可导入的厂商 Provider', en: 'No vendor providers to import into yet' },

  // agents page
  'agents.title': { zh: 'AI Agent', en: 'AI Agents' },
  'agents.subtitle': {
    zh: '自动检测本机安装的 Agent 工具；每个 Agent 附带配置文件模板，可复制或直接写入磁盘。',
    en: 'Detects installed agent tools; each agent ships settings templates you can copy or write to disk.',
  },
  'agents.installed': { zh: '已安装', en: 'installed' },
  'agents.notInstalled': { zh: '未安装', en: 'not installed' },
  'agents.installHint': { zh: '安装：', en: 'Install:' },
  'agents.templates': { zh: '设置模板', en: 'Settings templates' },
  'agents.copy': { zh: '复制', en: 'Copy' },
  'agents.copied': { zh: '已复制到剪贴板', en: 'Copied to clipboard' },
  'agents.writeDisk': { zh: '写入磁盘', en: 'Write to disk' },
  'agents.written': { zh: '{file} 模板已写入磁盘', en: '{file} template written to disk' },
  'agents.templateHint': {
    zh: '模板中的 <占位符> 需替换为厂商的 Base URL、API Key 与模型；「配置文件」页的生成器可自动填充。',
    en: 'Replace <PLACEHOLDERS> with your vendor base URL, API key and model — or use the Configs page builder to fill them automatically.',
  },

  // providers page
  'providers.title': { zh: 'LLM Provider', en: 'LLM Providers' },
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
  'preset.importFile': { zh: '导入 TOML 文件…', en: 'Import TOML file…' },
  'preset.importDone': { zh: '已从文件导入 {count} 个预设', en: 'Imported {count} preset(s) from file' },
  'preset.importFailed': { zh: '导入失败：{msg}', en: 'Import failed: {msg}' },
  'preset.importedVendor': { zh: '厂商「{name}」已导入', en: 'Vendor "{name}" imported' },
  'preset.importedVendorDesc': {
    zh: '一把 API Key 服务 claude code、codex 和 opencode。',
    en: 'One API key serves claude code, codex and opencode.',
  },
  'preset.importFailedShort': { zh: '导入失败：{msg}', en: 'Import failed: {msg}' },
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
  'profiles.title': { zh: 'Agent Profile', en: 'Agent Profiles' },
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
  'configs.notExists': { zh: '文件尚不存在，保存后将创建。', en: 'File does not exist yet; saving creates it.' },
  'configs.save': { zh: '保存', en: 'Save' },
  'configs.saved': { zh: '{file} 已保存', en: '{file} saved' },
  'configs.reload': { zh: '重载', en: 'Reload' },
  'configs.loadFailed': { zh: '加载配置失败', en: 'Failed to load configs' },
  'configs.hint': {
    zh: '编辑会直接写入磁盘（原子写入，权限 0600）。auth.json 中包含 API Key，请谨慎分享。',
    en: 'Edits write straight to disk (atomic, mode 0600). auth.json contains API keys — share with care.',
  },

  // config preview
  'configs.previewTitle': { zh: '配置生成器', en: 'Config Builder' },
  'configs.previewDesc': {
    zh: '选择 Agent、LLM Provider 与模型，实时生成该组合将要使用的配置文件；可一键写入磁盘。',
    en: 'Pick an agent, provider and model to see the exact config files the session would use — and optionally write them to disk.',
  },
  'configs.agent': { zh: 'Agent', en: 'Agent' },
  'configs.apply': { zh: '写入磁盘', en: 'Write to disk' },
  'configs.applied': { zh: '配置已写入磁盘', en: 'Config written to disk' },
  'configs.applyFailed': { zh: '写入失败：{msg}', en: 'Write failed: {msg}' },
  'configs.envOnly': { zh: '环境变量（启动时注入，无配置文件）', en: 'Env vars (injected at launch, no config file)' },
  'configs.pickProvider': { zh: '选择 Provider', en: 'select provider' },
  'configs.pickModel': { zh: '选择模型', en: 'select model' },
  'configs.noUsableProviders': { zh: '没有支持该 Agent 的 Provider', en: 'No providers serve this agent' },
  'configs.previewError': { zh: '生成失败：{msg}', en: 'Preview failed: {msg}' },
  'configs.diskTitle': { zh: '磁盘上的当前配置', en: 'Current on-disk configs' },

  // model combobox
  'models.fetch': { zh: '从 API 获取模型列表', en: 'Fetch model list from API' },
  'models.fetchFailed': { zh: '获取模型列表失败', en: 'Failed to fetch models' },
  'models.typeOrPick': { zh: '输入或选择模型…', en: 'Type or pick a model…' },
  'models.typeHint': {
    zh: '输入模型名后回车即可使用（自动注册到 Provider）',
    en: 'Type a model name to use it (auto-registers on the provider)',
  },
  'models.configuredGroup': { zh: '已配置', en: 'Configured' },
  'models.remoteGroup': { zh: '来自厂商 API', en: 'From vendor API' },
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
