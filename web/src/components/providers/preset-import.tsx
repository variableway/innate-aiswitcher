import { useRef, useState } from 'react'
import { Check, FileUp, Trash2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { useDeletePreset, useImportPreset, useImportPresetFile, usePresets } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import type { Preset } from '@/lib/types'
import { PROTOCOL_AGENTS } from '@/lib/types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function PresetImportDialog({ open, onOpenChange }: Props) {
  const { data: presets = [], isPending } = usePresets()
  const [selected, setSelected] = useState<Preset | null>(null)
  const [apiKey, setApiKey] = useState('')
  const importMutation = useImportPreset()
  const importFileMutation = useImportPresetFile()
  const deleteMutation = useDeletePreset()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const { t } = useI18n()

  const close = (next: boolean) => {
    if (!next) {
      setSelected(null)
      setApiKey('')
    }
    onOpenChange(next)
  }

  const importPreset = () => {
    if (!selected) return
    importMutation.mutate(
      { slug: selected.slug, apiKey },
      {
        onSuccess: () => close(false),
      }
    )
  }

  const onFileSelected = async (file: File) => {
    const content = await file.text()
    importFileMutation.mutate(content)
  }

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('preset.title')}</DialogTitle>
          <DialogDescription>{t('preset.desc')}</DialogDescription>
        </DialogHeader>

        {isPending ? (
          <div className="text-muted-foreground flex items-center gap-2 py-6 text-sm">
            <Spinner /> {t('preset.loading')}
          </div>
        ) : (
          <FieldGroup>
            <div className="flex flex-col gap-2">
              {presets.map((preset) => {
                const active = selected?.slug === preset.slug
                const protocols = (preset.variants ?? []).map((v) => v.protocol)
                const agents = [
                  ...new Set(protocols.flatMap((p) => PROTOCOL_AGENTS[p] ?? [])),
                ]
                return (
                  <div
                    key={preset.slug}
                    className={`flex items-start gap-2 rounded-lg border p-3 transition-colors ${
                      active ? 'border-primary bg-accent/50' : 'hover:bg-accent/40'
                    }`}
                  >
                    <button
                      type="button"
                      className="flex min-w-0 flex-1 cursor-pointer flex-col gap-2 text-left"
                      onClick={() => setSelected(preset)}
                    >
                      <span className="flex flex-wrap items-center gap-2 text-sm font-medium">
                        {active && <Check data-icon="inline-start" />}
                        {preset.name}
                        <Badge variant="secondary" className="font-mono">
                          {preset.slug}
                        </Badge>
                        <Badge variant="outline">
                          {preset.source === 'user' ? t('preset.user') : t('preset.builtin')}
                        </Badge>
                      </span>
                      <span className="text-muted-foreground flex flex-wrap items-center gap-1.5 text-xs">
                        <span>
                          {t('providers.agents')}: {agents.join(', ') || '—'}
                        </span>
                        {(preset.models ?? []).map((m) => (
                          <Badge key={m} variant="outline">
                            {m}
                          </Badge>
                        ))}
                      </span>
                    </button>
                    {preset.source === 'user' && (
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={t('preset.delete')}
                        className="text-muted-foreground hover:text-destructive"
                        disabled={deleteMutation.isPending}
                        onClick={() => deleteMutation.mutate(preset.slug)}
                      >
                        <Trash2 />
                      </Button>
                    )}
                  </div>
                )
              })}
            </div>
            {selected && (
              <div className="text-muted-foreground flex flex-col gap-1 rounded-lg border p-3 font-mono text-xs">
                {(selected.variants ?? []).map((v) => (
                  <span key={v.protocol}>
                    {v.protocol.padEnd(17)} {v.base_url}
                  </span>
                ))}
              </div>
            )}
            <Field>
              <FieldLabel htmlFor="preset-key">{t('preset.apiKey')}</FieldLabel>
              <Input
                id="preset-key"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder={t('preset.apiKeyPlaceholder')}
              />
              <FieldDescription>{t('preset.apiKeyDesc')}</FieldDescription>
            </Field>
          </FieldGroup>
        )}

        <DialogFooter className="items-center sm:justify-between">
          <input
            ref={fileInputRef}
            type="file"
            accept=".toml"
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) void onFileSelected(file)
              e.target.value = ''
            }}
          />
          <Button
            variant="outline"
            onClick={() => fileInputRef.current?.click()}
            disabled={importFileMutation.isPending}
          >
            {importFileMutation.isPending ? (
              <Spinner data-icon="inline-start" />
            ) : (
              <FileUp data-icon="inline-start" />
            )}
            {t('preset.importFile')}
          </Button>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => close(false)}>
              {t('preset.cancel')}
            </Button>
            <Button
              disabled={!selected || !apiKey || importMutation.isPending}
              onClick={importPreset}
            >
              {importMutation.isPending && <Spinner data-icon="inline-start" />}
              {t('preset.import')}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
