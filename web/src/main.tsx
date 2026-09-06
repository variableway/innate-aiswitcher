import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { router } from '@/routes'
import { LangProvider } from '@/lib/i18n'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 10_000, retry: 1 },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <LangProvider>
        <RouterProvider router={router} />
      </LangProvider>
    </QueryClientProvider>
  </StrictMode>
)
