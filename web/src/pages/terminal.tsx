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
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { XtermConsole, type TerminalSession } from '@/components/terminal/xterm-console'
import { useProviders } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import { providerAgents } from '@/lib/types'

interface SessionInit {
  title: string
  command?: string
}

export function TerminalPage() {
  const [sessions, setSessions] = useState<TerminalSession[]>([])
  const [initializers, setInitializers] = useState<Record<string, string | undefined>>({})
  const [active, setActive] = useState<string>('')
  const { data: providers = [] } = useProviders()
  const { t } = useI18n()

  const spawn = (init: SessionInit) => {
    const id = `t-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`
    setSessions((prev) => [...prev, { id, title: init.title }])
    setInitializers((prev) => ({ ...prev, [id]: init.command }))
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
  }

  const quickLaunches: SessionInit[] = providers.flatMap((p) =>
    providerAgents(p).map((agent) => ({
      title: `${agent}@${p.slug}`,
      command: `aisw start ${agent} ${p.slug}`,
    }))
  )

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header className="border-b flex items-center justify-between gap-2 px-6 py-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h1 className="flex items-center gap-2 text-lg font-semibold">
            <SquareTerminal className="size-5" />
            {t('terminal.title')}
          </h1>
          <p className="text-muted-foreground text-sm">{t('terminal.subtitle')}</p>
        </div>
        <div className="flex gap-2">
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
        </div>
      </header>

      {sessions.length === 0 ? (
        <div className="flex flex-1 items-center justify-center p-6">
          <div className="text-muted-foreground flex max-w-md flex-col items-center gap-3 text-center">
            <SquareTerminal className="size-10" />
            <p className="text-sm">{t('terminal.empty.desc')}</p>
            <Button variant="outline" onClick={() => spawn({ title: 'shell' })}>
              <Plus data-icon="inline-start" />
              {t('terminal.open')}
            </Button>
          </div>
        </div>
      ) : (
        <Tabs value={active} onValueChange={setActive} className="flex min-h-0 flex-1 flex-col">
          <TabsList className="m-3 mb-0 w-fit">
            {sessions.map((session) => (
              <TabsTrigger key={session.id} value={session.id} className="gap-1.5">
                <span className="font-mono text-xs">{session.title}</span>
                <button
                  type="button"
                  aria-label={t('terminal.close', { name: session.title })}
                  className="hover:text-destructive -mr-1 cursor-pointer rounded-sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    close(session.id)
                  }}
                >
                  <X data-icon="inline-end" />
                </button>
              </TabsTrigger>
            ))}
          </TabsList>
          <div className="relative min-h-0 flex-1">
            {sessions.map((session) => (
              <div key={session.id} className="absolute inset-0">
                <XtermConsole
                  session={session}
                  active={session.id === active}
                  initialCommand={initializers[session.id]}
                />
              </div>
            ))}
          </div>
        </Tabs>
      )}
    </div>
  )
}
