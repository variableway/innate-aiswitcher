import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { useSaveProvider } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import type { Provider } from '@/lib/types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  provider: Provider | null
}

interface FormState {
  slug: string
  name: string
  api_key: string
  api_protocol: string
  base_url: string
  default_model: string
  models: string
  notes: string
  active: boolean
}

function initialState(p: Provider | null): FormState {
  return {
    slug: p?.slug ?? '',
    name: p?.name ?? '',
    api_key: '',
    api_protocol: p?.api_protocol ?? 'openai_chat',
    base_url: p?.base_url ?? '',
    default_model: p?.default_model ?? '',
    models: (p?.models ?? []).join(', '),
    notes: p?.notes ?? '',
    active: p?.active ?? true,
  }
}

export function ProviderFormDialog({ open, onOpenChange, provider }: Props) {
  const [form, setForm] = useState<FormState>(initialState(provider))
  // Validation shows only after a field was touched or a submit was attempted.
  const [touched, setTouched] = useState<Partial<Record<keyof FormState, boolean>>>({})
  const [submitAttempted, setSubmitAttempted] = useState(false)
  const saveMutation = useSaveProvider(provider?.slug ?? null)
  const { t } = useI18n()

  useEffect(() => {
    if (open) {
      setForm(initialState(provider))
      setTouched({})
      setSubmitAttempted(false)
    }
  }, [open, provider])

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((prev) => ({ ...prev, [key]: value }))
  const touch = (key: keyof FormState) => setTouched((prev) => ({ ...prev, [key]: true }))
  const showInvalid = (key: keyof FormState, invalid: boolean) =>
    (touched[key] || submitAttempted) && invalid

  const models = form.models
    .split(',')
    .map((m) => m.trim())
    .filter(Boolean)

  const isVendor = Boolean(provider?.variants && Object.keys(provider.variants).length > 0)
  const slugInvalid = !form.slug.trim()
  const modelsInvalid = models.length === 0
  const baseUrlInvalid = !form.base_url.trim()
  const invalid = slugInvalid || baseUrlInvalid || modelsInvalid

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (invalid) {
      setSubmitAttempted(true)
      return
    }
    saveMutation.mutate(
      {
        slug: form.slug.trim(),
        name: form.name.trim() || form.slug.trim(),
        base_url: form.base_url.trim(),
        api_key: form.api_key,
        api_protocol: form.api_protocol,
        default_model: form.default_model.trim() || models[0],
        models,
        notes: form.notes.trim(),
        active: form.active,
        // vendor rows keep their per-protocol variants on edit
        ...(provider?.variants ? { variants: provider.variants } : {}),
      },
      { onSuccess: () => onOpenChange(false) }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {provider ? t('providerForm.editTitle') : t('providerForm.addTitle')}
          </DialogTitle>
          <DialogDescription>
            {isVendor ? t('providerForm.vendorNote') : t('providerForm.singleNote')}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit}>
          <FieldGroup>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field data-invalid={showInvalid('slug', slugInvalid) || undefined}>
                <FieldLabel htmlFor="p-slug">{t('providerForm.slug')}</FieldLabel>
                <Input
                  id="p-slug"
                  value={form.slug}
                  onChange={(e) => set('slug', e.target.value)}
                  onBlur={() => touch('slug')}
                  aria-invalid={showInvalid('slug', slugInvalid)}
                  placeholder="glm"
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="p-name">{t('providerForm.name')}</FieldLabel>
                <Input
                  id="p-name"
                  value={form.name}
                  onChange={(e) => set('name', e.target.value)}
                  placeholder="GLM (Volcengine Ark)"
                />
              </Field>
            </div>
            <Field>
              <FieldLabel htmlFor="p-key">{t('providerForm.apiKey')}</FieldLabel>
              <Input
                id="p-key"
                type="password"
                value={form.api_key}
                onChange={(e) => set('api_key', e.target.value)}
                placeholder={
                  provider?.api_key
                    ? t('providerForm.apiKeySaved')
                    : t('providerForm.apiKeyPlaceholder')
                }
              />
              <FieldDescription>{t('providerForm.apiKeyDesc')}</FieldDescription>
            </Field>
            <Field data-invalid={showInvalid('models', modelsInvalid) || undefined}>
              <FieldLabel htmlFor="p-models">{t('providerForm.models')}</FieldLabel>
              <Input
                id="p-models"
                value={form.models}
                onChange={(e) => set('models', e.target.value)}
                onBlur={() => touch('models')}
                aria-invalid={showInvalid('models', modelsInvalid)}
                placeholder="glm-5.2, glm-5.3"
              />
              <FieldDescription>{t('providerForm.modelsDesc')}</FieldDescription>
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="p-default-model">{t('providerForm.defaultModel')}</FieldLabel>
                <Input
                  id="p-default-model"
                  value={form.default_model}
                  onChange={(e) => set('default_model', e.target.value)}
                  placeholder={t('providerForm.defaultModelHint')}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="p-protocol">{t('providerForm.protocol')}</FieldLabel>
                <Select value={form.api_protocol} onValueChange={(v) => set('api_protocol', v)}>
                  <SelectTrigger id="p-protocol" className="w-full">
                    <SelectValue placeholder="protocol" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="openai_chat">OpenAI Chat</SelectItem>
                      <SelectItem value="anthropic">Anthropic</SelectItem>
                      <SelectItem value="openai_responses">OpenAI Responses</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
            </div>
            <Field data-invalid={showInvalid('base_url', baseUrlInvalid) || undefined}>
              <FieldLabel htmlFor="p-base-url">{t('providerForm.baseUrl')}</FieldLabel>
              <Input
                id="p-base-url"
                type="url"
                value={form.base_url}
                onChange={(e) => set('base_url', e.target.value)}
                onBlur={() => touch('base_url')}
                aria-invalid={showInvalid('base_url', baseUrlInvalid)}
                placeholder="https://api.example.com/v1"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="p-notes">{t('providerForm.notes')}</FieldLabel>
              <Textarea
                id="p-notes"
                value={form.notes}
                onChange={(e) => set('notes', e.target.value)}
                rows={2}
              />
            </Field>
            <Field orientation="horizontal" className="rounded-lg border p-3">
              <FieldContent>
                <FieldLabel htmlFor="p-active" className="flex-col items-start gap-0.5">
                  <span className="text-sm font-medium">{t('providerForm.active')}</span>
                  <span className="text-muted-foreground text-xs font-normal">
                    {t('providerForm.activeDesc')}
                  </span>
                </FieldLabel>
              </FieldContent>
              <Switch
                id="p-active"
                checked={form.active}
                onCheckedChange={(v) => set('active', v)}
              />
            </Field>
          </FieldGroup>
          <DialogFooter className="mt-6">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('providerForm.cancel')}
            </Button>
            <Button type="submit" disabled={saveMutation.isPending}>
              {saveMutation.isPending && <Spinner data-icon="inline-start" />}
              {t('providerForm.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
