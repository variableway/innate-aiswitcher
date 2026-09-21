import { useEffect, useState } from 'react'
import { CircleHelp, ExternalLink } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useI18n, type TranslationKey } from '@/lib/i18n'
import { useRankings } from '@/lib/queries'
import type { RankingItem, RankingMetric } from '@/lib/types'
import { cn } from '@/lib/utils'

/** Tab order for the metric picker; the metrics-info dialog lists whatever the backend catalogs. */
const METRIC_IDS = [
  'coding',
  'intelligence',
  'agentic',
  'price_input',
  'price_output',
  'context',
  'output_limit',
  'released',
] as const

const METRIC_LABEL_KEYS: Record<string, TranslationKey> = {
  coding: 'rankings.metric.coding',
  intelligence: 'rankings.metric.intelligence',
  agentic: 'rankings.metric.agentic',
  price_input: 'rankings.metric.price_input',
  price_output: 'rankings.metric.price_output',
  context: 'rankings.metric.context',
  output_limit: 'rankings.metric.output_limit',
  released: 'rankings.metric.released',
}

/** Bilingual descriptions for the known metrics; anything the backend adds later falls back to its own description. */
const METRIC_DESC_KEYS: Record<string, TranslationKey> = {
  coding: 'rankings.desc.coding',
  intelligence: 'rankings.desc.intelligence',
  agentic: 'rankings.desc.agentic',
  price_input: 'rankings.desc.price_input',
  price_output: 'rankings.desc.price_output',
  context: 'rankings.desc.context',
  output_limit: 'rankings.desc.output_limit',
  released: 'rankings.desc.released',
}

function formatContext(tokens?: number): string {
  if (!tokens) return '—'
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(tokens % 1_000_000 === 0 ? 0 : 1)}M`
  if (tokens >= 1000) return `${Math.round(tokens / 1000)}K`
  return String(tokens)
}

function formatIndex(value?: number): string {
  if (value == null) return '—'
  return String(Math.round(value * 10) / 10)
}

function formatPrice(value?: number): string {
  if (value == null) return '—'
  return `$${value.toFixed(2).replace(/\.00$/, '').replace(/(\.\d)0$/, '$1')}`
}

export function RankingsPage() {
  const { t } = useI18n()
  const [metric, setMetric] = useState<string>('coding')
  const [search, setSearch] = useState('')
  const [deferredSearch, setDeferredSearch] = useState('')
  const [infoOpen, setInfoOpen] = useState(false)
  const [detail, setDetail] = useState<RankingItem | null>(null)

  useEffect(() => {
    const timer = setTimeout(() => setDeferredSearch(search.trim()), 250)
    return () => clearTimeout(timer)
  }, [search])

  const rankings = useRankings(metric, deferredSearch)
  const items = rankings.data?.items ?? []
  const total = rankings.data?.total ?? 0
  const degraded = (rankings.data?.sources ?? []).filter((s) => !s.ok)

  const metricLabel = (id: string, fallback?: string): string => {
    const key = METRIC_LABEL_KEYS[id]
    return key ? t(key) : (fallback ?? id)
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header className="border-b flex items-center justify-between px-6 py-4">
        <div className="flex flex-col gap-0.5">
          <h1 className="text-lg font-semibold">{t('rankings.title')}</h1>
          <p className="text-muted-foreground text-sm">{t('rankings.subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          {degraded.map((s) => (
            <Badge key={s.id} variant="destructive" title={s.detail}>
              {t('rankings.sourceDegraded', { name: s.name })}
            </Badge>
          ))}
          <span className="text-muted-foreground text-xs">
            {rankings.data?.fetchedAt
              ? t('rankings.fetchedAt', { time: new Date(rankings.data.fetchedAt).toLocaleString() })
              : ''}
          </span>
          <Button variant="outline" onClick={() => setInfoOpen(true)}>
            <CircleHelp data-icon="inline-start" />
            {t('rankings.metricsInfo')}
          </Button>
        </div>
      </header>

      <div className="border-b flex items-center gap-3 px-6 py-3">
        <Tabs value={metric} onValueChange={setMetric}>
          <TabsList className="max-w-full overflow-x-auto">
            {METRIC_IDS.map((id) => (
              <TabsTrigger key={id} value={id}>
                {metricLabel(id)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <Input
          className="max-w-xs"
          placeholder={t('rankings.searchPlaceholder')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <span className="text-muted-foreground ml-auto text-xs whitespace-nowrap">
          {rankings.isPending ? '' : t('rankings.total', { count: total })}
          {items.length > 0 && total > items.length
            ? ` · ${t('rankings.showing', { count: items.length })}`
            : ''}
        </span>
      </div>

      <div className="flex-1 overflow-auto">
        {rankings.isPending ? (
          <div className="flex flex-col gap-2 p-6">
            {Array.from({ length: 10 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : rankings.isError ? (
          <Empty className="m-auto max-w-md">
            <EmptyHeader>
              <EmptyTitle>{t('rankings.loadFailed')}</EmptyTitle>
              <EmptyDescription>{(rankings.error as Error)?.message}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : items.length === 0 ? (
          <Empty className="m-auto max-w-md">
            <EmptyHeader>
              <EmptyTitle>{t('rankings.empty.title')}</EmptyTitle>
              <EmptyDescription>{t('rankings.empty.desc')}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button variant="outline" onClick={() => setSearch('')}>
                {t('rankings.empty.reset')}
              </Button>
            </EmptyContent>
          </Empty>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10 text-right">#</TableHead>
                <TableHead>{t('rankings.colModel')}</TableHead>
                <TableHead>{t('rankings.colVendor')}</TableHead>
                <TableHead className={cn('text-right', metric === 'intelligence' && 'text-foreground font-medium')}>
                  {metricLabel('intelligence')}
                </TableHead>
                <TableHead className={cn('text-right', metric === 'coding' && 'text-foreground font-medium')}>
                  {metricLabel('coding')}
                </TableHead>
                <TableHead className={cn('text-right', metric === 'agentic' && 'text-foreground font-medium')}>
                  {metricLabel('agentic')}
                </TableHead>
                <TableHead className={cn('text-right', metric === 'price_input' && 'text-foreground font-medium')}>
                  {t('rankings.colPriceIn')}
                </TableHead>
                <TableHead className={cn('text-right', metric === 'price_output' && 'text-foreground font-medium')}>
                  {t('rankings.colPriceOut')}
                </TableHead>
                <TableHead className={cn('text-right', metric === 'context' && 'text-foreground font-medium')}>
                  {metricLabel('context')}
                </TableHead>
                <TableHead>{t('rankings.colAbilities')}</TableHead>
                <TableHead className={cn('text-right', metric === 'released' && 'text-foreground font-medium')}>
                  {t('rankings.colReleased')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item, index) => (
                <RankingRow
                  key={`${item.providerId}/${item.modelId}`}
                  item={item}
                  rank={index + 1}
                  metric={metric}
                  onOpen={() => setDetail(item)}
                />
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      <MetricsInfoDialog open={infoOpen} onOpenChange={setInfoOpen} metrics={rankings.data?.metrics ?? []} />
      <ModelDetailDialog item={detail} onOpenChange={(open) => !open && setDetail(null)} />
    </div>
  )
}

function RankingRow({
  item,
  rank,
  metric,
  onOpen,
}: {
  item: RankingItem
  rank: number
  metric: string
  onOpen: () => void
}) {
  const { t } = useI18n()
  const abilities: TranslationKey[] = []
  if (item.reasoning) abilities.push('rankings.ability.reasoning')
  if (item.toolCall) abilities.push('rankings.ability.tools')
  if (item.attachment) abilities.push('rankings.ability.vision')
  if (item.openWeights) abilities.push('rankings.ability.openWeights')

  return (
    <TableRow className="cursor-pointer" onClick={onOpen}>
      <TableCell className="text-muted-foreground text-right font-mono text-xs">{rank}</TableCell>
      <TableCell>
        <div className="flex flex-col">
          <span className="font-medium">{item.name}</span>
          <span className="text-muted-foreground font-mono text-xs">{item.modelId}</span>
        </div>
      </TableCell>
      <TableCell>
        <Badge variant="outline">{item.providerName}</Badge>
      </TableCell>
      <TableCell
        className={cn(
          'text-right font-mono text-xs',
          metric === 'intelligence' && item.intelligence != null && 'font-medium text-foreground',
        )}
      >
        {formatIndex(item.intelligence)}
      </TableCell>
      <TableCell
        className={cn(
          'text-right font-mono text-xs',
          metric === 'coding' && item.coding != null && 'font-medium text-foreground',
        )}
      >
        {formatIndex(item.coding)}
      </TableCell>
      <TableCell
        className={cn(
          'text-right font-mono text-xs',
          metric === 'agentic' && item.agentic != null && 'font-medium text-foreground',
        )}
      >
        {formatIndex(item.agentic)}
      </TableCell>
      <TableCell className="text-right font-mono text-xs">{formatPrice(item.inputPrice)}</TableCell>
      <TableCell className="text-right font-mono text-xs">{formatPrice(item.outputPrice)}</TableCell>
      <TableCell className="text-right font-mono text-xs">
        {formatContext(metric === 'output_limit' ? item.outputLimit : item.context)}
      </TableCell>
      <TableCell>
        <div className="flex flex-wrap gap-1">
          {abilities.map((key) => (
            <Badge key={key} variant="secondary">
              {t(key)}
            </Badge>
          ))}
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground text-right font-mono text-xs">
        {item.releaseDate || '—'}
      </TableCell>
    </TableRow>
  )
}

function MetricsInfoDialog({
  open,
  onOpenChange,
  metrics,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  metrics: RankingMetric[]
}) {
  const { t } = useI18n()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('rankings.metricsInfo')}</DialogTitle>
          <DialogDescription>{t('rankings.metricsInfoDesc')}</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          {metrics.map((m) => (
            <div key={m.id} className="flex flex-col gap-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium">
                  {METRIC_LABEL_KEYS[m.id] ? t(METRIC_LABEL_KEYS[m.id]) : m.name}
                </span>
                <Badge variant="secondary">
                  {m.source === 'artificial_analysis'
                    ? t('rankings.source.artificial_analysis')
                    : t('rankings.source.models_dev')}
                </Badge>
                <Badge variant="outline">
                  {m.direction === 'asc' ? t('rankings.dir.asc') : t('rankings.dir.desc')}
                </Badge>
              </div>
              <p className="text-muted-foreground text-xs">
                {METRIC_DESC_KEYS[m.id] ? t(METRIC_DESC_KEYS[m.id]) : m.description}
              </p>
            </div>
          ))}
          <p className="text-muted-foreground text-xs">{t('rankings.compareRule')}</p>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function ModelDetailDialog({
  item,
  onOpenChange,
}: {
  item: RankingItem | null
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useI18n()
  return (
    <Dialog open={item != null} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        {item && (
          <>
            <DialogHeader>
              <DialogTitle>{item.name}</DialogTitle>
              <DialogDescription>
                {item.providerName} · <span className="font-mono">{item.modelId}</span>
              </DialogDescription>
            </DialogHeader>
            <div className="flex flex-wrap gap-2">
              <Badge variant="secondary">
                {t('rankings.metric.intelligence')} {formatIndex(item.intelligence)}
              </Badge>
              <Badge variant="secondary">
                {t('rankings.metric.coding')} {formatIndex(item.coding)}
              </Badge>
              <Badge variant="secondary">
                {t('rankings.metric.agentic')} {formatIndex(item.agentic)}
              </Badge>
              <Badge variant="outline">
                {t('rankings.colPriceIn')} {formatPrice(item.inputPrice)} /{' '}
                {t('rankings.colPriceOut')} {formatPrice(item.outputPrice)}
              </Badge>
              <Badge variant="outline">
                {t('rankings.metric.context')} {formatContext(item.context)}
              </Badge>
              {item.outputLimit ? (
                <Badge variant="outline">
                  {t('rankings.metric.output_limit')} {formatContext(item.outputLimit)}
                </Badge>
              ) : null}
              {item.knowledge ? (
                <Badge variant="outline">
                  {t('rankings.knowledge')} {item.knowledge}
                </Badge>
              ) : null}
              {item.releaseDate ? (
                <Badge variant="outline">
                  {t('rankings.colReleased')} {item.releaseDate}
                </Badge>
              ) : null}
            </div>
            <div className="flex flex-col gap-2">
              <span className="text-sm font-medium">{t('rankings.vendorBenchTitle')}</span>
              {item.benchmarks && item.benchmarks.length > 0 ? (
                <>
                  <p className="text-muted-foreground text-xs">{t('rankings.vendorBenchDisclaimer')}</p>
                  <div className="max-h-64 overflow-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('rankings.bench.colName')}</TableHead>
                          <TableHead className="text-right">{t('rankings.bench.colScore')}</TableHead>
                          <TableHead>{t('rankings.bench.colSource')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {item.benchmarks.map((bench, i) => (
                          <TableRow key={`${bench.name}-${i}`}>
                            <TableCell className="font-medium">
                              {bench.name}
                              {bench.version ? (
                                <span className="text-muted-foreground font-mono text-xs"> v{bench.version}</span>
                              ) : null}
                            </TableCell>
                            <TableCell className="text-right font-mono text-xs">
                              {bench.score}
                              {bench.metric ? (
                                <span className="text-muted-foreground"> · {bench.metric}</span>
                              ) : null}
                            </TableCell>
                            <TableCell className="max-w-40">
                              {bench.source ? (
                                <a
                                  href={bench.source}
                                  target="_blank"
                                  rel="noreferrer"
                                  className="text-primary inline-flex items-center gap-1 text-xs hover:underline"
                                >
                                  <ExternalLink className="size-3" />
                                  {bench.date || bench.source}
                                </a>
                              ) : (
                                <span className="text-muted-foreground text-xs">—</span>
                              )}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </>
              ) : (
                <p className="text-muted-foreground text-xs">{t('rankings.noBenchmarks')}</p>
              )}
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
