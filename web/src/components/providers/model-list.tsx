import { useState } from 'react'
import { Check, Plus, X } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAddModel, useRemoveModel } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import type { Provider } from '@/lib/types'

/**
 * The provider's configured model list. Every model shares the vendor's
 * stored API key, so adding one is just a name — no key handling.
 */
export function ModelList({ provider }: { provider: Provider }) {
  const [adding, setAdding] = useState(false)
  const [model, setModel] = useState('')
  const addMutation = useAddModel(provider.slug)
  const removeMutation = useRemoveModel(provider.slug)
  const { t } = useI18n()
  const models = provider.models ?? []

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

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-1.5">
        {models.length === 0 && (
          <span className="text-muted-foreground text-xs">{t('providers.noModels')}</span>
        )}
        {models.map((m) => (
          <Badge key={m} variant={m === provider.default_model ? 'default' : 'secondary'}>
            {m === provider.default_model && <Check data-icon="inline-start" />}
            {m}
            <button
              type="button"
              aria-label={t('models.remove', { model: m })}
              className="hover:text-destructive -mr-1 ml-1 cursor-pointer rounded-sm"
              disabled={removeMutation.isPending}
              onClick={() => removeMutation.mutate(m)}
            >
              <X data-icon="inline-end" />
            </button>
          </Badge>
        ))}
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
    </div>
  )
}
