'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  AlertTriangle,
  ArrowRight,
  Bot,
  Chrome,
  Clock,
  Code2,
  Cpu,
  Flame,
  Globe2,
  RefreshCw,
  Search,
  Server,
  ShieldQuestion,
  Sparkles,
  Terminal,
  TrendingUp,
  Wrench,
} from 'lucide-react'
import {
  BoundedTable,
  DetailModal,
  EmptyState,
  FilterBar,
  KpiTile,
  SummaryStrip,
  WorkbenchHeader,
} from '@/components/workbench/primitives'

export type AbomItem = {
  id: string
  category: string
  product: string
  capability_tags: string[]
  confidence: 'high' | 'medium' | 'low' | string
  device_ids: string[]
  user_context?: string
  evidence_refs: string[]
  first_seen_ms: number
  last_seen_ms: number
  recommended_review: string
}

export type AbomFleetResponse = {
  ok?: boolean
  generated_at_ms?: number
  total_items?: number
  category_count?: Record<string, number>
  items?: AbomItem[]
  empty_help?: string
}

export type AbomDeviceResponse = {
  ok?: boolean
  device_id?: string
  generated_at_ms?: number
  total_items?: number
  items?: AbomItem[]
  empty_help?: string
}

export type AbomInsightItem = {
  id: string
  category: string
  product: string
  confidence: string
  device_ids: string[]
  device_count: number
  first_seen_ms: number
  last_seen_ms: number
  reason: string
  high_attention?: boolean
}

export type AbomInsightSection = {
  id: string
  title: string
  detail: string
  threshold?: string
  items: AbomInsightItem[]
  total: number
}

export type AbomEndpointHotspot = {
  device_id: string
  item_count: number
  high_confidence_count: number
  low_confidence_count: number
  categories: string[]
  last_seen_ms: number
  high_attention: boolean
  reason: string
}

export type AbomInsightsResponse = {
  ok?: boolean
  generated_at_ms?: number
  window_ms?: number
  stale_after_ms?: number
  fleet_size?: number
  high_attention_devices?: string[]
  sections?: AbomInsightSection[]
  hotspots?: AbomEndpointHotspot[]
  empty_help?: string
}

const TIME_WINDOWS: { id: string; label: string; ms: number }[] = [
  { id: '24h', label: 'Last 24h', ms: 24 * 60 * 60 * 1000 },
  { id: '3d', label: 'Last 3 days', ms: 3 * 24 * 60 * 60 * 1000 },
  { id: '7d', label: 'Last 7 days', ms: 7 * 24 * 60 * 60 * 1000 },
  { id: '30d', label: 'Last 30 days', ms: 30 * 24 * 60 * 60 * 1000 },
]

const INSIGHT_ICONS: Record<string, typeof TrendingUp> = {
  newly_observed: TrendingUp,
  newly_observed_high_attention: AlertTriangle,
  high_confidence: Sparkles,
  low_confidence_needs_review: ShieldQuestion,
  widespread: Flame,
  stale: Clock,
}

export const ABOM_CATEGORY_ORDER = [
  'ai_desktop_app',
  'browser_ai_extension',
  'coding_agent',
  'cli_agent',
  'mcp_endpoint',
  'local_model_runtime',
  'model_gateway',
  'unknown_ai_automation',
] as const

export type AbomCategory = (typeof ABOM_CATEGORY_ORDER)[number]

const CATEGORY_DEFS: Record<
  AbomCategory,
  { label: string; description: string; icon: typeof Bot }
> = {
  ai_desktop_app: {
    label: 'AI desktop apps',
    description: 'Vendor desktop clients such as ChatGPT, Claude, Copilot, or Gemini.',
    icon: Sparkles,
  },
  browser_ai_extension: {
    label: 'Browser AI extensions',
    description: 'Browser extensions with AI naming or session activity.',
    icon: Chrome,
  },
  coding_agent: {
    label: 'Coding agents',
    description: 'IDE-class agents like Cursor, Continue, GitHub Copilot, and Tabnine.',
    icon: Code2,
  },
  cli_agent: {
    label: 'CLI agents',
    description: 'Shell-driven agents such as codex, claude-code, aider, gemini-cli.',
    icon: Terminal,
  },
  mcp_endpoint: {
    label: 'MCP endpoints',
    description: 'Model Context Protocol servers and clients.',
    icon: Wrench,
  },
  local_model_runtime: {
    label: 'Local model runtimes',
    description: 'Ollama, vLLM, llama.cpp, LM Studio, and similar runtimes.',
    icon: Cpu,
  },
  model_gateway: {
    label: 'Model gateways',
    description: 'External APIs and enterprise gateways acting as model providers.',
    icon: Globe2,
  },
  unknown_ai_automation: {
    label: 'Unknown AI automation',
    description: 'AI-shaped signals that need triage before classification.',
    icon: Bot,
  },
}

export function abomCategoryLabel(category: string): string {
  const def = CATEGORY_DEFS[category as AbomCategory]
  return def?.label || category
}

export function abomCategoryDescription(category: string): string {
  const def = CATEGORY_DEFS[category as AbomCategory]
  return def?.description || 'AI-related capability.'
}

function confidenceTone(confidence: string): { dot: string; label: string } {
  if (confidence === 'high') return { dot: 'bg-emerald-500', label: 'High' }
  if (confidence === 'medium') return { dot: 'bg-amber-500', label: 'Medium' }
  return { dot: 'bg-slate-400', label: 'Low' }
}

function formatRelative(ts?: number) {
  if (!ts) return 'no data'
  const seconds = Math.max(0, Math.round((Date.now() - ts) / 1000))
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 48) return `${hours}h ago`
  return `${Math.round(hours / 24)}d ago`
}

type AbomPanelProps = {
  scope: 'fleet' | 'device'
  deviceId?: string
  embedded?: boolean
}

export function AbomPanel({ scope, deviceId, embedded = false }: AbomPanelProps) {
  const [items, setItems] = useState<AbomItem[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [emptyHelp, setEmptyHelp] = useState<string | null>(null)
  const [activeCategory, setActiveCategory] = useState<AbomCategory | 'all'>('all')
  const [confidenceFilter, setConfidenceFilter] = useState<'all' | 'high' | 'medium' | 'low'>('all')
  const [query, setQuery] = useState('')
  const [windowId, setWindowId] = useState<string>('7d')
  const [detail, setDetail] = useState<AbomItem | null>(null)
  const [generatedAt, setGeneratedAt] = useState<number | null>(null)
  const [insights, setInsights] = useState<AbomInsightsResponse | null>(null)
  const [insightsLoading, setInsightsLoading] = useState(false)

  const fetchUrl = useMemo(() => {
    if (scope === 'device' && deviceId) {
      return `/api/visibility/abom/device?device_id=${encodeURIComponent(deviceId)}`
    }
    return '/api/visibility/abom/fleet'
  }, [scope, deviceId])

  const insightsUrl = useMemo(() => {
    if (scope !== 'fleet') return ''
    const ms = TIME_WINDOWS.find((win) => win.id === windowId)?.ms ?? TIME_WINDOWS[2].ms
    return `/api/visibility/abom/insights?since_ms=${ms}`
  }, [scope, windowId])

  const load = useCallback(async () => {
    setRefreshing(true)
    setError(null)
    try {
      const res = await fetch(fetchUrl, { cache: 'no-store' })
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`)
      }
      const body = (await res.json()) as AbomFleetResponse | AbomDeviceResponse
      setItems(Array.isArray(body.items) ? (body.items as AbomItem[]) : [])
      setEmptyHelp(body.empty_help || null)
      setGeneratedAt(body.generated_at_ms || Date.now())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load ABOM')
      setItems([])
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [fetchUrl])

  const loadInsights = useCallback(async () => {
    if (!insightsUrl) {
      setInsights(null)
      return
    }
    setInsightsLoading(true)
    try {
      const res = await fetch(insightsUrl, { cache: 'no-store' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const body = (await res.json()) as AbomInsightsResponse
      setInsights(body)
    } catch {
      setInsights(null)
    } finally {
      setInsightsLoading(false)
    }
  }, [insightsUrl])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    void loadInsights()
  }, [loadInsights])

  const categoryCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const item of items) {
      counts[item.category] = (counts[item.category] || 0) + 1
    }
    return counts
  }, [items])

  const visibleItems = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return items.filter((item) => {
      if (activeCategory !== 'all' && item.category !== activeCategory) return false
      if (confidenceFilter !== 'all' && item.confidence !== confidenceFilter) return false
      if (!needle) return true
      const haystack = [
        item.product,
        item.category,
        item.confidence,
        item.recommended_review,
        item.capability_tags.join(' '),
        item.device_ids.join(' '),
        item.evidence_refs.join(' '),
      ]
        .join(' ')
        .toLowerCase()
      return haystack.includes(needle)
    })
  }, [items, query, activeCategory, confidenceFilter])

  const totalDevices = useMemo(() => {
    const ids = new Set<string>()
    for (const item of items) {
      for (const id of item.device_ids) ids.add(id)
    }
    return ids.size
  }, [items])

  const highConfidence = useMemo(() => items.filter((item) => item.confidence === 'high').length, [items])

  return (
    <div className={embedded ? '' : 'min-h-screen bg-gray-50'}>
      <WorkbenchHeader
        title={scope === 'fleet' ? 'Agent Bill of Materials' : 'Endpoint AI capability'}
        subtitle={
          scope === 'fleet'
            ? 'AI-capable tools, agents, runtimes, and gateways across the fleet, with supporting evidence.'
            : 'AI-capable tools, agents, runtimes, and gateways observed on this endpoint.'
        }
        actions={
          <div className="flex items-center gap-2">
            <span className="text-xs text-slate-500">
              {generatedAt ? `Refreshed ${formatRelative(generatedAt)}` : 'Loading'}
            </span>
            <button
              type="button"
              onClick={() => void load()}
              disabled={refreshing}
              className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
            >
              <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        }
      />

      <SummaryStrip>
        <KpiTile label="ABOM items" value={items.length} />
        <KpiTile label="Categories used" value={Object.keys(categoryCounts).length} />
        <KpiTile label={scope === 'device' ? 'Device' : 'Endpoints w/ AI'} value={scope === 'device' ? deviceId || '—' : totalDevices} />
        <KpiTile label="High confidence" value={highConfidence} />
      </SummaryStrip>

      {scope === 'fleet' ? (
        <FleetInsightsBlock
          insights={insights}
          loading={insightsLoading}
          windowId={windowId}
          onWindowChange={setWindowId}
          onJumpToCategory={(category) => setActiveCategory(category as AbomCategory)}
          onJumpToConfidence={(confidence) => setConfidenceFilter(confidence)}
        />
      ) : null}

      <FilterBar>
        <button
          type="button"
          onClick={() => setActiveCategory('all')}
          className={`rounded-full border px-3 py-1 text-xs font-semibold ${
            activeCategory === 'all'
              ? 'border-blue-600 bg-blue-600 text-white'
              : 'border-slate-200 bg-slate-50 text-slate-700'
          }`}
        >
          All ({items.length})
        </button>
        {ABOM_CATEGORY_ORDER.map((cat) => {
          const count = categoryCounts[cat] || 0
          if (count === 0 && activeCategory !== cat) return null
          return (
            <button
              key={cat}
              type="button"
              onClick={() => setActiveCategory(cat)}
              className={`rounded-full border px-3 py-1 text-xs font-semibold ${
                activeCategory === cat
                  ? 'border-blue-600 bg-blue-600 text-white'
                  : 'border-slate-200 bg-slate-50 text-slate-700'
              }`}
            >
              {abomCategoryLabel(cat)} ({count})
            </button>
          )
        })}
        <div className="ml-auto flex flex-wrap items-center gap-2">
          <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">Confidence</span>
          {(['all', 'high', 'medium', 'low'] as const).map((conf) => (
            <button
              key={conf}
              type="button"
              onClick={() => setConfidenceFilter(conf)}
              className={`rounded-full border px-3 py-1 text-xs font-semibold ${
                confidenceFilter === conf
                  ? 'border-blue-600 bg-blue-600 text-white'
                  : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50'
              }`}
            >
              {conf === 'all' ? 'All' : conf}
            </button>
          ))}
          <label className="relative min-w-[220px] flex-1 max-w-md">
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              className="h-9 w-full rounded-md border border-slate-200 bg-white pl-9 pr-3 text-sm outline-none focus:border-blue-500"
              placeholder="Search ABOM (product, evidence, device, capability)…"
            />
          </label>
        </div>
      </FilterBar>

      {error ? (
        <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          ABOM request failed: {error}
        </div>
      ) : null}

      <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        {loading ? (
          <EmptyState title="Loading ABOM" message="Aggregating AI capability evidence from process, browser, DNS, SASE, and finding telemetry." />
        ) : visibleItems.length === 0 ? (
          <EmptyState
            title={items.length === 0 ? 'No AI capability observed yet' : 'No items match your filter'}
            message={
              items.length === 0
                ? emptyHelp ||
                  'ABOM populates as agents report process, browser extension, SASE, DNS, and finding telemetry.'
                : 'Try a broader filter or category.'
            }
            hint={
              items.length === 0
                ? 'In the lab/demo environment, the sample scenarios walk through the data needed to populate ABOM.'
                : undefined
            }
            actions={
              items.length === 0 ? (
                <>
                  <a
                    href="/demo/scenarios"
                    className="inline-flex h-8 items-center gap-1 rounded-md border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                  >
                    View sample scenarios
                  </a>
                  <a
                    href="/agents"
                    className="inline-flex h-8 items-center gap-1 rounded-md border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                  >
                    Check agents
                  </a>
                </>
              ) : null
            }
          />
        ) : (
          <BoundedTable
            headers={['Category', 'Product', 'Capabilities', 'Confidence', scope === 'device' ? 'First seen' : 'Devices', 'Last seen', 'Action']}
            rows={visibleItems.map((item) => {
              const tone = confidenceTone(item.confidence)
              return [
                <span key={`${item.id}-cat`} className="inline-flex items-center gap-2 text-xs font-semibold text-slate-700">
                  <CategoryIcon category={item.category} />
                  {abomCategoryLabel(item.category)}
                </span>,
                <span key={`${item.id}-product`} className="text-sm font-semibold text-slate-900" title={item.recommended_review}>
                  {item.product}
                </span>,
                <span key={`${item.id}-tags`} className="flex flex-wrap gap-1">
                  {item.capability_tags.length === 0 ? (
                    <span className="text-xs text-slate-400">n/a</span>
                  ) : (
                    item.capability_tags.slice(0, 4).map((tag) => (
                      <span key={tag} className="rounded bg-slate-100 px-1.5 py-0.5 text-[11px] font-medium text-slate-600">
                        {tag}
                      </span>
                    ))
                  )}
                </span>,
                <span key={`${item.id}-conf`} className="inline-flex items-center gap-2 text-xs text-slate-700">
                  <span className={`h-2 w-2 rounded-full ${tone.dot}`} />
                  {tone.label}
                </span>,
                scope === 'device' ? (
                  <span key={`${item.id}-first`} className="text-xs text-slate-600">
                    {formatRelative(item.first_seen_ms)}
                  </span>
                ) : (
                  <DeviceIdLinks key={`${item.id}-devs`} deviceIds={item.device_ids} />
                ),
                <span key={`${item.id}-last`} className="text-xs text-slate-600">
                  {formatRelative(item.last_seen_ms)}
                </span>,
                <div key={`${item.id}-action`} className="flex flex-col items-start gap-1 text-xs font-semibold">
                  <button
                    type="button"
                    onClick={() => setDetail(item)}
                    className="text-blue-700 hover:text-blue-900"
                  >
                    View detail
                  </button>
                  {item.device_ids.length > 0 ? (
                    <a
                      href={controlDesignerHref(item)}
                      className="inline-flex items-center gap-1 text-emerald-700 hover:text-emerald-900"
                    >
                      Design control
                      <ArrowRight className="h-3 w-3" />
                    </a>
                  ) : null}
                </div>,
              ]
            })}
          />
        )}
      </section>

      <DetailModal
        open={Boolean(detail)}
        title={detail ? `${abomCategoryLabel(detail.category)}: ${detail.product}` : 'ABOM detail'}
        detail={
          detail
            ? {
                category: abomCategoryLabel(detail.category),
                category_id: detail.category,
                product: detail.product,
                confidence: detail.confidence,
                capabilities: detail.capability_tags,
                devices: detail.device_ids,
                evidence_refs: detail.evidence_refs,
                first_seen: detail.first_seen_ms ? new Date(detail.first_seen_ms).toISOString() : 'n/a',
                last_seen: detail.last_seen_ms ? new Date(detail.last_seen_ms).toISOString() : 'n/a',
                recommended_review: detail.recommended_review,
                description: abomCategoryDescription(detail.category),
              }
            : {}
        }
        onClose={() => setDetail(null)}
      />
    </div>
  )
}

function controlDesignerHref(item: AbomItem): string {
  const device = item.device_ids[0]
  const params = new URLSearchParams()
  if (device) params.set('device_id', device)
  // The designer accepts a finding id; fall back to the ABOM item id which will
  // surface the row reason in the audit trail when no finding id is present.
  params.set('finding_id', item.id)
  return `/control/controls?${params.toString()}`
}

function FleetInsightsBlock({
  insights,
  loading,
  windowId,
  onWindowChange,
  onJumpToCategory,
  onJumpToConfidence,
}: {
  insights: AbomInsightsResponse | null
  loading: boolean
  windowId: string
  onWindowChange: (id: string) => void
  onJumpToCategory: (category: string) => void
  onJumpToConfidence: (confidence: 'all' | 'high' | 'medium' | 'low') => void
}) {
  const sections = insights?.sections || []
  const hotspots = insights?.hotspots || []
  const totalSignals = sections.reduce((acc, section) => acc + section.total, 0)

  return (
    <section className="mb-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">Fleet insights</h3>
          <p className="text-xs text-slate-500">
            What changed in the AI footprint since the chosen window. Confidence and stale callouts surface inventory rows
            that deserve manual review.
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs">
          <span className="font-semibold uppercase tracking-wide text-slate-500">Window</span>
          <select
            value={windowId}
            onChange={(event) => onWindowChange(event.target.value)}
            className="h-8 rounded-md border border-slate-200 bg-white px-2 text-xs text-slate-700"
          >
            {TIME_WINDOWS.map((win) => (
              <option key={win.id} value={win.id}>
                {win.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {loading && !insights ? (
        <p className="mt-4 text-xs text-slate-500">Aggregating insights…</p>
      ) : null}

      {insights && totalSignals === 0 ? (
        <p className="mt-4 rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-600">
          No insight signals in this window. Try a longer window, or verify that agents are reporting fresh telemetry.
        </p>
      ) : null}

      <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {sections.map((section) => (
          <InsightCard
            key={section.id}
            section={section}
            onJumpToCategory={onJumpToCategory}
            onJumpToConfidence={onJumpToConfidence}
          />
        ))}
      </div>

      {hotspots.length > 0 ? (
        <div className="mt-4">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Endpoint hotspots</h4>
          <ul className="mt-2 grid gap-2 md:grid-cols-2">
            {hotspots.slice(0, 6).map((hotspot) => (
              <li
                key={hotspot.device_id}
                className={`rounded-md border px-3 py-2 text-xs ${
                  hotspot.high_attention ? 'border-amber-200 bg-amber-50 text-amber-900' : 'border-slate-200 bg-white text-slate-700'
                }`}
              >
                <div className="flex items-center justify-between gap-2">
                  <a
                    href={`/agents/${encodeURIComponent(hotspot.device_id)}`}
                    className="font-semibold text-blue-700 hover:text-blue-900"
                  >
                    {hotspot.device_id}
                  </a>
                  <span className="text-[11px] uppercase tracking-wide text-slate-500">
                    {hotspot.item_count} ABOM rows
                  </span>
                </div>
                <p className="mt-1 leading-relaxed">{hotspot.reason}</p>
                {hotspot.categories.length > 0 ? (
                  <p className="mt-1 text-[11px] text-slate-500">
                    Categories: {hotspot.categories.map((cat) => abomCategoryLabel(cat)).join(', ')}
                  </p>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  )
}

function InsightCard({
  section,
  onJumpToCategory,
  onJumpToConfidence,
}: {
  section: AbomInsightSection
  onJumpToCategory: (category: string) => void
  onJumpToConfidence: (confidence: 'all' | 'high' | 'medium' | 'low') => void
}) {
  const Icon = INSIGHT_ICONS[section.id] || Sparkles
  const tone =
    section.id === 'newly_observed_high_attention'
      ? 'border-amber-200 bg-amber-50'
      : section.id === 'low_confidence_needs_review'
        ? 'border-rose-200 bg-rose-50'
        : section.id === 'high_confidence'
          ? 'border-emerald-200 bg-emerald-50'
          : 'border-slate-200 bg-white'
  return (
    <article className={`rounded-md border p-3 shadow-sm ${tone}`}>
      <div className="flex items-start gap-2">
        <span className="mt-0.5 inline-flex h-7 w-7 items-center justify-center rounded-md bg-white/70 text-slate-700">
          <Icon className="h-4 w-4" />
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <h4 className="text-sm font-semibold text-slate-900">{section.title}</h4>
            <span className="text-xs font-semibold text-slate-700">{section.total}</span>
          </div>
          <p className="mt-0.5 text-xs leading-relaxed text-slate-700">{section.detail}</p>
          {section.threshold ? (
            <p className="mt-1 font-mono text-[11px] text-slate-500">{section.threshold}</p>
          ) : null}
        </div>
      </div>

      {section.items.length === 0 ? (
        <p className="mt-2 text-xs italic text-slate-500">No rows in this slice yet.</p>
      ) : (
        <ul className="mt-2 space-y-1.5 text-xs">
          {section.items.slice(0, 4).map((item) => (
            <li key={item.id} className="rounded bg-white/60 px-2 py-1.5">
              <div className="flex items-center justify-between gap-2">
                <button
                  type="button"
                  onClick={() => onJumpToCategory(item.category)}
                  className="truncate text-left font-semibold text-slate-900 hover:text-blue-700"
                  title={item.product}
                >
                  {item.product}
                </button>
                <span className="shrink-0 rounded-full border border-slate-200 bg-white px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-slate-600">
                  {item.confidence}
                </span>
              </div>
              <p className="text-[11px] text-slate-600">{item.reason}</p>
              {item.high_attention ? (
                <p className="text-[11px] font-semibold text-amber-700">High-attention endpoint</p>
              ) : null}
            </li>
          ))}
        </ul>
      )}

      {section.id === 'high_confidence' || section.id === 'low_confidence_needs_review' ? (
        <button
          type="button"
          onClick={() =>
            onJumpToConfidence(section.id === 'high_confidence' ? 'high' : 'low')
          }
          className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-blue-700 hover:text-blue-900"
        >
          Filter table to {section.id === 'high_confidence' ? 'high' : 'low'} confidence
          <ArrowRight className="h-3 w-3" />
        </button>
      ) : null}
    </article>
  )
}

function CategoryIcon({ category }: { category: string }) {
  const def = CATEGORY_DEFS[category as AbomCategory]
  const Icon = def?.icon || Bot
  return <Icon className="h-4 w-4 text-slate-500" />
}

function DeviceIdLinks({ deviceIds }: { deviceIds: string[] }) {
  if (deviceIds.length === 0) return <span className="text-xs text-slate-400">n/a</span>
  const shown = deviceIds.slice(0, 3)
  const rest = deviceIds.length - shown.length
  return (
    <div className="flex flex-wrap gap-1.5">
      {shown.map((id) => (
        <a
          key={id}
          href={`/agents/${encodeURIComponent(id)}`}
          className="inline-flex max-w-[180px] truncate rounded border border-blue-100 bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-800 hover:bg-blue-100"
          title={id}
        >
          {id}
        </a>
      ))}
      {rest > 0 ? <span className="self-center text-xs text-slate-500">+{rest} more</span> : null}
    </div>
  )
}

export type { AbomPanelProps }
