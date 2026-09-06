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
import { useAgents, useProfiles, useProviders, useSaveProfile } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import type { Profile } from '@/lib/types'
import { providerAgents } from '@/lib/types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  profile: Profile | null
}

interface FormState {
  slug: string
  name: string
  agent: string
  provider: string
  model: string
  default_args: string
  skip_permissions: string
  is_default: boolean
}

function initialState(p: Profile | null): FormState {
  return {
    slug: p?.slug ?? '',
    name: p?.name ?? '',
    agent: p?.agent ?? '',
    provider: p?.provider ?? '',
    model: p?.model ?? '',
    default_args: p?.default_args ?? '',
    skip_permissions: p?.skip_permissions ?? '',
    is_default: p?.is_default ?? false,
  }
}

export function ProfileFormDialog({ open, onOpenChange, profile }: Props) {
  const [form, setForm] = useState<FormState>(initialState(profile))
  const saveMutation = useSaveProfile(profile?.slug ?? null)
  const { t } = useI18n()
  const { data: agents = [] } = useAgents()
  const { data: providers = [] } = useProviders()
  const { data: profiles = [] } = useProfiles()

  useEffect(() => {
    if (open) setForm(initialState(profile))
  }, [open, profile])

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((prev) => ({ ...prev, [key]: value }))

  // Only offer providers that can actually serve the selected agent.
  const usableProviders = form.agent
    ? providers.filter((p) => providerAgents(p).includes(form.agent))
    : providers
  const provider = providers.find((p) => p.slug === form.provider)
  const models = provider?.models ?? []

  const invalid =
    !form.slug.trim() ||
    !form.agent ||
    !form.provider ||
    (profiles.some((p) => p.slug === form.slug && p.slug !== profile?.slug) ? true : false)

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (invalid) return
    saveMutation.mutate(
      {
        slug: form.slug.trim(),
        name: form.name.trim() || form.slug.trim(),
        agent: form.agent,
        provider: form.provider,
        model: form.model.trim() || undefined,
        default_args: form.default_args.trim() || undefined,
        skip_permissions: form.skip_permissions || undefined,
        is_default: form.is_default,
      },
      { onSuccess: () => onOpenChange(false) }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{profile ? t('profileForm.editTitle') : t('profileForm.addTitle')}</DialogTitle>
          <DialogDescription>
            t('profileForm.desc')
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit}>
          <FieldGroup>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field data-invalid={form.slug.trim() ? undefined : true}>
                <FieldLabel htmlFor="pr-slug">{t('providerForm.slug')}</FieldLabel>
                <Input
                  id="pr-slug"
                  value={form.slug}
                  onChange={(e) => set('slug', e.target.value)}
                  aria-invalid={!form.slug.trim()}
                  placeholder="codex-glm53"
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="pr-name">{t('profileForm.name')}</FieldLabel>
                <Input
                  id="pr-name"
                  value={form.name}
                  onChange={(e) => set('name', e.target.value)}
                  placeholder="Codex with GLM 5.3"
                />
              </Field>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field data-invalid={form.agent ? undefined : true}>
                <FieldLabel htmlFor="pr-agent">{t('profiles.agent')}</FieldLabel>
                <Select value={form.agent} onValueChange={(v) => set('agent', v)}>
                  <SelectTrigger id="pr-agent" className="w-full">
                    <SelectValue placeholder={t('profileForm.selectAgent')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {agents.map((a) => (
                        <SelectItem key={a.slug} value={a.slug}>
                          {a.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
              <Field data-invalid={form.provider ? undefined : true}>
                <FieldLabel htmlFor="pr-provider">{t('profiles.provider')}</FieldLabel>
                <Select
                  value={form.provider}
                  onValueChange={(v) => {
                    set('provider', v)
                    set('model', '')
                  }}
                >
                  <SelectTrigger id="pr-provider" className="w-full">
                    <SelectValue placeholder={t('profileForm.selectProvider')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {usableProviders.map((p) => (
                        <SelectItem key={p.slug} value={p.slug}>
                          {p.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                {form.agent && (
                  <FieldDescription>
                    {t('profileForm.providerNote', { agent: form.agent })}
                  </FieldDescription>
                )}
              </Field>
            </div>
            <Field>
              <FieldLabel htmlFor="pr-model">{t('profileForm.modelOverride')}</FieldLabel>
              {models.length > 0 ? (
                <Select value={form.model} onValueChange={(v) => set('model', v)}>
                  <SelectTrigger id="pr-model" className="w-full">
                    <SelectValue placeholder={t('profileForm.modelPlaceholder', { model: provider?.default_model ?? '—' })} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {models.map((m) => (
                        <SelectItem key={m} value={m}>
                          {m}
                          {m === provider?.default_model ? t('test.default') : ''}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  id="pr-model"
                  value={form.model}
                  onChange={(e) => set('model', e.target.value)}
                  placeholder={t('profileForm.modelEmpty')}
                />
              )}
            </Field>
            <Field>
              <FieldLabel htmlFor="pr-args">{t('profileForm.args')}</FieldLabel>
              <Input
                id="pr-args"
                value={form.default_args}
                onChange={(e) => set('default_args', e.target.value)}
                placeholder="--verbose"
              />
            </Field>
            <div className="flex items-center justify-between rounded-lg border p-3">
              <div className="flex flex-col">
                <span className="text-sm font-medium">{t('profileForm.isDefault')}</span>
                <span className="text-muted-foreground text-xs">{t('profileForm.isDefaultDesc')}</span>
              </div>
              <Switch checked={form.is_default} onCheckedChange={(v) => set('is_default', v)} />
            </div>
          </FieldGroup>
          <DialogFooter className="mt-6">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('providerForm.cancel')}
            </Button>
            <Button type="submit" disabled={invalid || saveMutation.isPending}>
              {saveMutation.isPending && <Spinner data-icon="inline-start" />}
              {t('profileForm.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
