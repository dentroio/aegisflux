'use client'

import { useEffect, useMemo, useState } from 'react'
import { 
  ArrowLeft, 
  Users, 
  AlertTriangle,
  RefreshCw,
  ShieldCheck,
  Search,
} from 'lucide-react'
import {
  BoundedTable,
  CopyValueButton,
  DetailModal,
  EmptyState,
  FilterBar,
  FormattedValue,
  KpiTile,
  SummaryStrip,
  WorkbenchHeader,
} from '@/components/workbench/primitives'
import { formatAgentId, formatDateTime, formatHash, formatHostname, formatRelativeAge } from '@/shared/formatting'

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
    ifaces: Record<string, {
      addrs: string[]
      mac: string
    }>
  }
  labels: string[]
  note: string
  created: string
  last_seen: string
  status: 'online' | 'offline' | 'unknown'
  detection_pack_status?: DetectionPackStatus | null
  visibility?: VisibilitySummary | null
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

const ROLLOUT_STALE_MS = 24 * 60 * 60 * 1000

export function AgentsManagementPanel({ embedded = false }: { embedded?: boolean }) {
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [view, setView] = useState<'agents' | 'rollout'>('agents')
  const [embeddedFilter, setEmbeddedFilter] = useState<
    'all' | 'online' | 'offline' | 'windows' | 'linux' | 'pack_issue' | 'budget'
  >(() => {
    if (typeof window === 'undefined') return 'all'
    const saved = window.localStorage.getItem('aegis.agents.workbench.filter')
    return saved === 'online' ||
      saved === 'offline' ||
      saved === 'windows' ||
      saved === 'linux' ||
      saved === 'pack_issue' ||
      saved === 'budget'
      ? saved
      : 'all'
  })
  const [query, setQuery] = useState('')
  const [detailModal, setDetailModal] = useState<{ title: string; payload: unknown } | null>(null)

  useEffect(() => {
    window.localStorage.setItem('aegis.agents.workbench.filter', embeddedFilter)
  }, [embeddedFilter])

  useEffect(() => {
    fetchAgents()
    const interval = setInterval(fetchAgents, 30000) // Refresh every 30 seconds
    return () => clearInterval(interval)
  }, [])

  const fetchAgents = async () => {
    try {
      setRefreshing(true)
      setError(null)

      const [response, visibilityResponse] = await Promise.all([
        fetch('/api/actions/agents'),
        fetch('/api/visibility/devices?limit=200')
      ])
      if (response.ok) {
        const data = await response.json()
        const visibilityData = visibilityResponse.ok ? await visibilityResponse.json() : {}
        const visibilityByDevice = new Map<string, VisibilitySummary>(
          (visibilityData.devices || []).map((device: any) => [device.device_id, device])
        )
        // Ensure all agents have a status property and normalize data structure
        const agentsWithStatus = (data.agents || []).map((agent: any) => ({
          ...agent,
          status: agent.status || 'unknown',
          // Normalize platform structure for different agent formats
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
            disk_gb: agent.platform?.disk_gb || 0
          },
          // Normalize network structure
          network: {
            ...agent.network,
            primary_ip: agent.network?.primary_ip || agent.network?.addrs?.[0]?.split('/')[0] || 'unknown',
            mac_address: agent.network?.mac_address || agent.network?.ifaces?.ens160?.mac || 'unknown',
            subnet: agent.network?.subnet || agent.network?.addrs?.[0] || 'unknown',
            gateway: agent.network?.gateway || 'unknown',
            dns_servers: agent.network?.dns_servers || []
          },
          // Add default capabilities if missing
          capabilities: agent.capabilities || {
            ebpf_loading: true,
            ebpf_attach: true,
            map_operations: true,
            kernel_modules: ['bpf'],
            supported_hooks: ['tc', 'xdp'],
            max_programs: 10,
            max_maps: 50
          },
          visibility: visibilityByDevice.get(agent.host_id) || visibilityByDevice.get(agent.agent_uid) || null
        }))
        // Sort agents by last_seen (most recent first) and group by host_id
        const sortedAgents = agentsWithStatus.sort((a: any, b: any) => 
          new Date(b.last_seen).getTime() - new Date(a.last_seen).getTime()
        )
        setAgents(sortedAgents)
      } else {
        setError('Failed to fetch agents')
      }
    } catch (err) {
      setError('Failed to fetch agents')
      console.error('Agent fetch error:', err)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  const getStatusBadge = (status: string, stale: boolean) => {
    if (stale) return <span className="badge badge-warning">Stale</span>
    switch (status) {
      case 'online':
        return <span className="badge badge-success">Online</span>
      case 'offline':
        return <span className="badge badge-danger">Offline</span>
      default:
        return <span className="badge badge-warning">Unknown</span>
    }
  }

  const getRolloutBadge = (rolloutState?: string) => {
    switch (rolloutState) {
      case 'applied':
        return <span className="badge badge-success">Applied</span>
      case 'rejected':
        return <span className="badge badge-danger">Rejected</span>
      case 'incompatible':
        return <span className="badge badge-warning">Incompatible</span>
      case 'expired':
        return <span className="badge badge-warning">Expired</span>
      case 'stale':
        return <span className="badge badge-warning">Stale</span>
      case 'rollback':
        return <span className="badge badge-danger">Rollback</span>
      case 'not_checked':
        return <span className="badge badge-warning">Not checked</span>
      default:
        return <span className="badge badge-warning">No pack</span>
    }
  }

  const formatDate = (dateString: string) => formatDateTime(dateString)

  const isPackStatusStale = (status?: DetectionPackStatus | null) => {
    if (!status) return false
    if (status.computed_stale) return true
    if (!status.last_check_at_ms) return false
    return Date.now() - status.last_check_at_ms > ROLLOUT_STALE_MS
  }

  const getPackHealthBadge = (status?: DetectionPackStatus | null) => {
    if (!status) return <span className="badge badge-warning">No telemetry</span>
    if (isPackStatusStale(status)) return <span className="badge badge-warning">Stale</span>
    return getRolloutBadge(status.rollout_state)
  }

  const formatTimeMS = (value?: number) => {
    return value ? formatDate(new Date(value).toISOString()) : 'n/a'
  }

  const rolloutRows = useMemo(() => agents
    .filter((agent) => agent.detection_pack_status)
    .map((agent) => ({
      agent,
      status: agent.detection_pack_status as DetectionPackStatus,
      stale: isPackStatusStale(agent.detection_pack_status)
    })), [agents])

  const rolloutCounts = rolloutRows.reduce<Record<string, number>>((counts, row) => {
    const key = row.stale ? 'stale' : (row.status.rollout_state || 'not_checked')
    counts[key] = (counts[key] || 0) + 1
    return counts
  }, {})

  const activePackCount = new Set(
    rolloutRows
      .map((row) => `${row.status.active_pack_id || 'none'}@${row.status.active_pack_version || 'none'}`)
      .filter((pack) => pack !== 'none@none')
  ).size

  const eventTypeCount = (agent: Agent, eventType: string) => {
    return Number(agent.visibility?.event_type_count?.[eventType] || 0)
  }

  const workbenchAgents = useMemo(() => {
    const byFilter = agents.filter((agent) => {
      switch (embeddedFilter) {
        case 'all':
          return true
        case 'online':
          return agent.status === 'online'
        case 'offline':
          return agent.status === 'offline'
        case 'windows':
          return (agent.platform.os || '').toLowerCase().includes('win')
        case 'linux':
          return (agent.platform.os || '').toLowerCase().includes('linux')
        case 'pack_issue':
          const st = agent.detection_pack_status
          return Boolean(st && st.rollout_state && st.rollout_state !== 'applied')
        case 'budget':
          return eventTypeCount(agent, 'aegis.agent.performance') > 3 || eventTypeCount(agent, 'aegis.collector.status') > 12
        default:
          return true
      }
    })
    const needle = query.trim().toLowerCase()
    if (!needle) return byFilter
    return byFilter.filter((agent) =>
      `${agent.hostname} ${agent.host_id} ${agent.agent_uid} ${agent.platform.os} ${agent.platform.primary_ip}`.toLowerCase().includes(needle),
    )
  }, [agents, embeddedFilter, query])

  const filterCounts = useMemo(() => {
    const c = (fn: (a: Agent) => boolean) => agents.filter(fn).length
    return {
      all: agents.length,
      online: c((a) => a.status === 'online'),
      offline: c((a) => a.status === 'offline'),
      windows: c((a) => (a.platform.os || '').toLowerCase().includes('win')),
      linux: c((a) => (a.platform.os || '').toLowerCase().includes('linux')),
      pack_issue: c((a) => {
        const st = a.detection_pack_status
        return Boolean(st && st.rollout_state && st.rollout_state !== 'applied')
      }),
      budget: c(
        (a) =>
          Number(a.visibility?.event_type_count?.['aegis.agent.performance'] || 0) > 3 ||
          Number(a.visibility?.event_type_count?.['aegis.collector.status'] || 0) > 12,
      ),
    }
  }, [agents])

  const staleAgents = agents.filter((agent) => Date.now() - new Date(agent.last_seen).getTime() > 5 * 60 * 1000).length
  const packIssueAgents = filterCounts.pack_issue
  const budgetPressureAgents = filterCounts.budget

  return (
    <div className={embedded ? 'bg-gray-50' : 'min-h-screen bg-gray-50'}>
      {!embedded && (
        <header className="bg-white border-b border-gray-200">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
            <WorkbenchHeader
              title="Agents Workbench"
              subtitle="Find agents needing attention and route to deliberate detail."
              actions={
                <>
                  <a href="/" className="btn btn-secondary h-9 px-3">
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Dashboard
                  </a>
                  <button onClick={fetchAgents} disabled={refreshing} className="btn btn-secondary h-9 px-3">
                    <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
                    Refresh
                  </button>
                </>
              }
            />
          </div>
        </header>
      )}

      <div className={embedded ? 'max-w-[1500px] mx-auto px-0 py-4' : 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8'}>
        {error && (
          <div className="mb-6 bg-danger-50 border border-danger-200 rounded-md p-4">
            <div className="flex">
              <AlertTriangle className="h-5 w-5 text-danger-400" />
              <div className="ml-3">
                <p className="text-sm text-danger-800">{error}</p>
              </div>
            </div>
          </div>
        )}

        <SummaryStrip>
          <KpiTile label="Total agents" value={agents.length} />
          <KpiTile label="Stale" value={staleAgents} />
          <KpiTile label="Pack issues" value={packIssueAgents} />
          <KpiTile label="Budget pressure" value={budgetPressureAgents} />
        </SummaryStrip>

        <FilterBar>
          {(
            [
              ['all', filterCounts.all],
              ['online', filterCounts.online],
              ['offline', filterCounts.offline],
              ['windows', filterCounts.windows],
              ['linux', filterCounts.linux],
              ['pack_issue', filterCounts.pack_issue],
              ['budget', filterCounts.budget],
            ] as const
          ).map(([id, count]) => (
            <button
              key={id}
              type="button"
              onClick={() => setEmbeddedFilter(id)}
              className={`rounded-full border px-3 py-1 text-xs font-semibold ${
                embeddedFilter === id
                  ? 'border-primary-600 bg-primary-600 text-white'
                  : 'border-gray-200 bg-gray-50 text-gray-700 hover:bg-gray-100'
              }`}
            >
              {id.replace('_', ' ')} ({count})
            </button>
          ))}
          <label className="relative min-w-[220px] flex-1">
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-gray-400" />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              className="input h-9 pl-9"
              placeholder="Search host, id, OS, IP"
            />
          </label>
          <div className="ml-auto inline-flex rounded-md border border-gray-200 bg-white p-1 text-xs">
            <button
              type="button"
              onClick={() => setView('agents')}
              className={`rounded px-2 py-1 font-semibold ${view === 'agents' ? 'bg-primary-600 text-white' : 'text-gray-700'}`}
            >
              Agents
            </button>
            <button
              type="button"
              onClick={() => setView('rollout')}
              className={`rounded px-2 py-1 font-semibold ${view === 'rollout' ? 'bg-primary-600 text-white' : 'text-gray-700'}`}
            >
              Rollout
            </button>
          </div>
        </FilterBar>

        <section className="card p-5">
          {loading ? (
            <EmptyState title="Loading agents" message="Collecting agent and rollout telemetry." />
          ) : view === 'agents' ? (
            workbenchAgents.length === 0 ? (
              <EmptyState title="No matching agents" message="Try adjusting the filter or search query." />
            ) : (
              <BoundedTable
                headers={['Identity', 'Status', 'Platform', 'Pack health', 'Signals', 'Action']}
                rows={workbenchAgents.map((agent) => {
                  const stale = Date.now() - new Date(agent.last_seen).getTime() > 5 * 60 * 1000
                  return [
                    <div key={`${agent.agent_uid}-id`} className="space-y-1">
                      <div className="flex items-center gap-2">
                        <FormattedValue value={formatHostname(agent.hostname || agent.host_id)} fullValue={agent.hostname || agent.host_id} mono={false} />
                        <CopyValueButton value={agent.hostname || agent.host_id} label="Copy hostname" />
                      </div>
                      <div className="flex items-center gap-2">
                        <FormattedValue value={formatAgentId(agent.agent_uid)} fullValue={agent.agent_uid} />
                        <CopyValueButton value={agent.agent_uid} label="Copy agent id" />
                      </div>
                    </div>,
                    <div key={`${agent.agent_uid}-status`} className="space-y-1">
                      {getStatusBadge(agent.status, stale)}
                      <p className="text-xs text-slate-500" title={formatDate(agent.last_seen)}>
                        Last seen {formatRelativeAge(agent.last_seen)}
                      </p>
                    </div>,
                    <div key={`${agent.agent_uid}-platform`} className="text-xs text-slate-600">
                      <p>{agent.platform.os || 'unknown'} · {agent.platform.architecture || 'unknown'}</p>
                      <p className="truncate">{agent.platform.primary_ip || 'n/a'}</p>
                    </div>,
                    <div key={`${agent.agent_uid}-pack`}>{getPackHealthBadge(agent.detection_pack_status)}</div>,
                    <div key={`${agent.agent_uid}-signals`} className="text-xs text-slate-600">
                      <p>Findings: {eventTypeCount(agent, 'aegis.risk_finding.created')}</p>
                      <p>Collectors: {eventTypeCount(agent, 'aegis.collector.status')}</p>
                      <p>Budget events: {eventTypeCount(agent, 'aegis.agent.performance')}</p>
                    </div>,
                    <a
                      key={`${agent.agent_uid}-action`}
                      href={`/agents/${encodeURIComponent(agent.host_id || agent.agent_uid)}`}
                      className="text-xs font-semibold text-primary-700 hover:text-primary-900"
                    >
                      Open detail
                    </a>,
                  ]
                })}
              />
            )
          ) : rolloutRows.length === 0 ? (
            <EmptyState title="No rollout telemetry" message="No detection-pack rollout telemetry is available." />
          ) : (
            <div className="space-y-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-slate-700">
                <ShieldCheck className="h-4 w-4 text-primary-600" />
                Detection pack rollout
              </div>
              <BoundedTable
                headers={['Agent', 'State', 'Active pack', 'Trust', 'Last check', 'Action']}
                rows={rolloutRows.map(({ agent, status }) => ([
                  <div key={`agent-${agent.agent_uid}`}>
                    <p className="text-sm font-medium text-slate-900">{formatHostname(agent.hostname || agent.host_id)}</p>
                    <p className="text-xs text-gray-500">{agent.platform.os} • {agent.agent_version}</p>
                  </div>,
                  <div key={`state-${agent.agent_uid}`}>{getPackHealthBadge(status)}</div>,
                  <div key={`pack-${agent.agent_uid}`} className="flex items-center gap-2">
                    <FormattedValue
                      value={formatHash(`${status.active_pack_id || 'none'}@${status.active_pack_version || 'none'}`)}
                      fullValue={`${status.active_pack_id || 'none'} @ ${status.active_pack_version || 'none'}`}
                    />
                    <CopyValueButton value={`${status.active_pack_id || 'none'} @ ${status.active_pack_version || 'none'}`} />
                  </div>,
                  <button
                    key={`trust-${agent.agent_uid}`}
                    className="text-xs text-primary-700 underline underline-offset-2"
                    onClick={() => setDetailModal({ title: `Trust detail: ${agent.hostname || formatAgentId(agent.agent_uid)}`, payload: status })}
                  >
                    View trust detail
                  </button>,
                  <span key={`check-${agent.agent_uid}`} className="text-xs text-gray-600" title={formatTimeMS(status.last_check_at_ms)}>
                    {status.last_check_at_ms ? formatRelativeAge(status.last_check_at_ms) : 'n/a'}
                  </span>,
                  <a
                    key={`action-${agent.agent_uid}`}
                    href={`/agents/${encodeURIComponent(agent.host_id || agent.agent_uid)}`}
                    className="text-xs font-semibold text-primary-700 hover:text-primary-900"
                  >
                    Open detail
                  </a>,
                ]))}
              />
            </div>
          )}
        </section>
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
