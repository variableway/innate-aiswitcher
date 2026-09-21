import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import { useTheme } from 'next-themes'
import {
  Bot,
  FileCode2,
  Globe,
  Menu,
  Moon,
  Plug,
  SquareTerminal,
  Store,
  Sun,
  Trophy,
  Zap,
} from 'lucide-react'
import { Toaster } from '@/components/ui/sonner'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useAgents } from '@/lib/queries'
import { useI18n, type TranslationKey } from '@/lib/i18n'
import { cn } from '@/lib/utils'

const NAV: { to: string; labelKey: TranslationKey; icon: typeof Plug }[] = [
  { to: '/providers', labelKey: 'nav.providers', icon: Plug },
  { to: '/agents', labelKey: 'nav.agents', icon: Bot },
  { to: '/profiles', labelKey: 'nav.profiles', icon: Zap },
  { to: '/terminal', labelKey: 'nav.terminal', icon: SquareTerminal },
  { to: '/configs', labelKey: 'nav.configs', icon: FileCode2 },
  { to: '/market', labelKey: 'nav.market', icon: Store },
  { to: '/rankings', labelKey: 'nav.rankings', icon: Trophy },
]

function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme()
  const { t } = useI18n()
  const dark = resolvedTheme === 'dark'
  return (
    <Button
      variant="outline"
      size="sm"
      className="text-muted-foreground justify-start"
      aria-label={dark ? t('theme.toLight') : t('theme.toDark')}
      title={dark ? t('theme.toLight') : t('theme.toDark')}
      onClick={() => setTheme(dark ? 'light' : 'dark')}
    >
      {dark ? <Sun data-icon="inline-start" /> : <Moon data-icon="inline-start" />}
      {dark ? t('theme.toLight') : t('theme.toDark')}
    </Button>
  )
}

function LangToggle({ className }: { className?: string }) {
  const { lang, setLang } = useI18n()
  return (
    <Button
      variant="outline"
      size="sm"
      className={cn('text-muted-foreground justify-start', className)}
      onClick={() => setLang(lang === 'zh' ? 'en' : 'zh')}
    >
      <Globe data-icon="inline-start" />
      {lang === 'zh' ? 'EN' : '中文'}
    </Button>
  )
}

function Brand() {
  const { t } = useI18n()
  return (
    <div className="flex items-center gap-2 px-2 py-3">
      <div className="bg-primary text-primary-foreground flex size-8 items-center justify-center rounded-md font-semibold">
        ai
      </div>
      <div className="flex flex-col">
        <span className="text-sm font-semibold">AISwitcher</span>
        <span className="text-muted-foreground text-xs">{t('app.tagline')}</span>
      </div>
    </div>
  )
}

export function AppShell() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const { data: agents = [] } = useAgents()
  const { t } = useI18n()
  const isActive = (to: string) => pathname === to || pathname.startsWith(`${to}/`)

  return (
    <div className="bg-background text-foreground flex h-svh flex-col md:flex-row">
      {/* Narrow screens: top bar with a nav menu instead of the sidebar. */}
      <div className="bg-sidebar text-sidebar-foreground border-sidebar-border flex items-center justify-between gap-2 border-b px-3 md:hidden">
        <Brand />
        <div className="flex items-center gap-2">
          <LangToggle />
          <ThemeToggle />
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" aria-label={t('app.menu')}>
                <Menu data-icon="inline-start" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <DropdownMenuLabel>AISwitcher</DropdownMenuLabel>
              <DropdownMenuGroup>
                {NAV.map(({ to, labelKey, icon: Icon }) => (
                  <DropdownMenuItem key={to} asChild>
                    <Link to={to} className={cn(isActive(to) && 'bg-accent text-accent-foreground')}>
                      <Icon data-icon="inline-start" />
                      {t(labelKey)}
                    </Link>
                  </DropdownMenuItem>
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <aside className="bg-sidebar text-sidebar-foreground border-sidebar-border hidden w-56 shrink-0 flex-col gap-1 border-r p-3 md:flex">
        <Brand />
        <nav className="flex flex-col gap-1 pt-2">
          {NAV.map(({ to, labelKey, icon: Icon }) => {
            const active = isActive(to)
            return (
              <Link
                key={to}
                to={to}
                className={cn(
                  'flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium',
                  active
                    ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                    : 'text-muted-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-foreground'
                )}
              >
                <Icon data-icon="inline-start" />
                {t(labelKey)}
              </Link>
            )
          })}
        </nav>
        <div className="mt-auto flex flex-col gap-2 px-1 pb-1">
          <LangToggle />
          <ThemeToggle />
          <span className="text-muted-foreground truncate px-2 text-xs" title={agents.map((a) => a.name).join(' · ')}>
            {agents.map((a) => a.name).join(' · ')}
          </span>
        </div>
      </aside>
      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Outlet />
      </main>
      <Toaster position="bottom-right" />
    </div>
  )
}
