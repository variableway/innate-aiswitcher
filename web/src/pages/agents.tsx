import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { toast } from 'sonner'
import { Bot, Check, CircleX, Copy, HardDriveDownload, Wand2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
import { ConfirmDialog } from '@/components/confirm-dialog'
import { PageHeader } from '@/components/page-header'
import { QueryState } from '@/components/query-state'
import { ModelCombobox } from '@/components/configs/model-combobox'
import { api } from '@/lib/api'
import { useI18n } from '@/lib/i18n'
import { CARD_GRID } from '@/lib/layout'
import {
  AGENT_TEMPLATES,
  providerAgents,
  type AgentTemplate,
  type InstalledAgent,
  type Provider,
} from '@/lib/types'

const INSTALL_HINTS: Record<string, string> = {
  claude: 'npm install -g @anthropic-ai/claude-code',
  codex: 'npm install -g @openai/codex',
  opencode: 'curl -fsSL https://opencode.ai/install | bash',
}

export function AgentsPage() {
  const query = useQuery({ queryKey: ['installed-agents'], queryFn: api.installedAgents })
  const providersQuery = useQuery({ queryKey: ['providers'], queryFn: api.listProviders })
  // Known on-disk paths so the write-to-disk confirmation can name its target.
  const configsQuery = useQuery({ queryKey: ['agent-configs'], queryFn: api.listAgentConfigs })
  const providers = (providersQuery.data ?? []).filter((p) => p.active)
  const { t } = useI18n()

  const configPaths = useMemo(() => {
    const map: Record<string, string> = {}
    for (const info of configsQuery.data ?? []) {
      for (const file of info.files) map[`${file.agent}/${file.name}`] = file.path
    }
    return map
  }, [configsQuery.data])

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <PageHeader icon={Bot} title={t('agents.title')} subtitle={t('agents.subtitle')} />

      <div className="flex-1 overflow-auto p-6">
        <QueryState
          query={query}
          skeleton={
            <div className={CARD_GRID}>
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-64 w-full" />
              ))}
            </div>
          }
          errorTitle={t('agents.loadFailed')}
        >
          {(data) => (
            <div className="flex flex-col gap-3">
              <p className="text-muted-foreground text-xs">{t('agents.templateHint')}</p>
              <div className={CARD_GRID}>
                {data.map((agent) => (
                  <AgentCard
                    key={agent.slug}
                    agent={agent}
                    providers={providers}
                    configPaths={configPaths}
                    onModelRegistered={() => void providersQuery.refetch()}
                  />
                ))}
              </div>
            </div>
          )}
        </QueryState>
      </div>
    </div>
  )
}

function AgentCard({
  agent,
  providers,
  configPaths,
  onModelRegistered,
}: {
  agent: InstalledAgent
  providers: Provider[]
  configPaths: Record<string, string>
  onModelRegistered: () => void
}) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const templates = AGENT_TEMPLATES[agent.slug] ?? []

  const usable = providers.filter((p) => providerAgents(p).includes(agent.slug))
  const usableSlugs = useMemo(() => usable.map((p) => p.slug).join(','), [usable])
  const [provider, setProvider] = useState<string>(usable[0]?.slug ?? '')
  const [model, setModel] = useState<string>('')

  // Backfill once providers arrive asynchronously; keep the selection coherent.
  useEffect(() => {
    if (provider && usableSlugs.split(',').includes(provider)) return
    setProvider(usableSlugs.split(',').filter(Boolean)[0] ?? '')
  }, [usableSlugs]) // eslint-disable-line react-hooks/exhaustive-deps
  const providerInfo = usable.find((p) => p.slug === provider)
  useEffect(() => {
    if (providerInfo && model && (providerInfo.models ?? []).includes(model)) return
    setModel(providerInfo?.default_model ?? providerInfo?.models?.[0] ?? '')
  }, [providerInfo?.slug]) // eslint-disable-line react-hooks/exhaustive-deps

  const goBuilder = () => {
    void navigate({
      to: '/configs',
      search: { agent: agent.slug, provider, model },
    })
  }
  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 flex-col gap-1">
            <CardTitle className="truncate" title={agent.name}>
              {agent.name}
            </CardTitle>
            <CardDescription
              className="truncate font-mono"
              title={`${agent.binary}${agent.version ? ` · ${agent.version}` : ''}`}
            >
              {agent.binary}
              {agent.version ? ` · ${agent.version}` : ''}
            </CardDescription>
          </div>
          {agent.installed ? (
            <Badge>
              <Check data-icon="inline-start" />
              {t('agents.installed')}
            </Badge>
          ) : (
            <Badge variant="destructive">
              <CircleX data-icon="inline-start" />
              {t('agents.notInstalled')}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {agent.installed ? (
          <p className="text-muted-foreground truncate font-mono text-xs" title={agent.path}>
            {agent.path}
          </p>
        ) : (
          <p className="text-muted-foreground font-mono text-xs">
            {t('agents.installHint')} {INSTALL_HINTS[agent.slug] ?? t('agents.installHintUnknown')}
          </p>
        )}
        <Separator />
        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium">{t('agents.builder')}</span>
          <p className="text-muted-foreground text-xs">{t('agents.builderHint')}</p>
          <div className="flex flex-wrap items-center gap-2">
            <Select
              value={provider}
              onValueChange={(v) => {
                setProvider(v)
                setModel('')
              }}
            >
              <SelectTrigger className="w-48">
                <SelectValue placeholder={t('configs.pickProvider')} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {usable.map((p) => (
                    <SelectItem key={p.slug} value={p.slug}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <ModelCombobox
              provider={provider}
              configured={providerInfo?.models ?? []}
              value={model}
              onChange={setModel}
              onModelRegistered={onModelRegistered}
            />
            <Button size="sm" disabled={!provider || !model || usable.length === 0} onClick={goBuilder}>
              <Wand2 data-icon="inline-start" />
              {t('agents.builder')}
            </Button>
          </div>
          {usable.length === 0 && (
            <p className="text-muted-foreground text-xs">{t('configs.noUsableProviders')}</p>
          )}
        </div>
        <Separator />
        <div className="flex flex-col gap-3">
          <span className="text-muted-foreground text-xs font-medium uppercase">
            {t('agents.templates')}
          </span>
          {templates.map((tpl) => (
            <TemplateBlock
              key={tpl.label}
              agent={agent.slug}
              template={tpl}
              targetPath={configPaths[`${agent.slug}/${tpl.name}`]}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function TemplateBlock({
  agent,
  template,
  targetPath,
}: {
  agent: string
  template: AgentTemplate
  targetPath?: string
}) {
  const { t } = useI18n()
  const [writing, setWriting] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)

  const copy = async () => {
    await navigator.clipboard.writeText(template.content)
    toast.success(t('agents.copied'))
  }

  const writeDisk = async () => {
    setWriting(true)
    try {
      await api.saveAgentConfig(agent, template.name, template.content)
      toast.success(t('agents.written', { file: template.label }))
      setConfirmOpen(false)
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setWriting(false)
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium">{template.label}</span>
        <div className="flex gap-1">
          <Button variant="ghost" size="sm" className="text-muted-foreground" onClick={() => void copy()}>
            <Copy data-icon="inline-start" />
            {t('agents.copy')}
          </Button>
          <Button variant="outline" size="sm" disabled={writing} onClick={() => setConfirmOpen(true)}>
            <HardDriveDownload data-icon="inline-start" />
            {t('agents.writeDisk')}
          </Button>
        </div>
      </div>
      <pre className="bg-muted/40 max-h-64 overflow-auto rounded-lg border p-3 font-mono text-xs">
        {template.content}
      </pre>
      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t('agents.writeConfirm.title')}
        description={t('agents.writeConfirm.desc')}
        confirmLabel={t('agents.writeConfirm.confirm')}
        pending={writing}
        onConfirm={() => void writeDisk()}
      >
        <p className="bg-muted/40 rounded-lg border p-2 font-mono text-xs break-all">
          {targetPath ?? template.label}
        </p>
      </ConfirmDialog>
    </div>
  )
}
