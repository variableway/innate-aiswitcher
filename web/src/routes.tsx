import { createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { AppShell } from '@/components/app-shell'
import { ProvidersPage } from '@/pages/providers'
import { ProfilesPage } from '@/pages/profiles'
import { TerminalPage } from '@/pages/terminal'
import { ConfigsPage } from '@/pages/configs'
import { AgentsPage } from '@/pages/agents'
import { MarketPage } from '@/pages/market'

const rootRoute = createRootRoute({
  component: AppShell,
})

const providersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/providers',
  component: ProvidersPage,
})

const profilesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/profiles',
  component: ProfilesPage,
})

const terminalRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/terminal',
  component: TerminalPage,
})

const agentsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/agents',
  component: AgentsPage,
})

const configsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/configs',
  component: ConfigsPage,
})

const marketRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/market',
  component: MarketPage,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: () => <ProvidersPage />,
})

const routeTree = rootRoute.addChildren([indexRoute, agentsRoute, providersRoute, profilesRoute, terminalRoute, configsRoute, marketRoute])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
