import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Check, CloudDownload, Database, FileJson, FolderOpen, RefreshCw, Settings2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
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
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useI18n } from '@/lib/i18n'
import { api } from '@/lib/api'
import { queryKeys, useFetchMarket, useImportMarketModels, useMarketCategories, useMarketModels, useProviders, useSaveMarketSettings } from '@/lib/queries'
import type { MarketModel, MarketStorage } from '@/lib/types'
import { cn } from '@/lib/utils'

function formatContext(tokens?: number): string {
  if (!tokens) return '—'
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(tokens % 1_000_000 === 0 ? 0 : 1)}M`
  if (tokens >= 1000) return `${Math.round(tokens / 1000)}K`
  return String(tokens)
}

export function MarketPage() {
  const { t } = useI18n()
  const [category, setCategory] = useState('')
  const [search, setSearch] = useState('')
  const [deferredSearch, setDeferredSearch] = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [importOpen, setImportOpen] = useState(false)
  const [selected, setSelected] = useState<Set<string>>(new Set())

  useEffect(() => {
    const timer = setTimeout(() => setDeferredSearch(search.trim()), 250)
    return () => clearTimeout(timer)
  }, [search])

  const categories = useMarketCategories()
  const queryCategory = category === 'all' ? '' : category
  const models = useMarketModels(queryCategory, deferredSearch)
  const fetchMutation = useFetchMarket()
  const items = models.data?.items ?? []
  const hasData = models.data?.hasData ?? categories.data?.hasData ?? false
  const fetchedAt = models.data?.fetchedAt || categories.data?.fetchedAt || ''

  const toggle = (identifier: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(identifier)) next.delete(identifier)
      else next.add(identifier)
      return next
    })
  }
  const allSelected = items.length > 0 && items.every((m) => selected.has(m.identifier))
  const toggleAll = () => {
    setSelected(allSelected ? new Set() : new Set(items.map((m) => m.identifier)))
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header className="border-b flex items-center justify-between px-6 py-4">
        <div className="flex flex-col gap-0.5">
          <h1 className="text-lg font-semibold">{t('market.title')}</h1>
          <p className="text-muted-foreground text-sm">{t('market.subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          <StorageBadge storage={models.data?.storage} />
          <span className="text-muted-foreground text-xs">
            {fetchedAt
              ? t('market.fetchedAt', { time: new Date(fetchedAt).toLocaleString() })
              : t('market.neverFetched')}
          </span>
          <Button variant="outline" onClick={() => setSettingsOpen(true)}>
            <Settings2 data-icon="inline-start" />
            {t('market.settings')}
          </Button>
          <Button
            disabled={fetchMutation.isPending}
            onClick={() => fetchMutation.mutate()}
          >
            {fetchMutation.isPending ? (
              <Spinner data-icon="inline-start" />
            ) : (
              <CloudDownload data-icon="inline-start" />
            )}
            {fetchMutation.isPending ? t('market.fetching') : t('market.fetch')}
          </Button>
        </div>
      </header>

      <div className="border-b flex items-center gap-3 px-6 py-3">
        <Select value={category} onValueChange={setCategory}>
          <SelectTrigger className="w-56">
            <SelectValue placeholder={t('market.allCategories')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('market.allCategories')}</SelectItem>
            {(categories.data?.categories ?? []).map((c) => (
              <SelectItem key={c.category} value={c.category}>
                {c.category} ({c.count})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Input
          className="max-w-xs"
          placeholder={t('market.searchPlaceholder')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <span className="text-muted-foreground ml-auto text-xs">
          {models.isPending ? '' : t('market.total', { count: items.length })}
        </span>
      </div>

      <div className="flex-1 overflow-auto">
        {models.isPending && categories.isPending ? (
          <div className="flex flex-col gap-2 p-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : models.isError ? (
          <Empty className="m-auto max-w-md">
            <EmptyHeader>
              <EmptyTitle>{t('market.loadFailed')}</EmptyTitle>
              <EmptyDescription>{(models.error as Error)?.message}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : !hasData ? (
          <Empty className="m-auto max-w-md">
            <EmptyHeader>
              <EmptyTitle>{t('market.empty.title')}</EmptyTitle>
              <EmptyDescription>{t('market.empty.desc')}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button disabled={fetchMutation.isPending} onClick={() => fetchMutation.mutate()}>
                <CloudDownload data-icon="inline-start" />
                {t('market.fetch')}
              </Button>
            </EmptyContent>
          </Empty>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  <Button variant="ghost" size="icon" className="size-6" onClick={toggleAll}>
                    {allSelected ? <Check className="size-4" /> : <div className="border-muted-foreground/40 size-4 rounded-sm border" />}
                  </Button>
                </TableHead>
                <TableHead>{t('market.colModel')}</TableHead>
                <TableHead>{t('market.colVendor')}</TableHead>
                <TableHead className="text-right">{t('market.colContext')}</TableHead>
                <TableHead>{t('market.colAbilities')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((m) => (
                <MarketRow
                  key={m.identifier}
                  model={m}
                  checked={selected.has(m.identifier)}
                  onToggle={() => toggle(m.identifier)}
                />
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      <footer className="border-t flex items-center justify-between gap-3 px-6 py-3">
        <span className="text-muted-foreground text-sm">
          {t('market.selected', { count: selected.size })}
        </span>
        <Button disabled={selected.size === 0} onClick={() => setImportOpen(true)}>
          <FolderOpen data-icon="inline-start" />
          {t('market.import')}
        </Button>
      </footer>

      <MarketSettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
      <MarketImportDialog
        open={importOpen}
        onOpenChange={(open) => setImportOpen(open)}
        models={items.filter((m) => selected.has(m.identifier))}
        onDone={() => setSelected(new Set())}
      />
    </div>
  )
}

function StorageBadge({ storage }: { storage?: MarketStorage }) {
  const { t } = useI18n()
  if (!storage) return null
  const label =
    storage === 'file' ? t('market.storageFile') : storage === 'both' ? t('market.storageBoth') : t('market.storageSqlite')
  return (
    <Badge variant="secondary" className="hidden gap-1 lg:inline-flex">
      {storage === 'file' ? <FileJson className="size-3" /> : <Database className="size-3" />}
      {label}
    </Badge>
  )
}

function MarketRow({
  model,
  checked,
  onToggle,
}: {
  model: MarketModel
  checked: boolean
  onToggle: () => void
}) {
  const { t } = useI18n()
  const abilities = Object.entries(model.abilities ?? {})
    .filter(([, on]) => on)
    .map(([name]) => name)
  return (
    <TableRow className={cn('cursor-pointer', checked && 'bg-accent/50')} onClick={onToggle}>
      <TableCell>
        {checked ? (
          <Check className="size-4" />
        ) : (
          <div className="border-muted-foreground/40 size-4 rounded-sm border" />
        )}
      </TableCell>
      <TableCell>
        <div className="flex flex-col">
          <span className="font-medium">{model.displayName || model.identifier}</span>
          <span className="text-muted-foreground font-mono text-xs">{model.identifier}</span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex flex-wrap gap-1">
          <Badge variant="outline">{model.category}</Badge>
          {(model.providers ?? [])
            .filter((p) => p !== model.category)
            .slice(0, 3)
            .map((p) => (
              <Badge key={p} variant="secondary">
                {p}
              </Badge>
            ))}
          {(model.providerCount ?? 0) > 4 && (
            <Badge variant="secondary">+{model.providerCount! - 4}</Badge>
          )}
          {model.enabled === false && <Badge variant="destructive">{t('market.disabled')}</Badge>}
        </div>
      </TableCell>
      <TableCell className="text-right font-mono text-xs">
        {formatContext(model.contextWindowTokens)}
      </TableCell>
      <TableCell>
        <div className="flex flex-wrap gap-1">
          {abilities.map((name) => (
            <Badge key={name} variant="secondary">
              {name}
            </Badge>
          ))}
        </div>
      </TableCell>
    </TableRow>
  )
}

function MarketSettingsDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useI18n()
  const [override, setOverride] = useState<MarketStorage | null>(null)
  const saveMutation = useSaveMarketSettings()
  const settingsQuery = useQuery({
    queryKey: queryKeys.marketSettings,
    queryFn: api.marketSettings,
    enabled: open,
  })
  const storage = override ?? settingsQuery.data?.storage ?? ''

  const close = (next: boolean) => onOpenChange(next)

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('market.settings')}</DialogTitle>
          <DialogDescription>{t('market.settingsDesc')}</DialogDescription>
        </DialogHeader>
        <Field>
          <FieldLabel>{t('market.storage')}</FieldLabel>
          <Select value={storage} onValueChange={(v) => setOverride(v as MarketStorage)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="sqlite">{t('market.storageSqlite')}</SelectItem>
              <SelectItem value="file">{t('market.storageFile')}</SelectItem>
              <SelectItem value="both">{t('market.storageBoth')}</SelectItem>
            </SelectContent>
          </Select>
        </Field>
        <DialogFooter>
          <Button variant="outline" onClick={() => close(false)}>
            {t('market.cancel')}
          </Button>
          <Button
            disabled={!storage || saveMutation.isPending}
            onClick={() =>
              saveMutation.mutate(
                { storage: storage as MarketStorage },
                { onSuccess: () => close(false) },
              )
            }
          >
            {saveMutation.isPending && <Spinner data-icon="inline-start" />}
            {t('market.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function MarketImportDialog({
  open,
  onOpenChange,
  models,
  onDone,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  models: MarketModel[]
  onDone: () => void
}) {
  const { t } = useI18n()
  const providers = useProviders()
  const importMutation = useImportMarketModels()
  const [providerPick, setProviderPick] = useState('')

  const providerList = useMemo(() => providers.data ?? [], [providers.data])
  const provider = providerPick || providerList[0]?.slug || ''

  const close = (next: boolean) => {
    onOpenChange(next)
    if (!next) {
      importMutation.reset()
      onDone()
    }
  }

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('market.import')}</DialogTitle>
          <DialogDescription>
            {t('market.importPickProvider')} · {t('market.selected', { count: models.length })}
          </DialogDescription>
        </DialogHeader>
        {providerList.length === 0 ? (
          <p className="text-muted-foreground text-sm">{t('market.noProviders')}</p>
        ) : (
          <>
            <Field>
              <FieldLabel>Provider</FieldLabel>
              <Select value={provider} onValueChange={setProviderPick}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {providerList.map((p) => (
                    <SelectItem key={p.slug} value={p.slug}>
                      {p.name} ({p.slug})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <div className="bg-accent/50 rounded-md p-3 font-mono text-xs">
              {models.slice(0, 8).map((m) => (
                <div key={m.identifier} className="truncate">
                  {m.identifier}
                </div>
              ))}
              {models.length > 8 && <div className="text-muted-foreground">+{models.length - 8}…</div>}
            </div>
            <p className="text-muted-foreground text-xs">{t('market.importHint')}</p>
          </>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={() => close(false)}>
            {t('market.cancel')}
          </Button>
          <Button
            disabled={!provider || models.length === 0 || importMutation.isPending || providerList.length === 0}
            onClick={() =>
              importMutation.mutate(
                { provider, models: models.map((m) => m.identifier) },
                { onSuccess: () => close(false) },
              )
            }
          >
            {importMutation.isPending && <Spinner data-icon="inline-start" />}
            <RefreshCw data-icon="inline-start" />
            {t('market.importSubmit')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
