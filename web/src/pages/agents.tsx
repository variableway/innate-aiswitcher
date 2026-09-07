import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Bot, Check, CircleX, Copy, HardDriveDownload } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { api } from '@/lib/api'
import { useI18n } from '@/lib/i18n'
import { AGENT_TEMPLATES, type AgentTemplate, type InstalledAgent } from '@/lib/types'

const INSTALL_HINTS: Record<string, string> = {
  claude: 'npm install -g @anthropic-ai/claude-code',
  codex: 'npm install -g @openai/codex',
  opencode: 'curl -fsSL https://opencode.ai/install | bash',
}

export function AgentsPage() {
  const { isPending, data } = useQuery({ queryKey: ['installed-agents'], queryFn: api.installedAgents })
  const { t } = useI18n()

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header className="border-b flex items-center justify-between px-6 py-4">
        <div className="flex flex-col gap-0.5">
          <h1 className="flex items-center gap-2 text-lg font-semibold">
            <Bot className="size-5" />
            {t('agents.title')}
          </h1>
          <p className="text-muted-foreground text-sm">{t('agents.subtitle')}</p>
        </div>
      </header>

      <div className="flex-1 overflow-auto p-6">
        {isPending ? (
          <div className="grid gap-4 xl:grid-cols-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-64 w-full" />
            ))}
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            <p className="text-muted-foreground text-xs">{t('agents.templateHint')}</p>
            <div className="grid gap-4 xl:grid-cols-2">
              {(data ?? []).map((agent) => (
                <AgentCard key={agent.slug} agent={agent} />
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function AgentCard({ agent }: { agent: InstalledAgent }) {
  const { t } = useI18n()
  const templates = AGENT_TEMPLATES[agent.slug] ?? []
  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 flex-col gap-1">
            <CardTitle className="truncate">{agent.name}</CardTitle>
            <CardDescription className="font-mono">
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
            {t('agents.installHint')} {INSTALL_HINTS[agent.slug]}
          </p>
        )}
        <div className="flex flex-col gap-3">
          <span className="text-muted-foreground text-xs font-medium uppercase">
            {t('agents.templates')}
          </span>
          {templates.map((tpl) => (
            <TemplateBlock key={tpl.label} agent={agent.slug} template={tpl} />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function TemplateBlock({ agent, template }: { agent: string; template: AgentTemplate }) {
  const { t } = useI18n()
  const [writing, setWriting] = useState(false)

  const copy = async () => {
    await navigator.clipboard.writeText(template.content)
    toast.success(t('agents.copied'))
  }

  const writeDisk = async () => {
    setWriting(true)
    try {
      await api.saveAgentConfig(agent, template.name, template.content)
      toast.success(t('agents.written', { file: template.label }))
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
          <Button variant="outline" size="sm" disabled={writing} onClick={() => void writeDisk()}>
            <HardDriveDownload data-icon="inline-start" />
            {t('agents.writeDisk')}
          </Button>
        </div>
      </div>
      <pre className="bg-muted/40 max-h-64 overflow-auto rounded-lg border p-3 font-mono text-xs">
        {template.content}
      </pre>
    </div>
  )
}
