'use client'

import { useEffect, useMemo, useState } from 'react'
import {
  AlertTriangle,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
  Monitor,
  Terminal,
} from 'lucide-react'
import {
  CopyValueButton,
  DetailModal,
  EmptyState,
  WorkbenchHeader,
} from '@/components/workbench/primitives'
import { formatAgentId, formatDateTime, formatHash, formatHostname, formatRelativeAge } from '@/shared/formatting'

/* ── Types ──────────────────────────────────────────────────────────────── */

interface Agent {
  agent_uid: string
  org_id: string
  host_id: string
  hostname: string
  agent_version: string
  capabilities: {
    ebpf_loading: boolean
    ebpf_attach: boolean
    map_operations: boolean
    kernel_modules: string[]
    supported_hooks: string[]
    max_programs: number
    max_maps: number
  }
  platform: {
    hostname: string
    fqdn: string
    os: string
    kernel_version: string
    architecture: string
    cpu_model: string
    memory_gb: number
    disk_gb: number
    primary_ip: string
  }
  network: {
    primary_ip: string
    mac_address: string
    subnet: string
    gateway: string
    dns_servers: string[]
    ifaces: Record<string, { addrs: string[]; mac: string }>
  }
  labels: string[]
  note: string
  created: string
  last_seen: string
  status: 'online' | 'offline' | 'unknown'
  detection_pack_status?: DetectionPackStatus | null
  visibility?: VisibilitySummary | null
  readiness?: AgentReadiness | null
}

export interface AgentReadiness {
  bucket: 'ready' | 'needs_attention' | 'stale' | 'degraded' | 'unknown' | string
  score: number
  summary?: string
  fix_first?: string
  dimensions?: AgentReadinessDimension[]
}

export interface AgentReadinessDimension {
  id: string
  label: string
  state: 'good' | 'warn' | 'bad' | 'unknown' | string
  value?: string
  detail?: string
  weight?: number
}

export const READINESS_BUCKET_LABEL: Record<string, string> = {
  ready: 'Ready',
  needs_attention: 'Needs attention',
  stale: 'Stale',
  degraded: 'Degraded',
  unknown: 'Unknown',
}

export const READINESS_BUCKET_TONE: Record<string, string> = {
  ready: 'border-emerald-200 bg-emerald-50 text-emerald-900',
  needs_attention: 'border-amber-200 bg-amber-50 text-amber-900',
  stale: 'border-slate-300 bg-slate-100 text-slate-800',
  degraded: 'border-rose-200 bg-rose-50 text-rose-900',
  unknown: 'border-slate-200 bg-slate-50 text-slate-700',
}

interface VisibilitySummary {
  event_count?: number
  event_type_count?: Record<string, number>
  last_seen_ms?: number
  last_event_type?: string
}

interface DetectionPackStatus {
  active_pack_id?: string
  active_pack_version?: string
  previous_pack_id?: string
  previous_pack_version?: string
  rollout_state?: string
  reason_detail?: string
  reason_codes?: string[]
  signature_status?: string
  hash_status?: string
  schema_status?: string
  compatibility_status?: string
  last_check_at_ms?: number
  last_applied_at_ms?: number
  last_rejected_at_ms?: number
  last_rejected_pack_id?: string
  last_rejected_reason?: string
  last_rejected_reason_codes?: string[]
  computed_stale?: boolean
}

/* ── Constants ──────────────────────────────────────────────────────────── */

const ROLLOUT_STALE_MS = 24 * 60 * 60 * 1000

const QUICK_FILTERS = [
  { id: 'all',             label: 'All' },
  { id: 'online',          label: 'Online' },
  { id: 'offline',         label: 'Offline' },
  { id: 'windows',         label: 'Windows' },
  { id: 'linux',           label: 'Linux' },
  { id: 'pack_issue',      label: 'Pack issues' },
  { id: 'needs_attention', label: 'Needs attention' },
] as const

type QuickFilterId = typeof QUICK_FILTERS[number]['id']

/* ── Small helpers ──────────────────────────────────────────────────────── */

/** Freshness dot + relative label, matching Clarion's FreshnessDot pattern. */
function FreshnessDot({ lastSeen }: { lastSeen: string }) {
  const ageMs = Date.now() - new Date(lastSeen).getTime()
  const level =
    ageMs < 2 * 60 * 1000  ? 'fresh'
    : ageMs < 5 * 60 * 1000  ? 'recent'
    : ageMs < 60 * 60 * 1000 ? 'stale'
    : 'old'
  const styles = {
    fresh:  { dot: 'bg-green-500', text: 'text-green-700' },
    recent: { dot: 'bg-yellow-400', text: 'text-yellow-700' },
    stale:  { dot: 'bg-red-400',   text: 'text-red-600' },
    old:    { dot: 'bg-gray-400',  text: 'text-gray-500' },
  }
  const { dot, text } = styles[level]
  return (
    <div className="flex items-center gap-1.5">
      <span className={`h-2 w-2 rounded-full flex-shrink-0 ${dot}`} />
      <span className={`text-xs ${text}`}>{formatRelativeAge(lastSeen)}</span>
    </div>
  )
}

function OSIcon({ os }: { os: string }) {
  const lower = (os || '').toLowerCase()
  if (lower.includes('win')) return <Monitor className="h-4 w-4 text-blue-500" />
  if (lower.includes('linux') || lower.includes('darwin')) return <Terminal className="h-4 w-4 text-green-600" />
  return <Server className="h-4 w-4 text-gray-400" />
}

export function ReadinessBadge({ readiness }: { readiness?: AgentReadiness | null }) {
  if (!readiness) return null
  const tone = READINESS_BUCKET_TONE[readiness.bucket] || READINESS_BUCKET_TONE.unknown
  return (
    <span
      title={readiness.summary || ''}
      className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${tone}`}
    >
      {READINESS_BUCKET_LABEL[readiness.bucket] || readiness.bucket}
      <span className="font-mono normal-case tracking-normal text-[10px] opacity-70">{readiness.score}</span>
    </span>
  )
}

/* ── Main component ─────────────────────────────────────────────────────── */

export function AgentsManagementPanel({ embedded = false }: { embedded?: boolean }) {
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [quickFilter, setQuickFilter] = useState<QuickFilterId>('all')
  const [query, setQuery] = useState('')
  const [detailModal, setDetailModal] = useState<{ title: string; payload: unknown } | null>(null)

  useEffect(() => {
    fetchAgents()
    const interval = setInterval(() => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return
      fetchAgents()
    }, 60000)
    return () => clearInterval(interval)
  }, [])

  const fetchAgents = async () => {
    try {
      setRefreshing(true)
      setError(null)
      const response = await fetch('/api/actions/console/summary/agents-workbench')
      if (response.ok) {
        const data = await response.json()
        const normalized = (data.agents || []).map((agent: any) => ({
          ...agent,
          status: agent.status || 'unknown',
          platform: {
            ...agent.platform,
            primary_ip: agent.platform?.primary_ip || agent.network?.addrs?.[0]?.split('/')[0] || 'unknown',
            hostname: agent.platform?.hostname || agent.hostname,
            os: agent.platform?.os || 'linux',
            kernel_version: agent.platform?.kernel_version || 'unknown',
            architecture: agent.platform?.architecture || agent.platform?.arch || 'unknown',
            fqdn: agent.platform?.fqdn || agent.hostname,
            cpu_model: agent.platform?.cpu_model || 'Unknown',
            memory_gb: agent.platform?.memory_gb || 0,
            disk_gb: agent.platform?.disk_gb || 0,
          },
          network: {
            ...agent.network,
            primary_ip: agent.network?.primary_ip || agent.network?.addrs?.[0]?.split('/')[0] || 'unknown',
            mac_address: agent.network?.mac_address || agent.network?.ifaces?.ens160?.mac || 'unknown',
            subnet: agent.network?.subnet || agent.network?.addrs?.[0] || 'unknown',
            gateway: agent.network?.gateway || 'unknown',
            dns_servers: agent.network?.dns_servers || [],
          },
          capabilities: agent.capabilities || {
            ebpf_loading: true, ebpf_attach: true, map_operations: true,
            kernel_modules: ['bpf'], supported_hooks: ['tc', 'xdp'],
            max_programs: 10, max_maps: 50,
          },
          visibility: agent.visibility || null,
        }))
        setAgents(
          normalized.sort((a: any, b: any) =>
            new Date(b.last_seen).getTime() - new Date(a.last_seen).getTime()
          )
        )
      } else {
        setError('Failed to fetch agents')
      }
    } catch (err) {
      setError('Failed to fetch agents')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  const isPackStatusStale = (status?: DetectionPackStatus | null) => {
    if (!status) return false
    if (status.computed_stale) return true
    if (!status.last_check_at_ms) return false
    return Date.now() - status.last_check_at_ms > ROLLOUT_STALE_MS
  }

  const eventTypeCount = (agent: Agent, eventType: string) =>
    Number(agent.visibility?.event_type_count?.[eventType] || 0)

  /* ── Computed counts for tab pills ─────────────────────────── */
  const tabCounts = useMemo(() => {
    const c = (fn: (a: Agent) => boolean) => agents.filter(fn).length
    return {
      all:             agents.length,
      online:          c((a) => a.status === 'online'),
      offline:         c((a) => a.status === 'offline'),
      windows:         c((a) => (a.platform.os || '').toLowerCase().includes('win')),
      linux:           c((a) => (a.platform.os || '').toLowerCase().includes('linux')),
      pack_issue:      c((a) => {
        const st = a.detection_pack_status
        return Boolean(st && st.rollout_state && st.rollout_state !== 'applied')
      }),
      needs_attention: c((a) => a.readiness?.bucket === 'needs_attention' || a.readiness?.bucket === 'degraded'),
    }
  }, [agents])

  /* ── Filtered list ──────────────────────────────────────────── */
  const filteredAgents = useMemo(() => {
    const byFilter = agents.filter((agent) => {
      switch (quickFilter) {
        case 'online':          return agent.status === 'online'
        case 'offline':         return agent.status === 'offline'
        case 'windows':         return (agent.platform.os || '').toLowerCase().includes('win')
        case 'linux':           return (agent.platform.os || '').toLowerCase().includes('linux')
        case 'pack_issue': {
          const st = agent.detection_pack_status
          return Boolean(st && st.rollout_state && st.rollout_state !== 'applied')
        }
        case 'needs_attention': return agent.readiness?.bucket === 'needs_attention' || agent.readiness?.bucket === 'degraded'
        default:                return true
      }
    })
    const needle = query.trim().toLowerCase()
    if (!needle) return byFilter
    return byFilter.filter((a) =>
      `${a.hostname} ${a.host_id} ${a.agent_uid} ${a.platform.os} ${a.platform.primary_ip}`.toLowerCase().includes(needle)
    )
  }, [agents, quickFilter, query])

  /* ── Summary card values ────────────────────────────────────── */
  const onlineCount = tabCounts.online
  const readyCount = agents.filter((a) => a.readiness?.bucket === 'ready').length
  const packIssueCount = tabCounts.pack_issue
  const needsAttentionCount = tabCounts.needs_attention
  const currentFocusLabel =
    quickFilter === 'all'             ? 'Full fleet overview'
    : quickFilter === 'online'          ? 'Online agents'
    : quickFilter === 'offline'         ? 'Offline agents'
    : quickFilter === 'windows'         ? 'Windows agents'
    : quickFilter === 'linux'           ? 'Linux agents'
    : quickFilter === 'pack_issue'      ? 'Detection pack issues'
    : 'Readiness attention needed'
  const nextBestAction =
    agents.length === 0
      ? 'No agents registered yet. Start the AegisFlux agent on an endpoint to begin visibility collection.'
      : packIssueCount > 0
        ? `${packIssueCount} agent${packIssueCount > 1 ? 's' : ''} have detection pack issues — review rollout state and compatibility.`
        : needsAttentionCount > 0
          ? `${needsAttentionCount} agent${needsAttentionCount > 1 ? 's' : ''} need attention — check readiness score and fix_first guidance.`
          : 'All agents are healthy. Open an agent record to confirm evidence quality before taking control actions.'

  /* ── Pack health badge ──────────────────────────────────────── */
  const PackHealthBadge = ({ status }: { status?: DetectionPackStatus | null }) => {
    if (!status) return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-gray-100 text-gray-600">No telemetry</span>
    if (isPackStatusStale(status)) return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-800">Stale</span>
    const rollout = status.rollout_state
    if (rollout === 'applied')       return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800">Applied</span>
    if (rollout === 'rejected')      return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-rose-100 text-rose-800">Rejected</span>
    if (rollout === 'incompatible')  return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-800">Incompatible</span>
    if (rollout === 'rollback')      return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-rose-100 text-rose-800">Rollback</span>
    return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-gray-100 text-gray-600">No pack</span>
  }

  /* ── Render ─────────────────────────────────────────────────── */
  return (
    <div>
      {/* Page header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Agents</h1>
          <p className="text-sm text-gray-500 mt-0.5">Registered AegisFlux agents and their readiness state.</p>
        </div>
        <button
          type="button"
          onClick={fetchAgents}
          disabled={refreshing}
          className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-[#1e3a5f] rounded-md hover:bg-[#162d4a] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
          {refreshing ? 'Refreshing…' : 'Refresh'}
        </button>
      </div>

      {/* Error */}
      {error && (
        <div className="mb-4 flex items-center gap-3 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Summary stat cards */}
      <div className="mb-6 grid gap-4 lg:grid-cols-4">
        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Agent Fleet</p>
          <p className="mt-2 text-2xl font-semibold text-slate-900">{agents.length.toLocaleString()}</p>
          <p className="mt-2 text-sm text-slate-600">
            {onlineCount} online · {agents.length - onlineCount} offline
          </p>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Readiness</p>
          <p className="mt-2 text-2xl font-semibold text-slate-900">{readyCount}</p>
          <p className="mt-2 text-sm text-slate-600">
            {readyCount} ready · {needsAttentionCount} need attention
          </p>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Current Focus</p>
          <p className="mt-2 text-lg font-semibold text-slate-900">{currentFocusLabel}</p>
          <p className="mt-2 text-sm text-slate-600">Use the filter tabs to narrow to a specific group.</p>
        </div>
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-amber-700">Next Best Action</p>
          <p className="mt-2 text-sm font-medium text-amber-900">{nextBestAction}</p>
          {packIssueCount > 0 && (
            <p className="mt-2 text-sm text-amber-800">{packIssueCount} pack issue{packIssueCount > 1 ? 's' : ''} require review.</p>
          )}
        </div>
      </div>

      {/* Quick filter tab bar */}
      <div className="flex flex-wrap items-end justify-between border-b border-gray-200 mb-4 gap-2">
        <div className="flex flex-wrap items-center gap-0">
          {QUICK_FILTERS.map((f) => (
            <button
              key={f.id}
              type="button"
              onClick={() => setQuickFilter(f.id)}
              className={`px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap ${
                quickFilter === f.id
                  ? 'border-[#1e3a5f] text-[#1e3a5f]'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              {f.label}
              <span
                className={`ml-1.5 text-xs px-1.5 py-0.5 rounded-full ${
                  quickFilter === f.id
                    ? 'bg-blue-100 text-blue-700'
                    : 'bg-gray-100 text-gray-500'
                }`}
              >
                {tabCounts[f.id]}
              </span>
            </button>
          ))}
        </div>
      </div>

      {/* Search filter bar */}
      <div className="bg-white rounded-lg shadow p-4 mb-6">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search by hostname, ID, OS, or IP…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent"
          />
        </div>
      </div>

      {/* Agent table */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500 text-sm">Loading agents…</div>
        ) : agents.length === 0 ? (
          <EmptyState
            title="No agents registered"
            message="Start the AegisFlux agent on an endpoint to see it here."
          />
        ) : filteredAgents.length === 0 ? (
          <div className="p-10 text-center">
            <p className="text-gray-500 text-sm mb-3">
              No agents match the "{QUICK_FILTERS.find((f) => f.id === quickFilter)?.label}" filter.
            </p>
            <button
              type="button"
              onClick={() => setQuickFilter('all')}
              className="text-sm text-[#1e3a5f] hover:underline"
            >
              Show all agents
            </button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Agent</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Platform</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Readiness</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Pack health</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Signals</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {filteredAgents.map((agent) => {
                  const stale = Date.now() - new Date(agent.last_seen).getTime() > 5 * 60 * 1000
                  const findings = eventTypeCount(agent, 'aegis.risk_finding.created')
                  return (
                    <tr
                      key={agent.agent_uid}
                      className="hover:bg-gray-50 cursor-pointer transition-colors"
                      onClick={() => window.location.href = `/agents/${encodeURIComponent(agent.host_id || agent.agent_uid)}`}
                    >
                      {/* Agent — hostname + UID */}
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-3">
                          <OSIcon os={agent.platform.os} />
                          <div>
                            <div className="text-sm font-medium text-gray-900 flex items-center gap-1.5">
                              {formatHostname(agent.hostname || agent.host_id)}
                              <CopyValueButton value={agent.hostname || agent.host_id} label="Copy hostname" />
                            </div>
                            <div className="text-xs font-mono text-gray-400 mt-0.5 flex items-center gap-1">
                              {formatAgentId(agent.agent_uid)}
                              <CopyValueButton value={agent.agent_uid} label="Copy agent ID" />
                            </div>
                          </div>
                        </div>
                      </td>

                      {/* Platform */}
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm text-gray-900">
                          {agent.platform.os || 'unknown'} · {agent.platform.architecture || 'unknown'}
                        </div>
                        <div className="text-xs font-mono text-gray-500 mt-0.5">
                          {agent.platform.primary_ip || 'n/a'}
                        </div>
                        {agent.labels.length > 0 && (
                          <div className="mt-1 flex flex-wrap gap-1">
                            {agent.labels.slice(0, 2).map((label) => (
                              <span
                                key={label}
                                className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700"
                              >
                                {label}
                              </span>
                            ))}
                          </div>
                        )}
                      </td>

                      {/* Status + last seen */}
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="mb-1">
                          {agent.status === 'online' && !stale
                            ? <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800">Online</span>
                            : stale
                              ? <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-800">Stale</span>
                              : <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-rose-100 text-rose-800">Offline</span>
                          }
                        </div>
                        <FreshnessDot lastSeen={agent.last_seen} />
                      </td>

                      {/* Readiness */}
                      <td className="px-6 py-4 whitespace-nowrap">
                        <ReadinessBadge readiness={agent.readiness} />
                        {agent.readiness?.fix_first && (
                          <p
                            className="mt-1 text-xs text-gray-500 max-w-[200px] truncate"
                            title={agent.readiness.fix_first}
                          >
                            {agent.readiness.fix_first}
                          </p>
                        )}
                      </td>

                      {/* Pack health */}
                      <td className="px-6 py-4 whitespace-nowrap">
                        <PackHealthBadge status={agent.detection_pack_status} />
                        {agent.detection_pack_status?.active_pack_id && (
                          <div
                            className="mt-1 text-xs font-mono text-gray-400 max-w-[180px] truncate"
                            title={`${agent.detection_pack_status.active_pack_id}@${agent.detection_pack_status.active_pack_version || ''}`}
                          >
                            {formatHash(`${agent.detection_pack_status.active_pack_id}@${agent.detection_pack_status.active_pack_version || ''}`)}
                          </div>
                        )}
                      </td>

                      {/* Signals */}
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        <div className="flex items-center gap-1">
                          {findings > 0 && <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />}
                          <span className={findings > 0 ? 'font-semibold text-amber-800' : ''}>
                            {findings} finding{findings !== 1 ? 's' : ''}
                          </span>
                        </div>
                        <div className="text-xs text-gray-400 mt-0.5">
                          {eventTypeCount(agent, 'aegis.collector.status')} collector events
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <DetailModal
        open={Boolean(detailModal)}
        title={detailModal?.title || 'Detail'}
        detail={detailModal?.payload || {}}
        onClose={() => setDetailModal(null)}
      />
    </div>
  )
}
