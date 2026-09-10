import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { api } from './api'
import { useI18n } from './i18n'

export const queryKeys = {
  providers: ['providers'] as const,
  profiles: ['profiles'] as const,
  agents: ['agents'] as const,
  presets: ['presets'] as const,
  market: (category: string, q: string) => ['market', category, q] as const,
  marketCategories: ['market-categories'] as const,
  marketSettings: ['market-settings'] as const,
}

export function useProviders() {
  return useQuery({ queryKey: queryKeys.providers, queryFn: api.listProviders })
}

export function useProfiles() {
  return useQuery({ queryKey: queryKeys.profiles, queryFn: api.listProfiles })
}

export function useAgents() {
  return useQuery({ queryKey: queryKeys.agents, queryFn: api.listAgents })
}

export function usePresets() {
  return useQuery({ queryKey: queryKeys.presets, queryFn: api.listPresets })
}

function useInvalidate() {
  const qc = useQueryClient()
  return (...keys: (readonly unknown[])[]) => {
    for (const key of keys) void qc.invalidateQueries({ queryKey: key })
  }
}

export function useSaveProvider(slugOrig: string | null) {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.createProvider>[0]) =>
      slugOrig ? api.updateProvider(slugOrig, input) : api.createProvider(input),
    onSuccess: () => {
      toast.success(slugOrig ? t('providerForm.updated') : t('providerForm.created'))
      invalidate(queryKeys.providers, queryKeys.profiles)
    },
    onError: (err: Error) => toast.error(err.message),
  })
}

export function useDeleteProvider() {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: api.deleteProvider,
    onSuccess: () => {
      toast.success(t('providerForm.deleted'))
      invalidate(queryKeys.providers, queryKeys.profiles)
    },
    onError: (err: Error) => toast.error(t('providerForm.deleteFailed', { msg: err.message })),
  })
}

export function useImportPreset() {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: ({ slug, apiKey }: { slug: string; apiKey: string }) =>
      api.importFromPreset(slug, apiKey),
    onSuccess: (provider) => {
      toast.success(t('preset.importedVendor', { name: provider.name }), {
        description: t('preset.importedVendorDesc'),
      })
      invalidate(queryKeys.providers, queryKeys.profiles)
    },
    onError: (err: Error) => toast.error(t('preset.importFailedShort', { msg: err.message })),
  })
}

export function useSavePreset() {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: api.savePreset,
    onSuccess: () => {
      toast.success(t('preset.saved'))
      invalidate(queryKeys.presets)
    },
    onError: (err: Error) => toast.error(err.message),
  })
}

export function useImportPresetFile() {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: api.importPresetFile,
    onSuccess: (result) => {
      toast.success(t('preset.importDone', { count: result.presets.length }))
      invalidate(queryKeys.presets)
    },
    onError: (err: Error) => toast.error(t('preset.importFailed', { msg: err.message })),
  })
}

export function useDeletePreset() {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: api.deletePreset,
    onSuccess: () => {
      toast.success(t('preset.deleted'))
      invalidate(queryKeys.presets)
    },
    onError: (err: Error) => toast.error(err.message),
  })
}

export function useAddModel(slug: string) {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: ({ model, isDefault }: { model: string; isDefault: boolean }) =>
      api.addModel(slug, model, isDefault),
    onSuccess: (provider, { model }) => {
      toast.success(t('models.added', { model }), {
        description: t('models.addedDesc', { slug: provider.slug }),
      })
      invalidate(queryKeys.providers)
    },
    onError: (err: Error) => toast.error(err.message),
  })
}

export function useRemoveModel(slug: string) {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: (model: string) => api.removeModel(slug, model),
    onSuccess: (_provider, model) => {
      toast.success(t('models.removed', { model }))
      invalidate(queryKeys.providers)
    },
    onError: (err: Error) => toast.error(err.message),
  })
}

export function useTestProvider() {
  const { t } = useI18n()
  return useMutation({
    mutationFn: ({ slug, model }: { slug: string; model?: string }) =>
      api.testProvider(slug, model),
    onSuccess: (result) => {
      if (!result.ok) {
        toast.error(t('test.failed', { code: result.status_code }))
      }
    },
    onError: (err: Error) => toast.error(err.message),
  })
}

export function useSaveProfile(slugOrig: string | null) {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: (input: Parameters<typeof api.createProfile>[0]) =>
      slugOrig ? api.updateProfile(slugOrig, input) : api.createProfile(input),
    onSuccess: () => {
      toast.success(slugOrig ? t('profiles.updated') : t('profiles.created'))
      invalidate(queryKeys.profiles)
    },
    onError: (err: Error) => toast.error(err.message),
  })
}

export function useDeleteProfile() {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: api.deleteProfile,
    onSuccess: () => {
      toast.success(t('profiles.deleted'))
      invalidate(queryKeys.profiles)
    },
    onError: (err: Error) => toast.error(t('providerForm.deleteFailed', { msg: err.message })),
  })
}

export function useMarketModels(category: string, q: string) {
  return useQuery({
    queryKey: queryKeys.market(category, q),
    queryFn: () => api.listMarketModels(category, q),
    placeholderData: (prev) => prev,
  })
}

export function useMarketCategories() {
  return useQuery({ queryKey: queryKeys.marketCategories, queryFn: api.listMarketCategories })
}

export function useFetchMarket() {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: api.fetchMarket,
    onSuccess: (result) => {
      toast.success(t('market.fetchDone', { count: result.fetched }), {
        description: t('market.fetchDoneDesc', {
          categories: result.categories,
          storages: result.storages.join(', '),
        }),
      })
      invalidate(queryKeys.marketCategories, ['market'] as const)
    },
    onError: (err: Error) => toast.error(t('market.fetchFailed', { msg: err.message })),
  })
}

export function useSaveMarketSettings() {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: api.saveMarketSettings,
    onSuccess: () => {
      toast.success(t('market.settingsSaved'))
      invalidate(queryKeys.marketCategories, ['market'] as const)
    },
    onError: (err: Error) => toast.error(err.message),
  })
}

export function useImportMarketModels() {
  const invalidate = useInvalidate()
  const { t } = useI18n()
  return useMutation({
    mutationFn: ({ provider, models }: { provider: string; models: string[] }) =>
      api.importMarketModels(provider, models),
    onSuccess: (provider, { models }) => {
      toast.success(t('market.importDone', { count: models.length, name: provider.name }), {
        description: t('market.importDoneDesc'),
      })
      invalidate(queryKeys.providers)
    },
    onError: (err: Error) => toast.error(err.message),
  })
}
