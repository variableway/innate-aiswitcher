export interface ProviderVariant {
  base_url: string
  endpoints?: Record<string, string>
  capabilities?: Record<string, unknown>
}

export interface Provider {
  id?: string
  slug: string
  name: string
  base_url: string
  api_key?: string
  api_protocol: string
  default_model?: string
  models?: string[]
  variants?: Record<string, ProviderVariant>
  notes?: string
  active: boolean
}

export interface Profile {
  id?: string
  slug: string
  name: string
  agent: string
  provider: string
  model?: string
  default_args?: string
  skip_permissions?: string
  is_default: boolean
}

export interface Agent {
  id?: string
  slug: string
  name: string
  binary: string
  adapter: string
  active: boolean
  skip_permissions_arg?: string
  skip_permissions_default?: boolean
}

export interface PresetVariant {
  protocol: string
  base_url: string
  endpoints?: Record<string, string>
  capabilities?: Record<string, unknown>
}

export interface Preset {
  slug: string
  name: string
  default_model?: string
  models?: string[]
  variants?: PresetVariant[]
  /** "builtin" presets ship in the binary; "user" presets are saved TOML files. */
  source?: 'builtin' | 'user'
}

export interface TestResult {
  ok: boolean
  status_code: number
  endpoint: string
  message: string
}

export interface ModelsResult {
  ok: boolean
  status_code: number
  endpoint: string
  models?: string[]
  message: string
}

/** Agents each wire protocol can serve — mirrors internal/providerconfig. */
export const PROTOCOL_AGENTS: Record<string, string[]> = {
  anthropic: ['claude'],
  openai_responses: ['codex'],
  openai_chat: ['codex', 'opencode'],
}

export function providerProtocols(p: Provider): string[] {
  if (p.variants && Object.keys(p.variants).length > 0) {
    return ['anthropic', 'openai_responses', 'openai_chat'].filter(
      (proto) => proto in p.variants!
    )
  }
  return [p.api_protocol]
}

export function providerAgents(p: Provider): string[] {
  const agents = new Set<string>()
  for (const proto of providerProtocols(p)) {
    for (const a of PROTOCOL_AGENTS[proto] ?? []) agents.add(a)
  }
  return [...agents]
}

export interface AgentConfigFile {
  agent: string
  name: string
  path: string
  lang: 'json' | 'toml'
  label: string
  exists: boolean
  content: string
  editable: boolean
}

export interface AgentConfigInfo {
  agent: string
  name: string
  files: AgentConfigFile[]
}

export interface ConfigPreviewFile {
  name: string
  label: string
  lang: string
  content: string
  writable: boolean
}

export interface ConfigPreview {
  agent: string
  provider: string
  model: string
  files: ConfigPreviewFile[]
  env?: Record<string, string>
}

export interface InstalledAgent {
  slug: string
  name: string
  binary: string
  installed: boolean
  path?: string
  version?: string
}

export interface AgentTemplate {
  name: string
  label: string
  lang: string
  content: string
}

/** Settings templates shipped with the app (placeholders in <ANGLE_BRACKETS>). */
export const AGENT_TEMPLATES: Record<string, AgentTemplate[]> = {
  claude: [
    {
      name: 'settings', label: 'settings.json', lang: 'json',
      content: `{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "<YOUR_API_KEY>",
    "ANTHROPIC_API_KEY": "<YOUR_API_KEY>",
    "ANTHROPIC_BASE_URL": "<PROVIDER_BASE_URL>",
    "ANTHROPIC_MODEL": "<MODEL>",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "<MODEL>",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "<MODEL>",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "<MODEL>"
  }
}
`,
    },
  ],
  codex: [
    {
      name: 'config', label: 'config.toml', lang: 'toml',
      content: `model = "<MODEL>"
model_provider = "<PROVIDER_NAME>"

[model_providers.<PROVIDER_NAME>]
name = "<PROVIDER_NAME>"
base_url = "<PROVIDER_BASE_URL>"
wire_api = "chat"
`,
    },
    {
      name: 'auth', label: 'auth.json', lang: 'json',
      content: `{
  "OPENAI_API_KEY": "<YOUR_API_KEY>"
}
`,
    },
  ],
  opencode: [
    {
      name: 'opencode', label: 'opencode.json', lang: 'json',
      content: `{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "<PROVIDER_KEY>": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "<PROVIDER_NAME>",
      "options": {
        "baseURL": "<PROVIDER_BASE_URL>",
        "apiKey": "<YOUR_API_KEY>"
      },
      "models": {
        "<MODEL>": {}
      }
    }
  }
}
`,
    },
  ],
}
