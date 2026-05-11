'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  AlertTriangle,
  ArrowRight,
  Bot,
  Chrome,
  ChevronDown,
  ChevronUp,
  Clock,
  Code2,
  Cpu,
  Flame,
  Globe2,
  RefreshCw,
  Search,
  ShieldQuestion,
  Sparkles,
  Terminal,
  TrendingUp,
  Wrench,
} from 'lucide-react'
import { DetailModal, EmptyState } from '@/components/workbench/primitives'

/* ── Types ──────────────────────────────────────────────────────────────── */

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

/* ── Constants ──────────────────────────────────────────────────────────── */

const TIME_WINDOWS: { id: string; label: string; ms: number }[] = [
  { id: '24h', label: 'Last 24h',    ms: 24 * 60 * 60 * 1000 },
  { id: '3d',  label: 'Last 3 days', ms: 3 * 24 * 60 * 60 * 1000 },
  { id: '7d',  label: 'Last 7 days', ms: 7 * 24 * 60 * 60 * 1000 },
  { id: '30d', label: 'Last 30 days',ms: 30 * 24 * 60 * 60 * 1000 },
]

const INSIGHT_ICONS: Record<string, typeof TrendingUp> = {
  newly_observed:                   TrendingUp,
  newly_observed_high_attention:    AlertTriangle,
  high_confidence:                  Sparkles,
  low_confidence_needs_review:      ShieldQuestion,
  widespread:                       Flame,
  stale:                            Clock,
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

const CATEGORY_DEFS: Record<AbomCategory, { label: string; description: string; icon: typeof Bot }> = {
  ai_desktop_app:        { label: 'AI desktop apps',       description: 'Vendor desktop clients such as ChatGPT, Claude, Copilot, or Gemini.', icon: Sparkles },
  browser_ai_extension:  { label: 'Browser AI extensions', description: 'Browser extensions with AI naming or session activity.',              icon: Chrome },
  coding_agent:          { label: 'Coding agents',         description: 'IDE-class agents like Cursor, Continue, GitHub Copilot, and Tabnine.', icon: Code2 },
  cli_agent:             { label: 'CLI agents',            description: 'Shell-driven agents such as codex, claude-code, aider, gemini-cli.',   icon: Terminal },
  mcp_endpoint:          { label: 'MCP endpoints',         description: 'Model Context Protocol servers and clients.',                          icon: Wrench },
  local_model_runtime:   { label: 'Local model runtimes',  description: 'Ollama, vLLM, llama.cpp, LM Studio, and similar runtimes.',          icon: Cpu },
  model_gateway:         { label: 'Model gateways',        description: 'External APIs and enterprise gateways acting as model providers.',    icon: Globe2 },
  unknown_ai_automation: { label: 'Unknown AI automation', description: 'AI-shaped signals that need triage before classification.',           icon: Bot },
}

export function abomCategoryLabel(category: string): string {
  return CATEGORY_DEFS[category as AbomCategory]?.label || category
}

export function abomCategoryDescription(category: string): string {
  return CATEGORY_DEFS[category as AbomCategory]?.description || 'AI-related capability.'
}

const QUICK_TABS = [
  { id: 'all', label: 'All' },
  ...ABOM_CATEGORY_ORDER.map((cat) => ({ id: cat, label: CATEGORY_DEFS[cat].label })),
] as const

type TabId = 'all' | AbomCategory

/* ── Helpers ────────────────────────────────────────────────────────────── */

function confidenceBadge(confidence: string) {
  if (confidence === 'high')   return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800">High</span>
  if (confidence === 'medium') return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-800">Medium</span>
  return                              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-gray-100 text-gray-600">Low</span>
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

function controlDesignerHref(item: AbomItem): string {
  const params = new URLSearchParams()
  if (item.device_ids[0]) params.set('device_id', item.device_ids[0])
  params.set('finding_id', item.id)
  return `/control/controls?${params.toString()}`
}

/* ── CategoryIcon ───────────────────────────────────────────────────────── */

function CategoryIcon({ category, className = 'h-4 w-4 text-gray-500' }: { category: string; className?: string }) {
  const Icon = CATEGORY_DEFS[category as AbomCategory]?.icon || Bot
  return <Icon className={className} />
}

/* ── DeviceIdLinks ──────────────────────────────────────────────────────── */

function DeviceIdLinks({ deviceIds }: { deviceIds: string[] }) {
  if (deviceIds.length === 0) return <span className="text-xs text-gray-400">n/a</span>
  const shown = deviceIds.slice(0, 2)
  const rest  = deviceIds.length - shown.length
  return (
    <div className="flex flex-wrap gap-1">
      {shown.map((id) => (
        <a
          key={id}
          href={`/agents/${encodeURIComponent(id)}`}
          className="inline-flex max-w-[160px] truncate rounded border border-blue-100 bg-blue-50 px-1.5 py-0.5 text-xs font-medium text-blue-800 hover:bg-blue-100 transition-colors"
          title={id}
        >
          {id}
        </a>
      ))}
      {rest > 0 && <span className="self-center text-xs text-gray-500">+{rest}</span>}
    </div>
  )
}

/* ── Insight panel (collapsed by default on fleet view) ─────────────────── */

function FleetInsightsPanel({
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
  const [open, setOpen] = useState(false)
  const sections  = insights?.sections  || []
  const hotspots  = insights?.hotspots  || []
  const hasSignals = sections.some((s) => s.total > 0)

  return (
    <div className="bg-white rounded-lg shadow mb-6 overflow-hidden">
      {/* Collapsible header */}
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between px-5 py-3.5 text-left hover:bg-gray-50 transition-colors"
      >
        <div className="flex items-center gap-3">
          <Sparkles className="h-4 w-4 text-blue-500" />
          <div>
            <span className="text-sm font-semibold text-gray-900">Fleet insights</span>
            <span className="ml-2 text-xs text-gray-400">
              {loading ? 'Loading…' : hasSignals ? `${sections.filter((s) => s.total > 0).length} active signals` : 'No signals in window'}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <select
            value={windowId}
            onChange={(e) => { e.stopPropagation(); onWindowChange(e.target.value) }}
            onClick={(e) => e.stopPropagation()}
            className="text-xs border border-gray-200 rounded-md px-2 py-1 text-gray-600 bg-white focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent"
          >
            {TIME_WINDOWS.map((w) => (
              <option key={w.id} value={w.id}>{w.label}</option>
            ))}
          </select>
          {open ? <ChevronUp className="h-4 w-4 text-gray-400" /> : <ChevronDown className="h-4 w-4 text-gray-400" />}
        </div>
      </button>

      {/* Expanded content */}
      {open && (
        <div className="border-t border-gray-100 px-5 py-4">
          {loading && !insights ? (
            <p className="text-xs text-gray-500">Aggregating insights…</p>
          ) : !hasSignals ? (
            <p className="text-xs text-gray-500 italic">No signals in this window. Try a longer window or verify agent telemetry.</p>
          ) : (
            <>
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {sections.filter((s) => s.total > 0).map((section) => {
                  const Icon = INSIGHT_ICONS[section.id] || Sparkles
                  const tone =
                    section.id === 'newly_observed_high_attention' ? 'border-amber-200 bg-amber-50'
                    : section.id === 'low_confidence_needs_review'  ? 'border-rose-100 bg-rose-50'
                    : section.id === 'high_confidence'               ? 'border-emerald-200 bg-emerald-50'
                    : 'border-gray-200 bg-white'
                  return (
                    <div key={section.id} className={`rounded-lg border p-3 ${tone}`}>
                      <div className="flex items-start gap-2 mb-2">
                        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-white/70">
                          <Icon className="h-4 w-4 text-gray-600" />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center justify-between gap-1">
                            <span className="text-sm font-semibold text-gray-900 truncate">{section.title}</span>
                            <span className="shrink-0 text-xs font-semibold text-gray-600">{section.total}</span>
                          </div>
                          <p className="text-xs text-gray-500 mt-0.5">{section.detail}</p>
                        </div>
                      </div>
                      {section.items.length > 0 && (
                        <ul className="space-y-1">
                          {section.items.slice(0, 3).map((item) => (
                            <li key={item.id} className="rounded bg-white/60 px-2 py-1">
                              <div className="flex items-center justify-between gap-1">
                                <button
                                  type="button"
                                  onClick={() => onJumpToCategory(item.category)}
                                  className="text-xs font-semibold text-gray-900 hover:text-[#1e3a5f] truncate text-left"
                                >
                                  {item.product}
                                </button>
                                <span className="shrink-0 text-[10px] uppercase tracking-wide text-gray-500">{item.confidence}</span>
                              </div>
                            </li>
                          ))}
                        </ul>
                      )}
                      {(section.id === 'high_confidence' || section.id === 'low_confidence_needs_review') && (
                        <button
                          type="button"
                          onClick={() => onJumpToConfidence(section.id === 'high_confidence' ? 'high' : 'low')}
                          className="mt-2 text-xs font-semibold text-[#1e3a5f] hover:underline flex items-center gap-1"
                        >
                          Filter table
                          <ArrowRight className="h-3 w-3" />
                        </button>
                      )}
                    </div>
                  )
                })}
              </div>

              {hotspots.length > 0 && (
                <div className="mt-4">
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-gray-400 mb-2">Endpoint hotspots</h4>
                  <div className="grid gap-2 md:grid-cols-2">
                    {hotspots.slice(0, 4).map((h) => (
                      <div
                        key={h.device_id}
                        className={`rounded-lg border px-3 py-2 text-xs ${h.high_attention ? 'border-amber-200 bg-amber-50 text-amber-900' : 'border-gray-200 bg-white text-gray-700'}`}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <a href={`/agents/${encodeURIComponent(h.device_id)}`} className="font-semibold text-[#1e3a5f] hover:underline truncate">
                            {h.device_id}
                          </a>
                          <span className="shrink-0 text-[10px] uppercase tracking-wide text-gray-500">{h.item_count} items</span>
                        </div>
                        <p className="mt-0.5 leading-snug text-gray-500">{h.reason}</p>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}

/* ── Main panel ─────────────────────────────────────────────────────────── */

export type AbomPanelProps = {
  scope: 'fleet' | 'device'
  deviceId?: string
  embedded?: boolean
}

export function AbomPanel({ scope, deviceId, embedded = false }: AbomPanelProps) {
  const [items, setItems]               = useState<AbomItem[]>([])
  const [loading, setLoading]           = useState(true)
  const [refreshing, setRefreshing]     = useState(false)
  const [error, setError]               = useState<string | null>(null)
  const [emptyHelp, setEmptyHelp]       = useState<string | null>(null)
  const [activeTab, setActiveTab]       = useState<TabId>('all')
  const [confidenceFilter, setConfidenceFilter] = useState<'all' | 'high' | 'medium' | 'low'>('all')
  const [query, setQuery]               = useState('')
  const [windowId, setWindowId]         = useState('7d')
  const [detail, setDetail]             = useState<AbomItem | null>(null)
  const [generatedAt, setGeneratedAt]   = useState<number | null>(null)
  const [insights, setInsights]         = useState<AbomInsightsResponse | null>(null)
  const [insightsLoading, setInsightsLoading] = useState(false)

  const fetchUrl = useMemo(() => {
    if (scope === 'device' && deviceId) {
      return `/api/visibility/abom/device?device_id=${encodeURIComponent(deviceId)}`
    }
    return '/api/visibility/abom/fleet'
  }, [scope, deviceId])

  const insightsUrl = useMemo(() => {
    if (scope !== 'fleet') return ''
    const ms = TIME_WINDOWS.find((w) => w.id === windowId)?.ms ?? TIME_WINDOWS[2].ms
    return `/api/visibility/abom/insights?since_ms=${ms}`
  }, [scope, windowId])

  const load = useCallback(async () => {
    setRefreshing(true)
    setError(null)
    try {
      const res  = await fetch(fetchUrl, { cache: 'no-store' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
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
    if (!insightsUrl) { setInsights(null); return }
    setInsightsLoading(true)
    try {
      const res  = await fetch(insightsUrl, { cache: 'no-store' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      setInsights((await res.json()) as AbomInsightsResponse)
    } catch {
      setInsights(null)
    } finally {
      setInsightsLoading(false)
    }
  }, [insightsUrl])

  useEffect(() => { void load() }, [load])
  useEffect(() => { void loadInsights() }, [loadInsights])

  /* ── Derived counts ────────────────────────────────────────────────── */
  const categoryCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const item of items) counts[item.category] = (counts[item.category] || 0) + 1
    return counts
  }, [items])

  const totalDevices = useMemo(() => {
    const ids = new Set<string>()
    for (const item of items) for (const id of item.device_ids) ids.add(id)
    return ids.size
  }, [items])

  const highConfidence = useMemo(() => items.filter((i) => i.confidence === 'high').length, [items])

  const tabCounts = useMemo(() => {
    const result: Record<string, number> = { all: items.length }
    for (const cat of ABOM_CATEGORY_ORDER) result[cat] = categoryCounts[cat] || 0
    return result
  }, [items, categoryCounts])

  /* ── Filtered rows ─────────────────────────────────────────────────── */
  const visibleItems = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return items.filter((item) => {
      if (activeTab !== 'all' && item.category !== activeTab) return false
      if (confidenceFilter !== 'all' && item.confidence !== confidenceFilter) return false
      if (!needle) return true
      return [item.product, item.category, item.capability_tags.join(' '), item.device_ids.join(' ')].join(' ').toLowerCase().includes(needle)
    })
  }, [items, activeTab, confidenceFilter, query])

  /* ── Summary values ────────────────────────────────────────────────── */
  const activeCategories = ABOM_CATEGORY_ORDER.filter((c) => (categoryCounts[c] || 0) > 0).length
  const currentFocusLabel =
    activeTab === 'all' ? 'Full AI capability inventory'
    : abomCategoryLabel(activeTab)
  const nextBestAction =
    items.length === 0
      ? 'No AI capability data yet. Ensure agents are reporting process, browser, SASE, and DNS telemetry.'
      : highConfidence > 0
        ? `${highConfidence} high-confidence item${highConfidence > 1 ? 's' : ''} are ready for control design.`
        : 'Review medium and low confidence items to improve classification accuracy.'

  return (
    <div className={embedded ? '' : 'min-h-screen bg-gray-50'}>
      {/* Page header — only shown in standalone (non-embedded) view */}
      {!embedded && (
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              {scope === 'fleet' ? 'Agent Bill of Materials' : 'Endpoint AI capability'}
            </h1>
            <p className="text-sm text-gray-500 mt-0.5">
              {scope === 'fleet'
                ? 'AI-capable tools, agents, runtimes, and gateways observed across the fleet.'
                : 'AI-capable tools and runtimes observed on this endpoint.'}
            </p>
          </div>
          <div className="flex items-center gap-3">
            {generatedAt && (
              <span className="text-xs text-gray-400">Updated {formatRelative(generatedAt)}</span>
            )}
            <button
              type="button"
              onClick={() => void load()}
              disabled={refreshing}
              className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-[#1e3a5f] rounded-md hover:bg-[#162d4a] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="mb-4 flex items-center gap-3 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          ABOM request failed: {error}
        </div>
      )}

      {/* Summary stat cards — fleet: 4 cards; embedded device: compact 3-card strip */}
      {embedded ? (
        <div className="mb-5 flex flex-wrap gap-4 border-b border-gray-100 pb-5">
          <div className="flex items-center gap-3 pr-4 border-r border-gray-200">
            <span className="text-2xl font-bold text-gray-900">{items.length}</span>
            <span className="text-sm text-gray-500">AI items observed</span>
          </div>
          <div className="flex items-center gap-3 pr-4 border-r border-gray-200">
            <span className="text-2xl font-bold text-gray-900">{activeCategories}</span>
            <span className="text-sm text-gray-500">categor{activeCategories !== 1 ? 'ies' : 'y'}</span>
          </div>
          <div className="flex items-center gap-3 pr-4 border-r border-gray-200">
            <span className={`text-2xl font-bold ${highConfidence > 0 ? 'text-emerald-700' : 'text-gray-400'}`}>{highConfidence}</span>
            <span className="text-sm text-gray-500">high confidence</span>
          </div>
          {generatedAt && (
            <div className="flex items-center gap-2 ml-auto">
              <span className="text-xs text-gray-400">Updated {formatRelative(generatedAt)}</span>
              <button
                type="button"
                onClick={() => void load()}
                disabled={refreshing}
                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 disabled:opacity-50 transition-colors"
              >
                <RefreshCw className={`h-3 w-3 ${refreshing ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            </div>
          )}
        </div>
      ) : (
        <div className="mb-6 grid gap-4 lg:grid-cols-4">
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">ABOM items</p>
            <p className="mt-2 text-2xl font-semibold text-slate-900">{items.length}</p>
            <p className="mt-2 text-sm text-slate-600">{activeCategories} active categor{activeCategories !== 1 ? 'ies' : 'y'}</p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
              {scope === 'device' ? 'Device' : 'Endpoints w/ AI'}
            </p>
            <p className="mt-2 text-2xl font-semibold text-slate-900">
              {scope === 'device' ? (deviceId || '—') : totalDevices}
            </p>
            <p className="mt-2 text-sm text-slate-600">
              {scope === 'device' ? 'endpoint under review' : 'endpoints observed with AI tooling'}
            </p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Current focus</p>
            <p className="mt-2 text-lg font-semibold text-slate-900 leading-tight">{currentFocusLabel}</p>
            <p className="mt-2 text-sm text-slate-600">Use the category tabs to filter by tool type.</p>
          </div>
          <div className="rounded-xl border border-amber-200 bg-amber-50 p-5 shadow-sm">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-amber-700">Next best action</p>
            <p className="mt-2 text-sm font-medium text-amber-900">{nextBestAction}</p>
            {highConfidence > 0 && (
              <p className="mt-2 text-sm text-amber-800">{highConfidence} item{highConfidence > 1 ? 's' : ''} ready for control design.</p>
            )}
          </div>
        </div>
      )}

      {/* Fleet insights (collapsible) */}
      {scope === 'fleet' && (
        <FleetInsightsPanel
          insights={insights}
          loading={insightsLoading}
          windowId={windowId}
          onWindowChange={setWindowId}
          onJumpToCategory={(cat) => setActiveTab(cat as AbomCategory)}
          onJumpToConfidence={setConfidenceFilter}
        />
      )}

      {/* Category tab bar (underline style like Clarion) */}
      <div className="flex flex-wrap items-end border-b border-gray-200 mb-4 overflow-x-auto">
        {QUICK_TABS.map((tab) => {
          const count = tabCounts[tab.id] ?? 0
          if (tab.id !== 'all' && count === 0) return null
          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id as TabId)}
              className={`px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap ${
                activeTab === tab.id
                  ? 'border-[#1e3a5f] text-[#1e3a5f]'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              {tab.label}
              <span className={`ml-1.5 text-xs px-1.5 py-0.5 rounded-full ${
                activeTab === tab.id ? 'bg-blue-100 text-blue-700' : 'bg-gray-100 text-gray-500'
              }`}>
                {count}
              </span>
            </button>
          )
        })}
      </div>

      {/* Filter bar */}
      <div className="bg-white rounded-lg shadow p-4 mb-6">
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search product, evidence, device…"
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent"
            />
          </div>
          <div className="flex items-center gap-2">
            <label className="text-xs font-medium text-gray-500">Confidence</label>
            <select
              value={confidenceFilter}
              onChange={(e) => setConfidenceFilter(e.target.value as typeof confidenceFilter)}
              className="text-sm border border-gray-300 rounded-md px-3 py-2 focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent"
            >
              <option value="all">All</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>
          </div>
        </div>
      </div>

      {/* Main table */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500 text-sm">Loading ABOM…</div>
        ) : items.length === 0 ? (
          <EmptyState
            title="No AI capability observed yet"
            message={emptyHelp || 'ABOM populates as agents report process, browser extension, SASE, DNS, and finding telemetry.'}
            hint="In the lab environment, the sample scenarios walk through the data needed to populate ABOM."
            actions={
              <>
                <a href="/demo/scenarios" className="inline-flex h-8 items-center gap-1 rounded-md border border-gray-200 bg-white px-3 text-xs font-semibold text-gray-700 hover:bg-gray-50">
                  View sample scenarios
                </a>
                <a href="/agents" className="inline-flex h-8 items-center gap-1 rounded-md border border-gray-200 bg-white px-3 text-xs font-semibold text-gray-700 hover:bg-gray-50">
                  Check agents
                </a>
              </>
            }
          />
        ) : visibleItems.length === 0 ? (
          <div className="p-10 text-center">
            <p className="text-gray-500 text-sm mb-3">No items match your current filters.</p>
            <button
              type="button"
              onClick={() => { setActiveTab('all'); setConfidenceFilter('all'); setQuery('') }}
              className="text-sm text-[#1e3a5f] hover:underline"
            >
              Clear all filters
            </button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Category</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Product</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Capabilities</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Confidence</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {scope === 'device' ? 'First seen' : 'Endpoints'}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Last seen</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {visibleItems.map((item) => (
                  <tr key={item.id} className="hover:bg-gray-50 transition-colors">
                    {/* Category */}
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center gap-2">
                        <CategoryIcon category={item.category} className="h-4 w-4 text-gray-400 shrink-0" />
                        <span className="text-sm font-medium text-gray-900">{abomCategoryLabel(item.category)}</span>
                      </div>
                    </td>

                    {/* Product */}
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-semibold text-gray-900">{item.product}</div>
                      {item.recommended_review && (
                        <div className="text-xs text-gray-400 mt-0.5 max-w-[200px] truncate" title={item.recommended_review}>
                          {item.recommended_review}
                        </div>
                      )}
                    </td>

                    {/* Capabilities */}
                    <td className="px-6 py-4">
                      {item.capability_tags.length === 0 ? (
                        <span className="text-xs text-gray-400">—</span>
                      ) : (
                        <div className="flex flex-wrap gap-1">
                          {item.capability_tags.slice(0, 3).map((tag) => (
                            <span key={tag} className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
                              {tag}
                            </span>
                          ))}
                          {item.capability_tags.length > 3 && (
                            <span className="text-xs text-gray-400">+{item.capability_tags.length - 3}</span>
                          )}
                        </div>
                      )}
                    </td>

                    {/* Confidence */}
                    <td className="px-6 py-4 whitespace-nowrap">
                      {confidenceBadge(item.confidence)}
                    </td>

                    {/* Endpoints / First seen */}
                    <td className="px-6 py-4 whitespace-nowrap">
                      {scope === 'device' ? (
                        <span className="text-sm text-gray-600">{formatRelative(item.first_seen_ms)}</span>
                      ) : (
                        <DeviceIdLinks deviceIds={item.device_ids} />
                      )}
                    </td>

                    {/* Last seen */}
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="text-sm text-gray-600">{formatRelative(item.last_seen_ms)}</span>
                    </td>

                    {/* Actions */}
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex flex-col items-start gap-1.5">
                        <button
                          type="button"
                          onClick={() => setDetail(item)}
                          className="text-xs font-semibold text-[#1e3a5f] hover:underline"
                        >
                          View detail
                        </button>
                        {item.device_ids.length > 0 && (
                          <a
                            href={controlDesignerHref(item)}
                            className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-700 hover:text-emerald-900"
                          >
                            Design control
                            <ArrowRight className="h-3 w-3" />
                          </a>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Detail modal */}
      <DetailModal
        open={Boolean(detail)}
        title={detail ? `${abomCategoryLabel(detail.category)}: ${detail.product}` : 'ABOM detail'}
        detail={
          detail ? {
            category:           abomCategoryLabel(detail.category),
            category_id:        detail.category,
            product:            detail.product,
            confidence:         detail.confidence,
            capabilities:       detail.capability_tags,
            devices:            detail.device_ids,
            evidence_refs:      detail.evidence_refs,
            first_seen:         detail.first_seen_ms ? new Date(detail.first_seen_ms).toISOString() : 'n/a',
            last_seen:          detail.last_seen_ms  ? new Date(detail.last_seen_ms).toISOString()  : 'n/a',
            recommended_review: detail.recommended_review,
            description:        abomCategoryDescription(detail.category),
          } : {}
        }
        onClose={() => setDetail(null)}
      />
    </div>
  )
}
