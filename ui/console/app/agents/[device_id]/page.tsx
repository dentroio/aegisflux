'use client'

import { useEffect, useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import {
  Activity,
  AlertTriangle,
  ArrowLeft,
  CheckCircle,
  Chrome,
  Clock,
  Cpu,
  Database,
  Globe2,
  Network,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
} from 'lucide-react'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import { AbomPanel } from '@/components/AbomPanel'
import { EvidenceGraphPanel } from '@/components/EvidenceGraphPanel'

type DeviceRecord = {
  device_id: string
  agent_id?: string
  source?: string
  sensor_version?: string
  first_seen_ms?: number
  last_seen_ms?: number
  last_event_type?: string
  event_count?: number
  event_type_count?: Record<string, number>
}

type EventRecord = {
  event_id: string
  event_type: string
  timestamp_ms: number
  received_at_ms?: number
  source?: string
  device_id?: string
  agent_id?: string
  sensor_version?: string
  payload?: Record<string, any>
}

type ProcessRecord = {
  event_id?: string
  timestamp_ms?: number
  process_guid?: string
  pid?: number
  ppid?: number
  name?: string
  path?: string
  command_line?: string
  user?: string
}

type FlowRecord = {
  event_id?: string
  timestamp_ms?: number
  flow_id?: string
  process_name?: string
  pid?: number
  protocol?: string
  direction?: string
  remote_ip?: string
  remote_port?: number
  remote_hostname?: string
}

type DnsRecord = {
  event_id?: string
  timestamp_ms?: number
  query?: string
  query_type?: string
  answers?: string[]
  resolver?: string
  process_guid?: string
  pid?: number
  correlation_method?: string
}

type FindingRecord = {
  event_id?: string
  timestamp_ms?: number
  finding_id?: string
  detection_id?: string
  severity?: string
  title?: string
  description?: string
  classification?: string
  risk_score?: number
  detected_patterns?: string[]
  recommended_action?: string
}

type BrowserExtension = {
  event_id?: string
  timestamp_ms?: number
  browser?: string
  profile?: string
  extension_id?: string
  name?: string
  version?: string
  permissions?: string[]
  host_permissions?: string[]
}

type SaseComponent = {
  event_id?: string
  timestamp_ms?: number
  component_type?: string
  vendor?: string
  product?: string
  name?: string
  status?: string
}

type CollectorStatus = {
  event_id?: string
  timestamp_ms?: number
  collector?: string
  status?: string
  message?: string
}

type AgentPerformance = {
  event_id?: string
  timestamp_ms?: number
  os?: string
  process_cpu_percent?: number | null
  process_memory_rss_mb?: number | null
  collector_runtime_ms?: number
  collector_name?: string
  collection_interval_ms?: number | null
  skipped_reason?: string | null
  event_queue_depth?: number
  spool_bytes?: number
  pack_eval_runtime_ms?: number | null
}

type VisibilityState = {
  devices: DeviceRecord[]
  events: EventRecord[]
  processes: ProcessRecord[]
  flows: FlowRecord[]
  dns: DnsRecord[]
  findings: FindingRecord[]
  extensions: BrowserExtension[]
  sase: SaseComponent[]
  collectors: CollectorStatus[]
  performance: AgentPerformance[]
}

const tabs = ['Overview', 'AI Capability', 'Evidence Path', 'Activity', 'Software & Controls', 'Detection Packs', 'Health', 'Policy']

const aiPattern = /chatgpt|openai|anthropic|claude|gemini|copilot|mistral|ollama|litellm|vllm|mcp|modelcontextprotocol/i

async function fetchJson<T>(url: string, fallback: T): Promise<T> {
  try {
    const response = await fetch(url, { cache: 'no-store' })
    if (!response.ok) return fallback
    return await response.json()
  } catch {
    return fallback
  }
}

export default function DeviceDetailPage({ params }: { params: { device_id: string } }) {
  const router = useRouter()
  const deviceId = decodeURIComponent(params.device_id)
  const [authGate, setAuthGate] = useState(false)
  const [activeTab, setActiveTab] = useState('Overview')
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [data, setData] = useState<VisibilityState>({
    devices: [],
    events: [],
    processes: [],
    flows: [],
    dns: [],
    findings: [],
    extensions: [],
    sase: [],
    collectors: [],
    performance: [],
  })

  useEffect(() => {
    if (!readLabAuthenticated()) {
      const next = `${window.location.pathname}${window.location.search}`
      window.location.replace(`/?next=${encodeURIComponent(next)}`)
      return
    }
    setAuthGate(true)
  }, [router])

  useEffect(() => {
    if (!authGate) return undefined
    loadDevice()
    return undefined
  }, [deviceId, authGate])

  const loadDevice = async () => {
    setRefreshing(true)
    const encoded = encodeURIComponent(deviceId)
    const res = await fetch(`/api/visibility/summary/device?device_id=${encoded}`, { cache: 'no-store' })
    if (!res.ok) {
      const [devices, events, processes, flows, dns, findings, extensions, sase, collectors, performance] = await Promise.all([
        fetchJson<{ devices?: DeviceRecord[] }>(`/api/visibility/devices?limit=120`, {}),
        fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?device_id=${encoded}&limit=120`, {}),
        fetchJson<{ processes?: ProcessRecord[] }>(`/api/visibility/processes?device_id=${encoded}&limit=80`, {}),
        fetchJson<{ flows?: FlowRecord[] }>(`/api/visibility/flows?device_id=${encoded}&limit=80`, {}),
        fetchJson<{ observations?: DnsRecord[]; dns?: DnsRecord[] }>(`/api/visibility/dns?device_id=${encoded}&limit=80`, {}),
        fetchJson<{ findings?: FindingRecord[] }>(`/api/visibility/findings?device_id=${encoded}&limit=80`, {}),
        fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?device_id=${encoded}&event_type=aegis.browser_extension.observed&limit=80`, {}),
        fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?device_id=${encoded}&event_type=aegis.sase_component.observed&limit=80`, {}),
        fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?device_id=${encoded}&event_type=aegis.collector.status&limit=80`, {}),
        fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?device_id=${encoded}&event_type=aegis.agent.performance&limit=80`, {}),
      ])
      setData({
        devices: devices.devices || [],
        events: events.events || [],
        processes: processes.processes || [],
        flows: flows.flows || [],
        dns: dns.observations || dns.dns || [],
        findings: findings.findings || [],
        extensions: (extensions.events || []).map((event) => ({ event_id: event.event_id, timestamp_ms: event.timestamp_ms, ...event.payload })),
        sase: (sase.events || []).map((event) => ({ event_id: event.event_id, timestamp_ms: event.timestamp_ms, ...event.payload })),
        collectors: (collectors.events || []).map((event) => ({ event_id: event.event_id, timestamp_ms: event.timestamp_ms, ...event.payload })),
        performance: (performance.events || []).map((event) => ({ event_id: event.event_id, timestamp_ms: event.timestamp_ms, ...event.payload })),
      })
      setLoading(false)
      setRefreshing(false)
      return
    }
    const bundle = await res.json()
    const mapPayloadEvent = (event: EventRecord & { payload?: Record<string, unknown> }) => {
      const payload = event.payload && typeof event.payload === 'object' ? event.payload : {}
      return { event_id: event.event_id, timestamp_ms: event.timestamp_ms, ...payload }
    }
    setData({
      devices: bundle.devices || [],
      events: (bundle.events || []) as EventRecord[],
      processes: (bundle.processes || []) as ProcessRecord[],
      flows: (bundle.flows || []) as FlowRecord[],
      dns: (bundle.dns || []) as DnsRecord[],
      findings: (bundle.findings || []) as FindingRecord[],
      extensions: (bundle.extension_events || []).map((event: EventRecord) => mapPayloadEvent(event)),
      sase: (bundle.sase_events || []).map((event: EventRecord) => mapPayloadEvent(event)),
      collectors: (bundle.collector_events || []).map((event: EventRecord) => mapPayloadEvent(event)),
      performance: (bundle.performance_events || []).map((event: EventRecord) => mapPayloadEvent(event)),
    })
    setLoading(false)
    setRefreshing(false)
  }

  const device = useMemo(() => {
    return data.devices.find((record) => record.device_id === deviceId) || {
      device_id: deviceId,
      agent_id: deviceId,
      event_count: data.events.length,
      event_type_count: {},
      last_seen_ms: latestTimestamp(data.events),
      source: data.events[0]?.source || 'visibility',
      sensor_version: data.events[0]?.sensor_version || 'unknown',
    }
  }, [data, deviceId])

  const isFresh = Boolean(device.last_seen_ms && Date.now() - device.last_seen_ms < 5 * 60 * 1000)
  const aiDns = data.dns.filter((record) => aiPattern.test(record.query || ''))
  const aiProcesses = data.processes.filter((record) => aiPattern.test(`${record.name || ''} ${record.path || ''} ${record.command_line || ''}`))
  const aiFindings = data.findings.filter((record) => aiPattern.test(`${record.title || ''} ${record.classification || ''} ${(record.detected_patterns || []).join(' ')}`))
  const filteredEvents = filterRows(data.events, query)

  const [analystBusy, setAnalystBusy] = useState(false)
  const [analystNote, setAnalystNote] = useState<string | null>(null)

  async function runEvidenceAnalyst() {
    setAnalystBusy(true)
    setAnalystNote(null)
    try {
      const payload = {
        device_id: deviceId,
        context: {
          findings: data.findings.length,
          processes: data.processes.length,
          flows: data.flows.length,
          dns: data.dns.length,
          collectors: data.collectors.filter((c) => c.status !== 'healthy').length,
          extensions: data.extensions.length,
          sase: data.sase.length,
        },
      }
      const res = await fetch('/api/actions/platform/ai/endpoint-evidence-analyst', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      const body = res.ok ? await res.json() : null
      setAnalystNote(
        body
          ? `${body.assessment} | ${body.confidence} → ${body.recommended_next_action}`
          : `Analyst request failed (HTTP ${res.status})`,
      )
    } finally {
      setAnalystBusy(false)
    }
  }

  if (!authGate) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 text-sm text-gray-600">
        Checking session… If this does not continue, <a className="ml-1 font-semibold text-primary-700 underline" href="/">open the console login</a>.
      </div>
    )
  }

  const shellHealth = isFresh
    ? { label: 'Fresh', tone: 'emerald' as const, text: 'Telemetry current' }
    : { label: 'Stale', tone: 'amber' as const, text: 'Telemetry stale' }

  function onLogoutShell() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  return (
    <ConsoleShell
      activeNavId="agents"
      breadcrumbs={[
        { label: 'Agents', href: '/agents' },
        { label: device.device_id },
      ]}
      health={shellHealth}
      onLogout={onLogoutShell}
    >
      <div className="min-w-0 bg-gray-50 pb-10">
        <div className="border-b border-gray-200 bg-white">
          <div className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4 px-4 py-4 sm:px-6 lg:px-8">
            <div className="flex min-w-0 items-center gap-3">
              <a href="/agents" className="btn btn-secondary h-9 shrink-0 px-3">
                <ArrowLeft className="mr-2 h-4 w-4" />
                Agents
              </a>
              <div className="min-w-0">
                <div className="flex min-w-0 flex-wrap items-center gap-2">
                  <Server className="h-5 w-5 shrink-0 text-primary-600" />
                  <h1 className="truncate text-xl font-bold text-gray-900">{device.device_id}</h1>
                  <FreshBadge fresh={isFresh} />
                </div>
                <p className="mt-1 truncate text-xs text-gray-500 sm:text-sm">
                  {device.source || 'visibility'} | {device.sensor_version || 'unknown'} | last seen{' '}
                  {formatAge(device.last_seen_ms)}
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={runEvidenceAnalyst}
                disabled={analystBusy}
                className="btn btn-secondary h-9 px-3 text-primary-800"
              >
                {analystBusy ? 'Analyzing…' : 'Explain AI activity'}
              </button>
              <a
                href={`/?panel=inventory&device=${encodeURIComponent(deviceId)}`}
                className="btn btn-secondary h-9 px-3 text-primary-700"
              >
                <Database className="mr-2 h-4 w-4" />
                Inventory
              </a>
              <button type="button" onClick={loadDevice} disabled={refreshing} className="btn btn-secondary h-9 px-3">
                <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            </div>
          </div>
        </div>
        {analystNote ? (
          <div className="mx-auto max-w-7xl px-4 pt-4 sm:px-6 lg:px-8">
            <div className="rounded-lg border border-slate-200 bg-white px-4 py-3 text-sm text-slate-700">{analystNote}</div>
          </div>
        ) : null}

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div className="mb-6 grid gap-4 md:grid-cols-5">
          <Metric label="Signals" value={device.event_count || data.events.length} />
          <Metric label="Observed programs" value={data.processes.length} />
          <Metric label="Network Flows" value={data.flows.length} />
          <Metric label="Domain lookups" value={data.dns.length} />
          <Metric label="Findings" value={data.findings.length} tone={data.findings.length ? 'warning' : 'normal'} />
        </div>

        {!isFresh && (
          <div className="mb-6 rounded-lg border border-warning-200 bg-warning-50 p-4 text-sm text-warning-800">
            <div className="flex items-start gap-3">
              <AlertTriangle className="mt-0.5 h-5 w-5" />
              <p>This device telemetry is stale. The detail page is showing the latest retained evidence from ingest.</p>
            </div>
          </div>
        )}

        <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex gap-2 overflow-x-auto pb-1">
            {tabs.map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`h-9 whitespace-nowrap rounded-md border px-3 text-sm font-medium ${
                  activeTab === tab ? 'border-primary-600 bg-primary-600 text-white' : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50'
                }`}
              >
                {tab}
              </button>
            ))}
          </div>
          <label className="relative min-w-0 lg:w-80">
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-gray-400" />
            <input value={query} onChange={(event) => setQuery(event.target.value)} className="input pl-9" placeholder="Filter this page" />
          </label>
        </div>

        <section className="card p-5">
          {loading ? (
            <EmptyState title="Loading agent detail" detail="Collecting the latest status and activity for this agent." />
          ) : activeTab === 'AI Capability' ? (
            <AbomPanel scope="device" deviceId={deviceId} embedded />
          ) : activeTab === 'Evidence Path' ? (
            <EvidenceGraphPanel
              initialDeviceId={deviceId}
              initialFindingId={data.findings[0]?.finding_id || data.findings[0]?.event_id || ''}
              embedded
              autoLoad
            />
          ) : (
            <TabContent
              activeTab={activeTab}
              device={device}
              events={filteredEvents}
              processes={filterRows(data.processes, query)}
              flows={filterRows(data.flows, query)}
              dns={filterRows(data.dns, query)}
              findings={filterRows(data.findings, query)}
              extensions={filterRows(data.extensions, query)}
              sase={filterRows(data.sase, query)}
              collectors={filterRows(data.collectors, query)}
              performance={filterRows(data.performance, query)}
              aiDns={aiDns}
              aiProcesses={aiProcesses}
              aiFindings={aiFindings}
              deviceFresh={isFresh}
            />
          )}
        </section>
      </main>
      </div>
    </ConsoleShell>
  )
}

function nextBestAction(props: {
  collectors: CollectorStatus[]
  performance: AgentPerformance[]
  findings: FindingRecord[]
  isFresh: boolean
}): string {
  const badCollector = props.collectors.some((c) => c.status && c.status !== 'healthy')
  const budget = props.performance[0]
  const queue = budget?.event_queue_depth ?? 0
  if (!props.isFresh) return 'Prioritize agent connectivity; evidence is stale.'
  if (props.findings.length) return 'Triage findings and validate scope before drafting observe-only controls.'
  if (badCollector) return 'Inspect collector health signals; restore telemetry before policy work.'
  if (queue > 500) return 'Investigate event queue pressure; confirm performance budget headroom.'
  return 'Continue monitoring; endpoint looks healthy in this window.'
}

function TabContent(props: {
  activeTab: string
  device: DeviceRecord
  events: EventRecord[]
  processes: ProcessRecord[]
  flows: FlowRecord[]
  dns: DnsRecord[]
  findings: FindingRecord[]
  extensions: BrowserExtension[]
  sase: SaseComponent[]
  collectors: CollectorStatus[]
  performance: AgentPerformance[]
  aiDns: DnsRecord[]
  aiProcesses: ProcessRecord[]
  aiFindings: FindingRecord[]
  deviceFresh: boolean
}) {
  const { activeTab } = props
  if (activeTab === 'Overview') {
    const confidence = props.findings.length > 5 ? 'Needs review' : props.findings.length ? 'Watch' : 'Stable'
    const unhealthy = props.collectors.filter((c) => c.status !== 'healthy').length
    const aiSignals = props.aiDns.length + props.aiProcesses.length + props.aiFindings.length
    return (
      <div className="grid gap-4 xl:grid-cols-3">
        <Panel icon={Activity} title="Agent status">
          <KeyValue label="Device" value={props.device.device_id} />
          <KeyValue label="Sensor" value={props.device.sensor_version || 'unknown'} />
          <KeyValue label="Telemetry" value={props.deviceFresh ? 'Reporting now' : 'Not reporting recently'} />
        </Panel>
        <Panel icon={ShieldCheck} title="Risk summary">
          <KeyValue label="Open findings" value={String(props.findings.length)} />
          <KeyValue label="State" value={confidence} />
        </Panel>
        <Panel icon={Chrome} title="AI and control signals">
          <KeyValue label="AI-related signals" value={String(aiSignals)} />
          <KeyValue label="Enterprise controls" value={String(props.sase.length)} />
        </Panel>
        <Panel icon={Globe2} title="Network activity">
          <KeyValue label="Connections" value={String(props.flows.length)} />
          <KeyValue label="Domain lookups" value={String(props.dns.length)} />
        </Panel>
        <Panel icon={Cpu} title="Agent health">
          <KeyValue label="Collector issues" value={String(unhealthy)} />
          <KeyValue label="Queue depth" value={String(props.performance[0]?.event_queue_depth ?? 'n/a')} />
        </Panel>
        <Panel icon={AlertTriangle} title="Next best action">
          <EmptyLine text={nextBestAction({
            collectors: props.collectors,
            performance: props.performance,
            findings: props.findings,
            isFresh: props.deviceFresh,
          })} />
        </Panel>
      </div>
    )
  }
  if (activeTab === 'Activity') {
    return (
      <div className="grid gap-6">
        <Panel icon={ShieldCheck} title="Findings that need attention">
          <Table rows={props.findings} empty="No findings for this agent." columns={['severity', 'title', 'classification', 'risk_score']} />
        </Panel>
        <Panel icon={Chrome} title="AI-related activity">
          <Table
            rows={[
              ...props.aiFindings.map((row) => ({ type: 'finding', signal: row.title, detail: row.classification, action: row.recommended_action })),
              ...props.aiDns.map((row) => ({ type: 'domain', signal: row.query, detail: row.answers?.join(', '), action: row.correlation_method })),
              ...props.aiProcesses.map((row) => ({ type: 'program', signal: row.name, detail: row.path || row.command_line, action: row.user })),
            ]}
            empty="No AI-related activity is visible in this window."
            columns={['type', 'signal', 'detail', 'action']}
          />
        </Panel>
        <Panel icon={Network} title="External connections">
          <Table rows={props.flows} empty="No network connections for this agent." columns={['process_name', 'remote_hostname', 'remote_ip', 'remote_port']} />
        </Panel>
        <Panel icon={Cpu} title="Observed programs">
          <Table rows={props.processes} empty="No observed program samples for this agent." columns={['name', 'user', 'path', 'command_line']} />
        </Panel>
      </div>
    )
  }
  if (activeTab === 'Software & Controls') {
    return (
      <div className="grid gap-6 lg:grid-cols-2">
        <Panel icon={Chrome} title="Browser extensions">
          <Table rows={props.extensions} empty="No browser extension inventory for this agent." columns={['browser', 'name', 'version', 'profile']} />
        </Panel>
        <Panel icon={ShieldCheck} title="Enterprise controls">
          <Table rows={props.sase} empty="No SASE/SSE control inventory for this agent." columns={['vendor', 'product', 'name', 'status']} />
        </Panel>
      </div>
    )
  }
  if (activeTab === 'Detection Packs') {
    return (
      <div className="rounded-lg border border-dashed border-slate-300 bg-slate-50 p-6 text-sm text-slate-700">
        Detection-pack rollout state is summarized in the Agents rollout table. This page shows endpoint-local findings for this agent: {props.findings.length}.
      </div>
    )
  }
  if (activeTab === 'Health') {
    const latest = props.performance[0]
    return (
      <div className="space-y-5">
        <div className="grid gap-4 md:grid-cols-4">
          <Metric label="CPU" value={formatPercent(latest?.process_cpu_percent)} />
          <Metric label="RSS" value={formatMb(latest?.process_memory_rss_mb)} />
          <Metric label="Queue" value={latest?.event_queue_depth ?? 'n/a'} />
          <Metric label="Spool" value={formatBytes(latest?.spool_bytes)} />
        </div>
        <div className="grid gap-5 lg:grid-cols-2">
          <Panel icon={Activity} title="Collector status">
            <Table rows={props.collectors} empty="No collector health telemetry." columns={['collector', 'status', 'message']} />
          </Panel>
          <Panel icon={Cpu} title="Performance budget sample">
            <Table
              rows={props.performance}
              empty="No agent performance telemetry."
              columns={['collector_name', 'collector_runtime_ms', 'process_cpu_percent', 'event_queue_depth', 'spool_bytes']}
            />
          </Panel>
        </div>
      </div>
    )
  }
  if (activeTab === 'Policy') {
    return (
      <div className="space-y-3 text-sm text-slate-700">
        <p className="font-semibold text-slate-900">Observe-only posture</p>
        <p>
          This console does not push enforcement. Use Control → Controls to work observe-only drafts derived from findings, then simulate
          blast radius before any future enforcement project.
        </p>
        <a href="/control/controls" className="font-semibold text-primary-700 hover:text-primary-900">
          Open draft controls
        </a>
      </div>
    )
  }
  return <Table rows={props.events} empty="No events for this device." columns={['event_type', 'source', 'sensor_version']} />
}

function Metric({ label, value, tone = 'normal' }: { label: string; value: string | number; tone?: 'normal' | 'warning' }) {
  return (
    <div className="card p-4">
      <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">{label}</p>
      <p className={`mt-2 text-2xl font-bold ${tone === 'warning' ? 'text-warning-700' : 'text-gray-900'}`}>{value}</p>
    </div>
  )
}

function Panel({ icon: Icon, title, children }: { icon: typeof Activity; title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-gray-200 p-4">
      <div className="mb-4 flex items-center gap-2">
        <Icon className="h-5 w-5 text-primary-600" />
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-700">{title}</h2>
      </div>
      <div className="space-y-3">{children}</div>
    </div>
  )
}

function Table<T extends Record<string, any>>({ rows, columns, empty }: { rows: T[]; columns: string[]; empty: string }) {
  if (rows.length === 0) return <EmptyState title="No telemetry" detail={empty} />
  const visibleRows = rows.slice(0, 40)
  const hiddenCount = rows.length - visibleRows.length
  return (
    <div className="overflow-hidden rounded-lg border border-slate-200">
      <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200 text-sm">
        <thead>
          <tr className="text-left text-xs font-medium uppercase tracking-wide text-gray-500">
            {columns.map((column) => <th key={column} className="py-2 pr-5">{labelize(column)}</th>)}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {visibleRows.map((row, index) => (
            <tr key={`${row.event_id || row.process_guid || row.flow_id || index}`}>
              {columns.map((column) => (
                <td key={column} className="max-w-sm px-3 py-3 align-top text-gray-700">
                  <span className="line-clamp-2 break-words" title={formatCell(row[column], false)}>{formatCell(row[column])}</span>
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      </div>
      {hiddenCount > 0 ? (
        <div className="border-t border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-500">
          Showing first {visibleRows.length} of {rows.length}. Use the page filter to narrow the list.
        </div>
      ) : null}
    </div>
  )
}

function KeyValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-gray-100 pb-2 last:border-0">
      <dt className="text-sm text-gray-500">{label}</dt>
      <dd className="text-right text-sm font-medium text-gray-900">{value}</dd>
    </div>
  )
}

function FreshBadge({ fresh }: { fresh: boolean }) {
  return fresh ? (
    <span className="badge badge-success"><CheckCircle className="mr-1 h-3 w-3" />Fresh</span>
  ) : (
    <span className="badge badge-warning"><Clock className="mr-1 h-3 w-3" />Stale</span>
  )
}

function EmptyState({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="rounded-lg border border-dashed border-gray-300 p-10 text-center">
      <Database className="mx-auto h-8 w-8 text-gray-400" />
      <h2 className="mt-3 text-sm font-semibold text-gray-900">{title}</h2>
      <p className="mt-1 text-sm text-gray-500">{detail}</p>
    </div>
  )
}

function EmptyLine({ text }: { text: string }) {
  return <p className="rounded-md border border-dashed border-gray-300 px-3 py-2 text-sm text-gray-500">{text}</p>
}

function filterRows<T>(rows: T[], query: string): T[] {
  const needle = query.trim().toLowerCase()
  if (!needle) return rows
  return rows.filter((row) => JSON.stringify(row).toLowerCase().includes(needle))
}

function latestTimestamp(events: EventRecord[]) {
  return events.reduce((latest, event) => Math.max(latest, event.received_at_ms || event.timestamp_ms || 0), 0)
}

function formatAge(value?: number) {
  if (!value) return 'unknown'
  const seconds = Math.max(0, Math.floor((Date.now() - value) / 1000))
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 48) return `${hours}h ago`
  return new Date(value).toLocaleString()
}

function formatCell(value: any, compact = true) {
  if (Array.isArray(value)) return value.join(', ')
  if (value === null || value === undefined || value === '') return 'n/a'
  if (typeof value === 'object') return JSON.stringify(value)
  const text = String(value)
  return compact && text.length > 140 ? `${text.slice(0, 137)}...` : text
}

function formatPercent(value?: number | null) {
  return typeof value === 'number' ? `${value.toFixed(1)}%` : 'n/a'
}

function formatMb(value?: number | null) {
  return typeof value === 'number' ? `${value.toFixed(1)} MB` : 'n/a'
}

function formatBytes(value?: number | null) {
  if (typeof value !== 'number') return 'n/a'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / (1024 * 1024)).toFixed(1)} MB`
}

function labelize(value: string) {
  return value.replace(/_/g, ' ')
}
