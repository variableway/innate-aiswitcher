import { useState } from 'react'
import { BookmarkPlus, KeyRound, Pencil, Plug, Plus, Sparkles, Trash2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { PageHeader } from '@/components/page-header'
import { QueryState } from '@/components/query-state'
import { TestDialog } from '@/components/providers/test-dialog'
import { ModelList } from '@/components/providers/model-list'
import { PresetImportDialog } from '@/components/providers/preset-import'
import { ProviderFormDialog } from '@/components/providers/provider-form'
import { useDeleteProvider, useProviders, useSavePreset } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import { CARD_GRID } from '@/lib/layout'
import type { Provider } from '@/lib/types'

export function ProvidersPage() {
  const query = useProviders()
  const [formOpen, setFormOpen] = useState(false)
  const [presetOpen, setPresetOpen] = useState(false)
  const [editing, setEditing] = useState<Provider | null>(null)
  const { t } = useI18n()

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <PageHeader
        icon={Plug}
        title={t('providers.title')}
        subtitle={t('providers.subtitle')}
        actions={
          <>
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
          </>
        }
      />

      <div className="flex-1 overflow-auto p-6">
        <QueryState
          query={query}
          skeleton={
            <div className={CARD_GRID}>
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-52 w-full" />
              ))}
            </div>
          }
          errorTitle={t('providers.loadFailed')}
          emptyTitle={t('providers.empty.title')}
          emptyDescription={t('providers.empty.desc')}
          emptyContent={
            <Button onClick={() => setPresetOpen(true)}>
              <Sparkles data-icon="inline-start" />
              {t('providers.empty.cta')}
            </Button>
          }
        >
          {(providers) => (
            <div className={CARD_GRID}>
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
        </QueryState>
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
  const [deleteOpen, setDeleteOpen] = useState(false)
  const { t } = useI18n()
  const hasKey = Boolean(provider.api_key)

  return (
    <Card className="flex h-full flex-col">
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 flex-col gap-1">
            <CardTitle className="truncate" title={provider.name}>
              {provider.name}
            </CardTitle>
            <CardDescription className="truncate font-mono" title={provider.slug}>
              {provider.slug}
            </CardDescription>
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
      </CardContent>
      <CardFooter className="mt-auto flex-wrap justify-end gap-2">
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
          onClick={() => setDeleteOpen(true)}
        >
          <Trash2 data-icon="inline-start" />
          {t('providers.delete')}
        </Button>
      </CardFooter>
      <TestDialog
        slug={provider.slug}
        defaultModel={provider.default_model}
        models={provider.models ?? []}
        open={testOpen}
        onOpenChange={setTestOpen}
      />
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t('providers.delete')}
        description={t('providers.deleteConfirm', { name: provider.name })}
        confirmLabel={t('providers.delete')}
        pending={deleteMutation.isPending}
        onConfirm={() =>
          deleteMutation.mutate(provider.slug, { onSuccess: () => setDeleteOpen(false) })
        }
      />
    </Card>
  )
}
