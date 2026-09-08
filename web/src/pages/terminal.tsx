import { useState } from 'react'
import { ChevronDown, Plus, SquareTerminal, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { PageHeader } from '@/components/page-header'
import { XtermConsole, type TerminalSession } from '@/components/terminal/xterm-console'
import { useProviders } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import { providerAgents } from '@/lib/types'

type SessionInit = Pick<TerminalSession, 'title' | 'command'>

export function TerminalPage() {
  const [sessions, setSessions] = useState<TerminalSession[]>([])
  /** Sessions whose process already exited — safe to close without asking. */
  const [exited, setExited] = useState<Record<string, boolean>>({})
  const [active, setActive] = useState<string>('')
  const [closing, setClosing] = useState<TerminalSession | null>(null)
  const { data: providers = [] } = useProviders()
  const { t } = useI18n()

  const spawn = (init: SessionInit) => {
    // Auto-number duplicate tab titles: shell, shell #2, shell #3…
    const sameBase = sessions.filter(
      (s) => s.title === init.title || s.title.startsWith(`${init.title} #`)
    )
    const title = sameBase.length === 0 ? init.title : `${init.title} #${sameBase.length + 1}`
    const id = `t-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`
    setSessions((prev) => [...prev, { id, title, command: init.command }])
    setActive(id)
  }

  const close = (id: string) => {
    setSessions((prev) => {
      const next = prev.filter((s) => s.id !== id)
      if (active === id && next.length > 0) {
        setActive(next[next.length - 1].id)
      } else if (next.length === 0) {
        setActive('')
      }
      return next
    })
    setExited((prev) => {
      const next = { ...prev }
      delete next[id]
      return next
    })
    setClosing((cur) => (cur?.id === id ? null : cur))
  }

  const requestClose = (session: TerminalSession) => {
    if (exited[session.id]) close(session.id)
    else setClosing(session)
  }

  const quickLaunches: SessionInit[] = providers.flatMap((p) =>
    providerAgents(p).map((agent) => ({
      title: `${agent}@${p.slug}`,
      command: `aisw start ${agent} ${p.slug}`,
    }))
  )

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <PageHeader
        icon={SquareTerminal}
        title={t('terminal.title')}
        subtitle={t('terminal.subtitle')}
        actions={
          <>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline">
                  {t('terminal.launch')}
                  <ChevronDown data-icon="inline-end" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-52">
                <DropdownMenuLabel>aisw start</DropdownMenuLabel>
                <DropdownMenuGroup>
                  {quickLaunches.length === 0 && (
                    <DropdownMenuItem disabled>{t('terminal.noProviders')}</DropdownMenuItem>
                  )}
                  {quickLaunches.map((launch) => (
                    <DropdownMenuItem
                      key={launch.title}
                      onClick={() => spawn(launch)}
                      className="font-mono text-xs"
                    >
                      {launch.command}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuGroup>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => spawn({ title: 'shell' })}>
                  {t('terminal.plainShell')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <Button onClick={() => spawn({ title: 'shell' })}>
              <Plus data-icon="inline-start" />
              {t('terminal.new')}
            </Button>
          </>
        }
      />

      {sessions.length === 0 ? (
        <div className="flex flex-1 items-center justify-center p-6">
          <Empty className="max-w-md">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <SquareTerminal />
              </EmptyMedia>
              <EmptyTitle>{t('terminal.title')}</EmptyTitle>
              <EmptyDescription>{t('terminal.empty.desc')}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button variant="outline" onClick={() => spawn({ title: 'shell' })}>
                <Plus data-icon="inline-start" />
                {t('terminal.open')}
              </Button>
            </EmptyContent>
          </Empty>
        </div>
      ) : (
        <Tabs value={active} onValueChange={setActive} className="flex min-h-0 flex-1 flex-col">
          <TabsList className="m-3 mb-0 w-fit max-w-full overflow-x-auto">
            {sessions.map((session) => (
              <TabsTrigger key={session.id} value={session.id} className="gap-1.5">
                <span className="font-mono text-xs">{session.title}</span>
                <span
                  role="button"
                  tabIndex={0}
                  aria-label={t('terminal.close', { name: session.title })}
                  className="hover:text-destructive -mr-1 cursor-pointer rounded-sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    requestClose(session)
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.stopPropagation()
                      e.preventDefault()
                      requestClose(session)
                    }
                  }}
                >
                  <X data-icon="inline-end" />
                </span>
              </TabsTrigger>
            ))}
          </TabsList>
          <div className="relative min-h-0 flex-1">
            {sessions.map((session) => (
              <div key={session.id} className="absolute inset-0">
                <XtermConsole
                  session={session}
                  active={session.id === active}
                  onExit={(id) => setExited((prev) => ({ ...prev, [id]: true }))}
                />
              </div>
            ))}
          </div>
        </Tabs>
      )}

      <ConfirmDialog
        open={closing !== null}
        onOpenChange={(open) => !open && setClosing(null)}
        title={t('terminal.closeConfirm.title')}
        description={
          closing ? t('terminal.closeConfirm.desc', { name: closing.title }) : undefined
        }
        confirmLabel={t('terminal.closeConfirm.confirm')}
        onConfirm={() => closing && close(closing.id)}
      />
    </div>
  )
}
