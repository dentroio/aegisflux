'use client'

import { useEffect, useMemo, useState } from 'react'
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
}

const tabs = [
  'Overview',
  'AI Activity',
  'Processes',
  'Network',
  'DNS',
  'Browser',
  'Inventory',
  'Findings',
  'Collector Health',
]

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
  const deviceId = decodeURIComponent(params.device_id)
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
  })

  useEffect(() => {
    loadDevice()
  }, [deviceId])

  const loadDevice = async () => {
    setRefreshing(true)
    const encoded = encodeURIComponent(deviceId)
    const [devices, events, processes, flows, dns, findings, extensions, sase, collectors] = await Promise.all([
      fetchJson<{ devices?: DeviceRecord[] }>(`/api/visibility/devices?limit=200`, {}),
      fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?device_id=${encoded}&limit=260`, {}),
      fetchJson<{ processes?: ProcessRecord[] }>(`/api/visibility/processes?device_id=${encoded}&limit=220`, {}),
      fetchJson<{ flows?: FlowRecord[] }>(`/api/visibility/flows?device_id=${encoded}&limit=220`, {}),
      fetchJson<{ observations?: DnsRecord[]; dns?: DnsRecord[] }>(`/api/visibility/dns?device_id=${encoded}&limit=220`, {}),
      fetchJson<{ findings?: FindingRecord[] }>(`/api/visibility/findings?device_id=${encoded}&limit=160`, {}),
      fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?device_id=${encoded}&event_type=aegis.browser_extension.observed&limit=160`, {}),
      fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?device_id=${encoded}&event_type=aegis.sase_component.observed&limit=160`, {}),
      fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?device_id=${encoded}&event_type=aegis.collector.status&limit=160`, {}),
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

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto max-w-7xl px-4 py-5 sm:px-6 lg:px-8">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex min-w-0 items-center gap-4">
              <a href="/agents" className="btn btn-secondary h-9 px-3">
                <ArrowLeft className="mr-2 h-4 w-4" />
                Agents
              </a>
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <Server className="h-6 w-6 text-primary-600" />
                  <h1 className="truncate text-2xl font-bold text-gray-900">{device.device_id}</h1>
                  <FreshBadge fresh={isFresh} />
                </div>
                <p className="mt-1 text-sm text-gray-500">
                  {device.source || 'visibility'} | {device.sensor_version || 'unknown'} | last seen {formatAge(device.last_seen_ms)}
                </p>
              </div>
            </div>
            <button onClick={loadDevice} disabled={refreshing} className="btn btn-secondary h-9 px-3">
              <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div className="mb-6 grid gap-4 md:grid-cols-5">
          <Metric label="Events" value={device.event_count || data.events.length} />
          <Metric label="Processes" value={data.processes.length} />
          <Metric label="Network Flows" value={data.flows.length} />
          <Metric label="DNS" value={data.dns.length} />
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
            <input value={query} onChange={(event) => setQuery(event.target.value)} className="input pl-9" placeholder="Filter current evidence" />
          </label>
        </div>

        <section className="card p-5">
          {loading ? (
            <EmptyState title="Loading device evidence" detail="Collecting visibility records from ingest." />
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
              aiDns={aiDns}
              aiProcesses={aiProcesses}
              aiFindings={aiFindings}
            />
          )}
        </section>
      </main>
    </div>
  )
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
  aiDns: DnsRecord[]
  aiProcesses: ProcessRecord[]
  aiFindings: FindingRecord[]
}) {
  const { activeTab } = props
  if (activeTab === 'Overview') {
    return (
      <div className="grid gap-5 lg:grid-cols-2">
        <Panel icon={Activity} title="Telemetry Summary">
          <KeyValue label="Agent ID" value={props.device.agent_id || 'unknown'} />
          <KeyValue label="Source" value={props.device.source || 'unknown'} />
          <KeyValue label="Sensor Version" value={props.device.sensor_version || 'unknown'} />
          <KeyValue label="Last Event" value={props.device.last_event_type || 'unknown'} />
        </Panel>
        <Panel icon={ShieldCheck} title="Evidence Counts">
          {Object.entries(props.device.event_type_count || {}).slice(0, 8).map(([type, count]) => (
            <KeyValue key={type} label={type} value={String(count)} />
          ))}
          {Object.keys(props.device.event_type_count || {}).length === 0 && <EmptyLine text="No event counts reported for this device." />}
        </Panel>
      </div>
    )
  }
  if (activeTab === 'AI Activity') {
    return (
      <div className="grid gap-5 lg:grid-cols-3">
        <ListPanel title="AI Findings" rows={props.aiFindings} empty="No AI findings in the current window." render={(row) => <EvidenceRow title={row.title || row.classification || 'AI finding'} detail={`${row.severity || 'info'} | risk ${row.risk_score || 0}`} />} />
        <ListPanel title="AI DNS" rows={props.aiDns} empty="No AI destinations in DNS evidence." render={(row) => <EvidenceRow title={row.query || 'dns'} detail={(row.answers || []).join(', ') || row.correlation_method || 'dns'} />} />
        <ListPanel title="AI Processes" rows={props.aiProcesses} empty="No MCP/model/tooling process signals." render={(row) => <EvidenceRow title={row.name || 'process'} detail={row.command_line || row.path || `pid ${row.pid || 'unknown'}`} />} />
      </div>
    )
  }
  if (activeTab === 'Processes') return <Table rows={props.processes} empty="No process telemetry for this device." columns={['pid', 'name', 'path', 'command_line', 'user']} />
  if (activeTab === 'Network') return <Table rows={props.flows} empty="No network flow telemetry for this device." columns={['process_name', 'protocol', 'direction', 'remote_ip', 'remote_port', 'remote_hostname']} />
  if (activeTab === 'DNS') return <Table rows={props.dns} empty="No DNS telemetry for this device." columns={['query', 'query_type', 'answers', 'resolver', 'correlation_method']} />
  if (activeTab === 'Browser') return <Table rows={props.extensions} empty="No browser extension telemetry for this device." columns={['browser', 'profile', 'extension_id', 'name', 'version']} />
  if (activeTab === 'Inventory') return <Table rows={props.sase} empty="No enterprise control inventory for this device." columns={['component_type', 'vendor', 'product', 'name', 'status']} />
  if (activeTab === 'Findings') return <Table rows={props.findings} empty="No findings for this device." columns={['severity', 'title', 'classification', 'risk_score', 'recommended_action']} />
  if (activeTab === 'Collector Health') return <Table rows={props.collectors} empty="No collector health telemetry for this device." columns={['collector', 'status', 'message']} />
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

function ListPanel<T>({ title, rows, empty, render }: { title: string; rows: T[]; empty: string; render: (row: T) => React.ReactNode }) {
  return (
    <div className="rounded-lg border border-gray-200 p-4">
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-gray-700">{title}</h2>
      {rows.length === 0 ? <EmptyLine text={empty} /> : <div className="space-y-2">{rows.slice(0, 12).map((row, index) => <div key={index}>{render(row)}</div>)}</div>}
    </div>
  )
}

function EvidenceRow({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="rounded-md border border-gray-200 px-3 py-2">
      <p className="truncate text-sm font-medium text-gray-900">{title}</p>
      <p className="mt-1 line-clamp-2 text-xs text-gray-500">{detail}</p>
    </div>
  )
}

function Table<T extends Record<string, any>>({ rows, columns, empty }: { rows: T[]; columns: string[]; empty: string }) {
  if (rows.length === 0) return <EmptyState title="No telemetry" detail={empty} />
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200 text-sm">
        <thead>
          <tr className="text-left text-xs font-medium uppercase tracking-wide text-gray-500">
            {columns.map((column) => <th key={column} className="py-2 pr-5">{labelize(column)}</th>)}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {rows.slice(0, 120).map((row, index) => (
            <tr key={`${row.event_id || row.process_guid || row.flow_id || index}`}>
              {columns.map((column) => <td key={column} className="max-w-md py-3 pr-5 align-top text-gray-700">{formatCell(row[column])}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
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

function formatCell(value: any) {
  if (Array.isArray(value)) return value.join(', ')
  if (value === null || value === undefined || value === '') return 'n/a'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function labelize(value: string) {
  return value.replace(/_/g, ' ')
}
