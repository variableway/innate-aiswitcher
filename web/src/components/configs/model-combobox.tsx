import { useState } from 'react'

import { toast } from 'sonner'
import { Check, ChevronsUpDown, CloudDownload, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Spinner } from '@/components/ui/spinner'
import { api } from '@/lib/api'
import { useI18n } from '@/lib/i18n'

interface Props {
  /** Provider slug used to fetch the remote model list. */
  provider: string
  /** Models configured on the provider (always selectable). */
  configured: string[]
  value: string
  onChange: (model: string) => void
  /** Called after a custom/remote model was registered on the provider. */
  onModelRegistered?: () => void
}

/**
 * Editable model combobox: pick a configured model, fetch the vendor's real
 * model list from its API, or type any name — custom names are registered on
 * the provider (sharing its API key) before being used.
 */
export function ModelCombobox({ provider, configured, value, onChange, onModelRegistered }: Props) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [remote, setRemote] = useState<string[] | null>(null)
  const [fetching, setFetching] = useState(false)
  const [registering, setRegistering] = useState<string | null>(null)

  const remoteOnly = (remote ?? []).filter((m) => !configured.includes(m))
  const custom = query.trim() && !configured.includes(query.trim()) && !remoteOnly.includes(query.trim())
    ? query.trim()
    : null

  const fetchRemote = async () => {
    setFetching(true)
    try {
      const result = await api.remoteModels(provider)
      if (!result.ok || !result.models?.length) {
        toast.error(t('models.fetchFailed'), { description: result.message?.slice(0, 200) })
      }
      setRemote(result.models ?? [])
    } catch (err) {
      toast.error(t('models.fetchFailed'), { description: (err as Error).message })
    } finally {
      setFetching(false)
    }
  }

  const registerAndUse = async (model: string) => {
    if (!model) return
    setRegistering(model)
    try {
      await api.addModel(provider, model, false)
      toast.success(t('models.added', { model }), { description: t('models.addedDesc', { slug: provider }) })
      onChange(model)
      setOpen(false)
      onModelRegistered?.()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setRegistering(null)
    }
  }

  return (
    <div className="flex w-64 gap-2">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" role="combobox" aria-expanded={open} className="w-full justify-between font-normal">
            <span className="truncate font-mono text-xs">{value || t('configs.pickModel')}</span>
            <ChevronsUpDown className="text-muted-foreground" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-64 p-0" align="start">
          <Command shouldFilter={false}>
            <CommandInput
              value={query}
              onValueChange={setQuery}
              placeholder={t('models.typeOrPick')}
            />
            <CommandList>
              <CommandEmpty>{t('models.typeHint')}</CommandEmpty>
              {configured.length > 0 && (
                <CommandGroup heading={t('models.configuredGroup')}>
                  {configured.map((m) => (
                    <CommandItem
                      key={m}
                      value={m}
                      onSelect={() => {
                        onChange(m)
                        setOpen(false)
                      }}
                    >
                      <Check data-icon="inline-start" className={m === value ? '' : 'invisible'} />
                      <span className="font-mono text-xs">{m}</span>
                    </CommandItem>
                  ))}
                </CommandGroup>
              )}
              {remoteOnly.length > 0 && (
                <CommandGroup heading={t('models.remoteGroup')}>
                  {remoteOnly.map((m) => (
                    <CommandItem key={m} value={m} onSelect={() => void registerAndUse(m)}>
                      <CloudDownload data-icon="inline-start" />
                      <span className="font-mono text-xs">{m}</span>
                    </CommandItem>
                  ))}
                </CommandGroup>
              )}
              {custom && (
                <CommandGroup>
                  <CommandItem value={`custom:${custom}`} onSelect={() => void registerAndUse(custom)}>
                    {registering === custom ? <Spinner data-icon="inline-start" /> : <Plus data-icon="inline-start" />}
                    <span className="font-mono text-xs">{custom}</span>
                    <span className="text-muted-foreground ml-auto text-xs">
                      {t('models.useAndAdd')}
                    </span>
                  </CommandItem>
                </CommandGroup>
              )}
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
      <Button
        variant="outline"
        size="icon"
        aria-label={t('models.fetch')}
        title={t('models.fetch')}
        disabled={!provider || fetching}
        onClick={() => void fetchRemote()}
      >
        {fetching ? <Spinner /> : <CloudDownload />}
      </Button>
    </div>
  )
}
