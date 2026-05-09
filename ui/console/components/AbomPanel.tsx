'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Bot, Chrome, Code2, Cpu, Globe2, RefreshCw, Server, Sparkles, Terminal, Wrench, Search } from 'lucide-react'
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
  const [query, setQuery] = useState('')
  const [detail, setDetail] = useState<AbomItem | null>(null)
  const [generatedAt, setGeneratedAt] = useState<number | null>(null)

  const fetchUrl = useMemo(() => {
    if (scope === 'device' && deviceId) {
      return `/api/visibility/abom/device?device_id=${encodeURIComponent(deviceId)}`
    }
    return '/api/visibility/abom/fleet'
  }, [scope, deviceId])

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

  useEffect(() => {
    void load()
  }, [load])

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
  }, [items, query, activeCategory])

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
        <label className="relative ml-auto min-w-[220px] flex-1 max-w-md">
          <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className="h-9 w-full rounded-md border border-slate-200 bg-white pl-9 pr-3 text-sm outline-none focus:border-blue-500"
            placeholder="Search ABOM (product, evidence, device, capability)…"
          />
        </label>
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
                <button
                  key={`${item.id}-action`}
                  type="button"
                  onClick={() => setDetail(item)}
                  className="text-xs font-semibold text-blue-700 hover:text-blue-900"
                >
                  View detail
                </button>,
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
