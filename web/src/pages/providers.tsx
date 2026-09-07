import { useState } from 'react'
import { BookmarkPlus, KeyRound, Pencil, Plus, Sparkles, Trash2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { TestDialog } from '@/components/providers/test-dialog'
import { ModelList } from '@/components/providers/model-list'
import { PresetImportDialog } from '@/components/providers/preset-import'
import { ProviderFormDialog } from '@/components/providers/provider-form'
import { useDeleteProvider, useProviders, useSavePreset } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import type { Provider } from '@/lib/types'

export function ProvidersPage() {
  const { data: providers, isPending, isError, error } = useProviders()
  const [formOpen, setFormOpen] = useState(false)
  const [presetOpen, setPresetOpen] = useState(false)
  const [editing, setEditing] = useState<Provider | null>(null)
  const { t } = useI18n()

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header className="border-b flex items-center justify-between px-6 py-4">
        <div className="flex flex-col gap-0.5">
          <h1 className="text-lg font-semibold">{t('providers.title')}</h1>
          <p className="text-muted-foreground text-sm">{t('providers.subtitle')}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setPresetOpen(true)}>
            <Sparkles data-icon="inline-start" />
            {t('providers.fromPreset')}
          </Button>
          <Button
            onClick={() => {
              setEditing(null)
              setFormOpen(true)
            }}
          >
            <Plus data-icon="inline-start" />
            {t('providers.add')}
          </Button>
        </div>
      </header>

      <div className="flex-1 overflow-auto p-6">
        {isPending ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-52 w-full" />
            ))}
          </div>
        ) : isError ? (
          <Empty className="m-auto max-w-md">
            <EmptyHeader>
              <EmptyTitle>{t('providers.loadFailed')}</EmptyTitle>
              <EmptyDescription>{error.message}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : providers.length === 0 ? (
          <Empty className="m-auto max-w-md">
            <EmptyHeader>
              <EmptyTitle>{t('providers.empty.title')}</EmptyTitle>
              <EmptyDescription>{t('providers.empty.desc')}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button onClick={() => setPresetOpen(true)}>
                <Sparkles data-icon="inline-start" />
                {t('providers.empty.cta')}
              </Button>
            </EmptyContent>
          </Empty>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {providers.map((p) => (
              <ProviderCard
                key={p.slug}
                provider={p}
                onEdit={() => {
                  setEditing(p)
                  setFormOpen(true)
                }}
              />
            ))}
          </div>
        )}
      </div>

      <ProviderFormDialog
        open={formOpen}
        onOpenChange={(open) => {
          setFormOpen(open)
          if (!open) setEditing(null)
        }}
        provider={editing}
      />
      <PresetImportDialog open={presetOpen} onOpenChange={setPresetOpen} />
    </div>
  )
}

function ProviderCard({ provider, onEdit }: { provider: Provider; onEdit: () => void }) {
  const deleteMutation = useDeleteProvider()
  const savePresetMutation = useSavePreset()
  const [testOpen, setTestOpen] = useState(false)
  const { t } = useI18n()
  const hasKey = Boolean(provider.api_key)

  return (
    <Card className="flex h-full flex-col">
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 flex-col gap-1">
            <CardTitle className="truncate">{provider.name}</CardTitle>
            <CardDescription className="font-mono">{provider.slug}</CardDescription>
          </div>
          {hasKey ? (
            <Badge>
              <KeyRound data-icon="inline-start" />
              {t('providers.keySet')}
            </Badge>
          ) : (
            <Badge variant="secondary">{t('providers.noKey')}</Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col gap-3">
        <ModelList provider={provider} />
        <div className="mt-auto flex flex-wrap justify-end gap-2 pt-2">
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground"
            disabled={savePresetMutation.isPending}
            onClick={() => savePresetMutation.mutate(provider.slug)}
          >
            <BookmarkPlus data-icon="inline-start" />
            {t('providers.savePreset')}
          </Button>
          <Button variant="secondary" size="sm" onClick={() => setTestOpen(true)}>
            {t('providers.test')}
          </Button>
          <Button variant="outline" size="sm" onClick={onEdit}>
            <Pencil data-icon="inline-start" />
            {t('providers.edit')}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            disabled={deleteMutation.isPending}
            onClick={() => {
              if (window.confirm(t('providers.deleteConfirm', { name: provider.name }))) {
                deleteMutation.mutate(provider.slug)
              }
            }}
          >
            <Trash2 data-icon="inline-start" />
            {t('providers.delete')}
          </Button>
        </div>
        <TestDialog
          slug={provider.slug}
          defaultModel={provider.default_model}
          models={provider.models ?? []}
          open={testOpen}
          onOpenChange={setTestOpen}
        />
      </CardContent>
    </Card>
  )
}
