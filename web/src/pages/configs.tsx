import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { FileCode2, HardDriveDownload, RotateCw, Save } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { ModelCombobox } from '@/components/configs/model-combobox'
import { api } from '@/lib/api'
import { useI18n } from '@/lib/i18n'
import type { AgentConfigFile, AgentConfigInfo, ConfigPreview } from '@/lib/types'
import { providerAgents } from '@/lib/types'

export function ConfigsPage() {
  const { t } = useI18n()
  const configsQuery = useQuery({ queryKey: ['agent-configs'], queryFn: api.listAgentConfigs })
  const agentsQuery = useQuery({ queryKey: ['agents'], queryFn: api.listAgents })
  const providersQuery = useQuery({ queryKey: ['providers'], queryFn: api.listProviders })

  const agents = (agentsQuery.data ?? []).filter((a) => a.active)
  const providers = providersQuery.data ?? []

  const [agent, setAgent] = useState<string>(agents[0]?.slug ?? 'claude')
  const usableProviders = providers.filter((p) => p.active && providerAgents(p).includes(agent))
  const [provider, setProvider] = useState<string>('')
  const [model, setModel] = useState<string>('')

  // keep provider/model coherent when agent or provider changes
  useEffect(() => {
    if (provider && usableProviders.some((p) => p.slug === provider)) return
    setProvider(usableProviders[0]?.slug ?? '')
    setModel('')
  }, [agent, JSON.stringify(usableProviders.map((p) => p.slug))])
  const providerInfo = usableProviders.find((p) => p.slug === provider)
  useEffect(() => {
    if (providerInfo && model && (providerInfo.models ?? []).includes(model)) return
    setModel(providerInfo?.default_model ?? providerInfo?.models?.[0] ?? '')
  }, [provider, providerInfo?.slug])

  const previewQuery = useQuery({
    queryKey: ['config-preview', agent, provider, model],
    queryFn: () => api.configPreview(agent, provider, model),
    enabled: Boolean(agent && provider && model),
    retry: false,
  })

  const queryClient = useQueryClient()
  const applyMutation = useMutation({
    mutationFn: async () => {
      const preview = previewQuery.data
      if (!preview) return
      for (const file of preview.files) {
        if (!file.writable || !file.name) continue
        await api.saveAgentConfig(preview.agent, file.name, file.content)
      }
    },
    onSuccess: async () => {
      toast.success(t('configs.applied'))
      await queryClient.invalidateQueries({ queryKey: ['agent-configs'] })
    },
    onError: (err: Error) => toast.error(t('configs.applyFailed', { msg: err.message })),
  })

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header className="border-b flex items-center justify-between px-6 py-4">
        <div className="flex flex-col gap-0.5">
          <h1 className="flex items-center gap-2 text-lg font-semibold">
            <FileCode2 className="size-5" />
            {t('configs.title')}
          </h1>
          <p className="text-muted-foreground text-sm">{t('configs.subtitle')}</p>
        </div>
      </header>

      <div className="flex-1 overflow-auto p-6">
        {/* ── Config builder: agent + provider + model → generated files ── */}
        <section className="flex flex-col gap-4">
          <div className="flex flex-wrap items-end gap-4">
            <div className="flex flex-col gap-1.5">
              <span className="text-sm font-medium">{t('configs.agent')}</span>
              <ToggleGroup
                type="single"
                value={agent}
                onValueChange={(v) => v && setAgent(v)}
              >
                {agents.map((a) => (
                  <ToggleGroupItem key={a.slug} value={a.slug}>
                    {a.name}
                  </ToggleGroupItem>
                ))}
              </ToggleGroup>
            </div>
            <div className="flex w-56 flex-col gap-1.5">
              <span className="text-sm font-medium">{t('nav.providers')}</span>
              <Select value={provider} onValueChange={setProvider}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={t('configs.pickProvider')} />
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
            </div>
            <div className="flex flex-col gap-1.5">
              <span className="text-sm font-medium">{t('providerForm.models')}</span>
              <ModelCombobox
                provider={provider}
                configured={providerInfo?.models ?? []}
                value={model}
                onChange={setModel}
                onModelRegistered={() => void providersQuery.refetch()}
              />
            </div>
            <Button
              className="ml-auto"
              disabled={!previewQuery.data || applyMutation.isPending}
              onClick={() => applyMutation.mutate()}
            >
              <HardDriveDownload data-icon="inline-start" />
              {t('configs.apply')}
            </Button>
          </div>

          {usableProviders.length === 0 ? (
            <Empty className="max-w-md">
              <EmptyHeader>
                <EmptyTitle>{t('configs.noUsableProviders')}</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : previewQuery.isPending ? (
            <Skeleton className="h-64 w-full" />
          ) : previewQuery.isError ? (
            <Empty className="max-w-md">
              <EmptyHeader>
                <EmptyTitle>{t('configs.previewError', { msg: (previewQuery.error as Error).message })}</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : previewQuery.data ? (
            <PreviewResultView preview={previewQuery.data} />
          ) : null}
        </section>

        <Separator className="my-6" />

        {/* ── Current on-disk configs (editable) ── */}
        <section className="flex flex-col gap-3">
          <h2 className="text-base font-semibold">{t('configs.diskTitle')}</h2>
          <p className="text-muted-foreground text-xs">{t('configs.hint')}</p>
          {configsQuery.isPending ? (
            <div className="grid gap-4 xl:grid-cols-2">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-72 w-full" />
              ))}
            </div>
          ) : configsQuery.isError ? (
            <Empty className="max-w-md">
              <EmptyHeader>
                <EmptyTitle>{t('configs.loadFailed')}</EmptyTitle>
                <EmptyDescription>{(configsQuery.error as Error).message}</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="grid gap-4 xl:grid-cols-2">
              {(configsQuery.data ?? []).map((info) => (
                <AgentConfigCard key={info.agent} agent={info} />
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}

function PreviewResultView({ preview }: { preview: ConfigPreview }) {
  const { t } = useI18n()
  return (
    <div className="flex flex-col gap-3">
      {preview.files.map((file) => (
        <div key={file.label} className="flex flex-col gap-1.5">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">{file.label}</span>
            <Badge variant="outline" className="font-mono">
              {file.lang || 'info'}
            </Badge>
            {file.writable ? <Badge variant="secondary">{t('preset.user')}</Badge> : null}
          </div>
          {file.content ? (
            <pre className="bg-muted/40 max-h-96 overflow-auto rounded-lg border p-3 font-mono text-xs">
              {file.content}
            </pre>
          ) : (
            <p className="text-muted-foreground font-mono text-xs">{t('configs.notExists')}</p>
          )}
        </div>
      ))}
      {preview.env && Object.keys(preview.env).length > 0 && (
        <div className="flex flex-col gap-1.5">
          <span className="text-sm font-medium">{t('configs.envOnly')}</span>
          <pre className="bg-muted/40 overflow-auto rounded-lg border p-3 font-mono text-xs">
            {Object.entries(preview.env)
              .map(([k, v]) => `${k}=${v}`)
              .join('\n')}
          </pre>
        </div>
      )}
    </div>
  )
}

function AgentConfigCard({ agent }: { agent: AgentConfigInfo }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{agent.name}</CardTitle>
        <CardDescription className="font-mono">{agent.agent}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {agent.files.map((file) => (
          <ConfigFileEditor key={`${file.agent}/${file.name}`} file={file} />
        ))}
      </CardContent>
    </Card>
  )
}

function ConfigFileEditor({ file }: { file: AgentConfigFile }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [content, setContent] = useState(file.content)
  const [saving, setSaving] = useState(false)
  const dirty = content !== file.content

  useEffect(() => {
    setContent(file.content)
  }, [file.content])

  const save = async () => {
    setSaving(true)
    try {
      await api.saveAgentConfig(file.agent, file.name, content)
      toast.success(t('configs.saved', { file: file.label }))
      await qc.invalidateQueries({ queryKey: ['agent-configs'] })
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const reload = async () => {
    await qc.invalidateQueries({ queryKey: ['agent-configs'] })
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <span className="text-sm font-medium">{file.label}</span>
          <Badge variant="outline" className="font-mono">
            {file.lang}
          </Badge>
          {!file.exists && (
            <span className="text-muted-foreground text-xs">{t('configs.notExists')}</span>
          )}
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" size="sm" className="text-muted-foreground" onClick={reload}>
            <RotateCw data-icon="inline-start" />
            {t('configs.reload')}
          </Button>
          <Button size="sm" onClick={save} disabled={!dirty || saving}>
            {saving ? <Spinner data-icon="inline-start" /> : <Save data-icon="inline-start" />}
            {t('configs.save')}
          </Button>
        </div>
      </div>
      <p className="text-muted-foreground truncate font-mono text-xs" title={file.path}>
        {file.path}
      </p>
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        spellCheck={false}
        className="bg-muted/40 focus-visible:border-ring focus-visible:ring-ring/50 h-56 w-full resize-y rounded-lg border p-3 font-mono text-xs outline-none focus-visible:ring-2"
      />
    </div>
  )
}
