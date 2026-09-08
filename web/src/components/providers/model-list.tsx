import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Check, DatabaseZap, Eye, Plus, X } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Switch } from '@/components/ui/switch'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { useAddModel, useRemoveModel } from '@/lib/queries'
import { api } from '@/lib/api'
import { useI18n } from '@/lib/i18n'
import type { ModelMeta, Provider } from '@/lib/types'

/**
 * The provider's model list. Every model shares the vendor's stored API key.
 * Click a model badge to annotate it (pricing, multimodal, note); removal
 * needs a second confirming click.
 */
export function ModelList({ provider }: { provider: Provider }) {
  const { t } = useI18n()
  const [adding, setAdding] = useState(false)
  const [model, setModel] = useState('')
  const addMutation = useAddModel(provider.slug)
  const removeMutation = useRemoveModel(provider.slug)
  const [confirming, setConfirming] = useState<string | null>(null)
  const [defaultRemove, setDefaultRemove] = useState<string | null>(null)
  const [enriching, setEnriching] = useState(false)
  const qc = useQueryClient()
  const models = provider.models ?? []

  const enrich = async () => {
    setEnriching(true)
    try {
      const res = await api.enrichModels(provider.slug)
      toast.success(t('models.enrichDone', { updated: res.result.updated, matched: res.result.matched }), {
        description: res.result.missed?.length
          ? t('models.enrichMissed', { missed: res.result.missed.join(', ') })
          : undefined,
      })
      await qc.invalidateQueries({ queryKey: ['providers'] })
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setEnriching(false)
    }
  }

  const submit = () => {
    const name = model.trim()
    if (!name) return
    addMutation.mutate(
      { model: name, isDefault: false },
      {
        onSuccess: () => {
          setModel('')
          setAdding(false)
        },
      }
    )
  }

  const remove = (m: string) => {
    // Removing the default model has wider impact — ask with an AlertDialog.
    if (m === provider.default_model) {
      setDefaultRemove(m)
      return
    }
    if (confirming !== m) {
      setConfirming(m)
      return
    }
    removeMutation.mutate(m)
    setConfirming(null)
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-1.5">
        {models.length === 0 && (
          <span className="text-muted-foreground text-xs">{t('providers.noModels')}</span>
        )}
        {models.map((m) => {
          const meta = provider.model_meta?.[m]
          const title = meta
            ? [meta.input_price, meta.output_price, meta.note].filter(Boolean).join(' · ') || m
            : m
          const isConfirming = confirming === m
          return (
            <Badge
              key={m}
              variant={
                isConfirming ? 'destructive' : m === provider.default_model ? 'default' : 'secondary'
              }
              title={title}
            >
              <ModelMetaPopover provider={provider} model={m} meta={meta}>
                <button
                  type="button"
                  className="cursor-pointer"
                  aria-label={t('meta.title') + ' ' + m}
                >
                  {m === provider.default_model && <Check data-icon="inline-start" />}
                  {m}
                </button>
              </ModelMetaPopover>
              {meta?.multimodal && <Eye data-icon="inline-end" />}
              <button
                type="button"
                aria-label={isConfirming ? t('meta.confirmDelete') : t('models.remove', { model: m })}
                title={isConfirming ? t('meta.deleteHint', { model: m }) : t('models.remove', { model: m })}
                className="hover:text-destructive ml-0.5 flex cursor-pointer items-center rounded-sm"
                disabled={removeMutation.isPending}
                onBlur={() => isConfirming && setConfirming(null)}
                onClick={() => remove(m)}
              >
                {isConfirming ? t('meta.confirmDelete') : ''}
                <X className="size-3" />
              </button>
            </Badge>
          )
        })}
      </div>
      <p className="text-muted-foreground text-xs">{t('meta.editHint')}</p>
      <div>
        <Button
          variant="ghost"
          size="sm"
          className="text-muted-foreground"
          disabled={enriching || models.length === 0}
          onClick={() => void enrich()}
        >
          <DatabaseZap data-icon="inline-start" />
          {t('models.enrich')}
        </Button>
      </div>
      {adding ? (
        <div className="flex gap-2">
          <Input
            autoFocus
            placeholder={t('models.placeholder')}
            value={model}
            onChange={(e) => setModel(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') submit()
              if (e.key === 'Escape') {
                setAdding(false)
                setModel('')
              }
            }}
          />
          <Button size="sm" variant="outline" onClick={submit} disabled={addMutation.isPending}>
            {t('models.addBtn')}
          </Button>
        </div>
      ) : (
        <div>
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground"
            onClick={() => setAdding(true)}
          >
            <Plus data-icon="inline-start" />
            {t('models.add')}
          </Button>
        </div>
      )}
      {models.length === 0 && !adding && provider.default_model && (
        <p className="text-muted-foreground text-xs">
          {t('models.default', { model: provider.default_model })}
        </p>
      )}
      <ConfirmDialog
        open={defaultRemove !== null}
        onOpenChange={(open) => !open && setDefaultRemove(null)}
        title={t('models.removeDefault.title')}
        description={
          defaultRemove ? t('models.removeDefault.desc', { model: defaultRemove }) : undefined
        }
        confirmLabel={t('common.delete')}
        pending={removeMutation.isPending}
        onConfirm={() => {
          if (!defaultRemove) return
          removeMutation.mutate(defaultRemove, { onSuccess: () => setDefaultRemove(null) })
        }}
      />
    </div>
  )
}

/** Click-to-edit annotations for one model (pricing, modality, note). */
function ModelMetaPopover({
  provider,
  model,
  meta,
  children,
}: {
  provider: Provider
  model: string
  meta?: ModelMeta
  children: React.ReactNode
}) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState<ModelMeta>(meta ?? {})
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setSaving(true)
    try {
      const nextMeta = { ...(provider.model_meta ?? {}), [model]: form }
      // clean empty meta entries
      if (!form.input_price && !form.output_price && !form.multimodal && !form.note) {
        delete nextMeta[model]
      }
      await fetch(`/api/aisw/providers/${provider.slug}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          slug: provider.slug,
          name: provider.name,
          base_url: provider.base_url,
          api_key: '',
          api_protocol: provider.api_protocol,
          default_model: provider.default_model,
          models: provider.models ?? [],
          model_meta: nextMeta,
          variants: provider.variants,
          active: provider.active,
        }),
      })
      toast.success(t('meta.saved'))
      setOpen(false)
      await qc.invalidateQueries({ queryKey: ['providers'] })
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) setForm(meta ?? {})
      }}
    >
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className="w-72" align="start">
        <FieldGroup>
          <p className="font-mono text-xs font-semibold">{model}</p>
          <Field>
            <FieldLabel htmlFor="mm-in">{t('meta.inputPrice')}</FieldLabel>
            <Input
              id="mm-in"
              value={form.input_price ?? ''}
              onChange={(e) => setForm({ ...form, input_price: e.target.value })}
              placeholder="2.5 / 1M"
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="mm-out">{t('meta.outputPrice')}</FieldLabel>
            <Input
              id="mm-out"
              value={form.output_price ?? ''}
              onChange={(e) => setForm({ ...form, output_price: e.target.value })}
              placeholder="10 / 1M"
            />
            <FieldDescription>{t('meta.priceHint')}</FieldDescription>
          </Field>
          <div className="flex items-center justify-between rounded-lg border p-2.5">
            <span className="text-sm">{t('meta.multimodal')}</span>
            <Switch
              checked={Boolean(form.multimodal)}
              onCheckedChange={(v) => setForm({ ...form, multimodal: v })}
            />
          </div>
          <Field>
            <FieldLabel htmlFor="mm-note">{t('meta.note')}</FieldLabel>
            <Input
              id="mm-note"
              value={form.note ?? ''}
              onChange={(e) => setForm({ ...form, note: e.target.value })}
            />
          </Field>
          <Button size="sm" disabled={saving} onClick={() => void save()}>
            {t('meta.save')}
          </Button>
        </FieldGroup>
      </PopoverContent>
    </Popover>
  )
}
