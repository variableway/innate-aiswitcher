import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import { FileCode2, Languages, Plug, SquareTerminal, Zap } from 'lucide-react'
import { Toaster } from '@/components/ui/sonner'
import { Button } from '@/components/ui/button'
import { useI18n, type TranslationKey } from '@/lib/i18n'
import { cn } from '@/lib/utils'

const NAV: { to: string; labelKey: TranslationKey; icon: typeof Plug }[] = [
  { to: '/providers', labelKey: 'nav.providers', icon: Plug },
  { to: '/profiles', labelKey: 'nav.profiles', icon: Zap },
  { to: '/terminal', labelKey: 'nav.terminal', icon: SquareTerminal },
  { to: '/configs', labelKey: 'nav.configs', icon: FileCode2 },
]

export function AppShell() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const { lang, setLang, t } = useI18n()
  return (
    <div className="bg-background text-foreground flex h-svh">
      <aside className="bg-card border-r flex w-56 shrink-0 flex-col gap-1 p-3">
        <div className="flex items-center gap-2 px-2 py-3">
          <div className="bg-primary text-primary-foreground flex size-8 items-center justify-center rounded-md font-semibold">
            ai
          </div>
          <div className="flex flex-col">
            <span className="text-sm font-semibold">AISwitcher</span>
            <span className="text-muted-foreground text-xs">{t('app.tagline')}</span>
          </div>
        </div>
        <nav className="flex flex-col gap-1 pt-2">
          {NAV.map(({ to, labelKey, icon: Icon }) => {
            const active = pathname === to || pathname.startsWith(`${to}/`)
            return (
              <Link
                key={to}
                to={to}
                className={cn(
                  'flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium',
                  active
                    ? 'bg-accent text-accent-foreground'
                    : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'
                )}
              >
                <Icon data-icon="inline-start" />
                {t(labelKey)}
              </Link>
            )
          })}
        </nav>
        <div className="mt-auto flex flex-col gap-2 px-1 pb-1">
          <Button
            variant="outline"
            size="sm"
            className="text-muted-foreground justify-start"
            onClick={() => setLang(lang === 'zh' ? 'en' : 'zh')}
          >
            <Languages data-icon="inline-start" />
            {lang === 'zh' ? 'English' : '中文'}
          </Button>
          <span className="text-muted-foreground px-2 text-xs">{t('app.agents')}</span>
        </div>
      </aside>
      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Outlet />
      </main>
      <Toaster position="bottom-right" />
    </div>
  )
}
