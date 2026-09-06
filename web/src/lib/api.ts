import type { Agent, AgentConfigFile, AgentConfigInfo, ConfigPreview, ModelsResult, Preset, Profile, Provider, TestResult } from './types'

const API = '/api/aisw'

class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new ApiError((data as { message?: string }).message || res.statusText, res.status)
  }
  return data as T
}

export const api = {
  listProviders: () => request<Provider[]>('/providers'),
  getProvider: (slug: string) => request<Provider>(`/providers/${slug}`),
  createProvider: (p: Partial<Provider>) =>
    request<Provider>('/providers', { method: 'POST', body: JSON.stringify(p) }),
  updateProvider: (slug: string, p: Partial<Provider>) =>
    request<Provider>(`/providers/${slug}`, { method: 'PUT', body: JSON.stringify(p) }),
  deleteProvider: (slug: string) =>
    request<{ ok: boolean }>(`/providers/${slug}`, { method: 'DELETE' }),

  importFromPreset: (presetSlug: string, apiKey: string) =>
    request<Provider>('/providers/from-preset', {
      method: 'POST',
      body: JSON.stringify({ preset_slug: presetSlug, api_key: apiKey }),
    }),

  savePreset: (slug: string) =>
    request<{ ok: boolean; path: string }>('/presets', {
      method: 'POST',
      body: JSON.stringify({ slug }),
    }),
  importPresetFile: (content: string) =>
    request<{ ok: boolean; presets: string[] }>('/presets/import', {
      method: 'POST',
      body: JSON.stringify({ content }),
    }),
  deletePreset: (slug: string) =>
    request<{ ok: boolean }>(`/presets/${slug}`, { method: 'DELETE' }),

  addModel: (slug: string, model: string, isDefault = false) =>
    request<Provider>(`/providers/${slug}/models`, {
      method: 'POST',
      body: JSON.stringify({ model, default: isDefault }),
    }),
  removeModel: (slug: string, model: string) =>
    request<Provider>(`/providers/${slug}/models/${model}`, { method: 'DELETE' }),

  testProvider: (slug: string, model?: string) =>
    request<TestResult>(`/providers/${slug}/test`, {
      method: 'POST',
      body: JSON.stringify(model ? { model } : {}),
    }),
  remoteModels: (slug: string) => request<ModelsResult>(`/providers/${slug}/models`),

  listProfiles: () => request<Profile[]>('/profiles'),
  createProfile: (p: Partial<Profile>) =>
    request<Profile>('/profiles', { method: 'POST', body: JSON.stringify(p) }),
  updateProfile: (slug: string, p: Partial<Profile>) =>
    request<Profile>(`/profiles/${slug}`, { method: 'PUT', body: JSON.stringify(p) }),
  deleteProfile: (slug: string) =>
    request<{ ok: boolean }>(`/profiles/${slug}`, { method: 'DELETE' }),

  listAgents: () => request<Agent[]>('/agents'),
  listPresets: () => request<Preset[]>('/presets'),

  listAgentConfigs: () => request<AgentConfigInfo[]>('/agent-configs'),
  configPreview: (agent: string, provider: string, model: string) =>
    request<ConfigPreview>(`/config-preview?agent=${encodeURIComponent(agent)}&provider=${encodeURIComponent(provider)}&model=${encodeURIComponent(model)}`),
  saveAgentConfig: (agent: string, name: string, content: string) =>
    request<AgentConfigFile>(`/agent-configs/${agent}/${name}`, {
      method: 'PUT',
      body: JSON.stringify({ content }),
    }),
}
