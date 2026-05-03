'use client'

import { useEffect, useMemo, useState } from 'react'
import {
  Activity,
  AlertTriangle,
  ArrowDownUp,
  Bot,
  Cpu,
  Database,
  FlaskConical,
  Globe2,
  Network,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
  Sparkles,
  TerminalSquare,
} from 'lucide-react'

type EventRecord = {
  event_id: string
  event_type: string
  timestamp_ms: number
  received_at_ms?: number
  source?: string
  device_id: string
  agent_id: string
  sequence: number
  payload: Record<string, any>
}

type ProcessRecord = {
  process_guid: string
  pid: number
  ppid?: number
  name: string
  path?: string
  command_line?: string
  event_id?: string
}

type FlowRecord = {
  flow_id: string
  pid?: number
  process_name?: string
  protocol: string
  direction: string
  local_ip: string
  local_port?: number
  remote_ip: string
  remote_port?: number
}

type DnsRecord = {
  query: string
  answers?: string[]
  resolver?: string
  correlation_method?: string
}

type FindingRecord = {
  event_type: string
  detection_id?: string
  finding_id?: string
  classification?: string
  severity?: string
  title?: string
  risk_score?: number
  process_guid?: string
  detected_patterns?: string[]
  evidence?: Array<{ type: string; value: string; confidence: number }>
}

type VisibilityData = {
  events: EventRecord[]
  processes: ProcessRecord[]
  flows: FlowRecord[]
  dns: DnsRecord[]
  findings: FindingRecord[]
}

type DeviceRecord = {
  device_id: string
  agent_id: string
  source: string
  sensor_version: string
  first_seen_ms: number
  last_seen_ms: number
  last_event_type: string
  event_count: number
  event_type_count: Record<string, number>
}

const tabs = [
  { id: 'processes', label: 'Processes', icon: TerminalSquare },
  { id: 'flows', label: 'Flows', icon: Network },
  { id: 'dns', label: 'DNS', icon: Globe2 },
  { id: 'findings', label: 'Findings', icon: AlertTriangle },
  { id: 'events', label: 'Events', icon: Database },
] as const

const wedgeIdeas = [
  {
    title: 'Observe to action in one loop',
    text: 'Most tools stop at detection. Aegis can show the path, explain the blast radius, draft the control, and stage rollback.',
  },
  {
    title: 'Agentic AI governance',
    text: 'Detect AI toolchains, browser AI usage, shell automation, and model gateways as first-class endpoint activity, not generic malware.',
  },
  {
    title: 'Policy with evidence',
    text: 'Every policy recommendation should cite process, flow, DNS, identity, and finding evidence before anything is enforced.',
  },
  {
    title: 'Lab-fast, enterprise-hard',
    text: 'Start with portable collectors and tunnels, then graduate the same event contract to signed agents, mTLS, and kernel-grade controls.',
  },
]

export default function VisibilityConsole() {
  const [devices, setDevices] = useState<DeviceRecord[]>([])
  const [selectedDevice, setSelectedDevice] = useState('')
  const [activeTab, setActiveTab] = useState<(typeof tabs)[number]['id']>('processes')
  const [query, setQuery] = useState('')
  const [data, setData] = useState<VisibilityData>({
    events: [],
    processes: [],
    flows: [],
    dns: [],
    findings: [],
  })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null)

  useEffect(() => {
    fetchDevices()
  }, [])

  useEffect(() => {
    if (!selectedDevice) return
    fetchVisibility()
    const interval = setInterval(fetchVisibility, 15000)
    return () => clearInterval(interval)
  }, [selectedDevice])

  async function fetchJson<T>(path: string): Promise<T> {
    const response = await fetch(path, { cache: 'no-store' })
    if (!response.ok) {
      throw new Error(`${path} returned HTTP ${response.status}`)
    }
    return response.json()
  }

  async function fetchDevices() {
    try {
      setLoading(true)
      setError(null)
      const response = await fetchJson<{ devices?: DeviceRecord[] }>('/api/visibility/devices?limit=50')
      const nextDevices = response.devices || []
      setDevices(nextDevices)
      if (!selectedDevice && nextDevices.length > 0) {
        setSelectedDevice(nextDevices[0].device_id)
      }
      if (selectedDevice && !nextDevices.some((device) => device.device_id === selectedDevice) && nextDevices.length > 0) {
        setSelectedDevice(nextDevices[0].device_id)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load devices')
    } finally {
      setLoading(false)
    }
  }

  async function fetchVisibility() {
    if (!selectedDevice) return

    try {
      setLoading(true)
      setError(null)

      const params = `device_id=${encodeURIComponent(selectedDevice)}&limit=120`
      const [events, processes, flows, dns, findings] = await Promise.all([
        fetchJson<{ events?: EventRecord[] }>(`/api/visibility/events?${params}`),
        fetchJson<{ processes?: ProcessRecord[] }>(`/api/visibility/processes?${params}`),
        fetchJson<{ flows?: FlowRecord[] }>(`/api/visibility/flows?${params}`),
        fetchJson<{ dns?: DnsRecord[] }>(`/api/visibility/dns?${params}`),
        fetchJson<{ findings?: FindingRecord[] }>(`/api/visibility/findings?${params}`),
      ])

      setData({
        events: events.events || [],
        processes: processes.processes || [],
        flows: flows.flows || [],
        dns: dns.dns || [],
        findings: findings.findings || [],
      })
      setLastRefresh(new Date())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load visibility data')
    } finally {
      setLoading(false)
    }
  }

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase()
    if (!needle) return data

    const includesNeedle = (value: unknown) =>
      JSON.stringify(value || '').toLowerCase().includes(needle)

    return {
      events: data.events.filter(includesNeedle),
      processes: data.processes.filter(includesNeedle),
      flows: data.flows.filter(includesNeedle),
      dns: data.dns.filter(includesNeedle),
      findings: data.findings.filter(includesNeedle),
    }
  }, [data, query])

  const stats = useMemo(() => {
    const latest = data.events[0]
    const risk = data.findings.reduce((max, finding) => Math.max(max, finding.risk_score || 0), 0)
    const aiSignals = data.findings.filter((finding) =>
      `${finding.classification || ''} ${finding.title || ''}`.toLowerCase().includes('agent')
    ).length

    return {
      events: data.events.length,
      processes: data.processes.length,
      flows: data.flows.length,
      dns: data.dns.length,
      findings: data.findings.length,
      risk,
      aiSignals,
      latestAge: latest ? ageFromMs(latest.received_at_ms || latest.timestamp_ms) : 'no data',
    }
  }, [data])

  const selectedDeviceRecord = devices.find((device) => device.device_id === selectedDevice)
  const platform = platformName(selectedDeviceRecord?.source || selectedDevice)

  return (
    <div className="min-h-screen bg-slate-50 text-slate-950">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-[1440px] flex-col gap-4 px-5 py-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-md bg-slate-950">
              <ShieldCheck className="h-6 w-6 text-white" />
            </div>
            <div>
              <h1 className="text-xl font-semibold text-slate-950">Aegis Visibility</h1>
              <p className="text-sm text-slate-500">Endpoint evidence, AI-agent signals, and control readiness</p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {devices.length === 0 ? (
              <span className="rounded-md border border-slate-300 bg-slate-50 px-3 py-2 text-sm text-slate-500">
                No reporting devices
              </span>
            ) : devices.map((device) => (
              <button
                key={device.device_id}
                onClick={() => setSelectedDevice(device.device_id)}
                className={`rounded-md border px-3 py-2 text-sm font-medium ${
                  selectedDevice === device.device_id
                    ? 'border-slate-950 bg-slate-950 text-white'
                    : 'border-slate-300 bg-white text-slate-700 hover:bg-slate-100'
                }`}
              >
                {platformName(device.source || device.device_id)}
              </button>
            ))}
            <button
              onClick={fetchVisibility}
              className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-[1440px] px-5 py-5">
        {error && (
          <div className="mb-4 flex items-center gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            <AlertTriangle className="h-4 w-4" />
            {error}
          </div>
        )}

        <section className="mb-5 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-7">
          <Metric icon={Server} label="Device" value={platform} detail={selectedDevice || 'waiting for telemetry'} />
          <Metric icon={Activity} label="Latest" value={stats.latestAge} detail={lastRefresh ? `UI ${lastRefresh.toLocaleTimeString()}` : 'waiting'} />
          <Metric icon={TerminalSquare} label="Processes" value={stats.processes.toString()} detail="snapshot rows" />
          <Metric icon={ArrowDownUp} label="Flows" value={stats.flows.toString()} detail="socket evidence" />
          <Metric icon={Globe2} label="DNS" value={stats.dns.toString()} detail="resolver observations" />
          <Metric icon={Bot} label="AI Signals" value={stats.aiSignals.toString()} detail={`${stats.findings} findings`} />
          <Metric icon={AlertTriangle} label="Max Risk" value={stats.risk.toString()} detail="non-blocking" />
        </section>

        <section className="mb-5 grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
          <div className="rounded-lg border border-slate-200 bg-white">
            <div className="flex flex-col gap-3 border-b border-slate-200 px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <h2 className="text-base font-semibold text-slate-950">Live Endpoint Evidence</h2>
                <p className="text-sm text-slate-500">Filtered by device. Auto-refreshes every 15 seconds.</p>
              </div>
              <div className="relative min-w-0 lg:w-80">
                <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                <input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  className="h-9 w-full rounded-md border border-slate-300 bg-white pl-9 pr-3 text-sm outline-none focus:border-slate-600"
                  placeholder="Filter process, IP, DNS, finding"
                />
              </div>
            </div>

            <div className="flex flex-wrap gap-2 border-b border-slate-200 px-4 py-3">
              {tabs.map((tab) => {
                const Icon = tab.icon
                const count = filtered[tab.id].length
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`inline-flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium ${
                      activeTab === tab.id
                        ? 'bg-slate-950 text-white'
                        : 'bg-slate-100 text-slate-700 hover:bg-slate-200'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {tab.label}
                    <span className={activeTab === tab.id ? 'text-slate-300' : 'text-slate-500'}>{count}</span>
                  </button>
                )
              })}
            </div>

            <div className="max-h-[620px] overflow-auto">
              {activeTab === 'processes' && <ProcessTable processes={filtered.processes} />}
              {activeTab === 'flows' && <FlowTable flows={filtered.flows} />}
              {activeTab === 'dns' && <DnsTable dns={filtered.dns} />}
              {activeTab === 'findings' && <FindingTable findings={filtered.findings} />}
              {activeTab === 'events' && <EventTable events={filtered.events} />}
            </div>
          </div>

          <aside className="space-y-4">
            <div className="rounded-lg border border-slate-200 bg-white p-4">
              <div className="mb-3 flex items-center gap-2">
                <Sparkles className="h-5 w-5 text-slate-800" />
                <h2 className="text-base font-semibold">How Aegis Wins</h2>
              </div>
              <div className="space-y-3">
                {wedgeIdeas.map((idea) => (
                  <div key={idea.title} className="border-l-2 border-slate-300 pl-3">
                    <div className="text-sm font-semibold text-slate-950">{idea.title}</div>
                    <p className="mt-1 text-sm leading-5 text-slate-600">{idea.text}</p>
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-lg border border-slate-200 bg-white p-4">
              <div className="mb-3 flex items-center gap-2">
                <FlaskConical className="h-5 w-5 text-slate-800" />
                <h2 className="text-base font-semibold">Innovation Bets</h2>
              </div>
              <div className="space-y-2 text-sm text-slate-700">
                <p><strong>Agent bill of materials:</strong> map every AI tool, model endpoint, script runner, and browser AI session per host.</p>
                <p><strong>Evidence-backed enforcement:</strong> generate controls only from observed process-to-flow paths, then simulate blast radius before rollout.</p>
                <p><strong>Autonomous change windows:</strong> let Aegis propose, stage, watch, and roll back controls with human approval gates.</p>
                <p><strong>Identity-aware endpoint graph:</strong> merge endpoint telemetry with network path and user intent to explain why traffic exists.</p>
              </div>
            </div>

            <div className="rounded-lg border border-slate-200 bg-white p-4">
              <div className="mb-3 flex items-center gap-2">
                <Cpu className="h-5 w-5 text-slate-800" />
                <h2 className="text-base font-semibold">Next Build Slice</h2>
              </div>
              <ol className="space-y-2 text-sm text-slate-700">
                <li>1. Add investigation drill-in: process to flows to DNS to finding.</li>
                <li>2. Add draft control actions from a finding, still observe-only.</li>
                <li>3. Add device health scoring from event freshness and collector status.</li>
              </ol>
            </div>
          </aside>
        </section>
      </main>
    </div>
  )
}

function Metric({
  icon: Icon,
  label,
  value,
  detail,
}: {
  icon: typeof Activity
  label: string
  value: string
  detail: string
}) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</div>
          <div className="mt-2 text-2xl font-semibold text-slate-950">{value}</div>
          <div className="mt-1 truncate text-xs text-slate-500">{detail}</div>
        </div>
        <Icon className="h-5 w-5 text-slate-500" />
      </div>
    </div>
  )
}

function ProcessTable({ processes }: { processes: ProcessRecord[] }) {
  return (
    <Table empty="No process evidence for this filter.">
      <thead className="bg-slate-50">
        <tr>
          <Th>PID</Th>
          <Th>Name</Th>
          <Th>Path</Th>
          <Th>Command</Th>
        </tr>
      </thead>
      <tbody className="divide-y divide-slate-100 bg-white">
        {processes.length === 0 ? (
          <EmptyRow colSpan={4} message="No process evidence for this filter." />
        ) : processes.slice(0, 80).map((process) => (
          <tr key={process.process_guid || `${process.pid}-${process.name}`} className="hover:bg-slate-50">
            <Td mono>{process.pid}</Td>
            <Td>{process.name}</Td>
            <Td muted>{process.path || 'unknown'}</Td>
            <Td muted>{process.command_line || 'not collected'}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function FlowTable({ flows }: { flows: FlowRecord[] }) {
  return (
    <Table empty="No flow evidence for this filter.">
      <thead className="bg-slate-50">
        <tr>
          <Th>Process</Th>
          <Th>Direction</Th>
          <Th>Local</Th>
          <Th>Remote</Th>
          <Th>Protocol</Th>
        </tr>
      </thead>
      <tbody className="divide-y divide-slate-100 bg-white">
        {flows.length === 0 ? (
          <EmptyRow colSpan={5} message="No flow evidence for this filter." />
        ) : flows.slice(0, 80).map((flow) => (
          <tr key={flow.flow_id || `${flow.local_ip}-${flow.remote_ip}-${flow.remote_port}`} className="hover:bg-slate-50">
            <Td>{flow.process_name || (flow.pid ? `pid ${flow.pid}` : 'unknown')}</Td>
            <Td><Badge value={flow.direction} /></Td>
            <Td mono>{socket(flow.local_ip, flow.local_port)}</Td>
            <Td mono>{socket(flow.remote_ip, flow.remote_port)}</Td>
            <Td mono>{flow.protocol}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function DnsTable({ dns }: { dns: DnsRecord[] }) {
  return (
    <Table empty="No DNS evidence for this filter.">
      <thead className="bg-slate-50">
        <tr>
          <Th>Query</Th>
          <Th>Answers</Th>
          <Th>Resolver</Th>
          <Th>Method</Th>
        </tr>
      </thead>
      <tbody className="divide-y divide-slate-100 bg-white">
        {dns.length === 0 ? (
          <EmptyRow colSpan={4} message="No DNS evidence for this filter." />
        ) : dns.slice(0, 80).map((record, index) => (
          <tr key={`${record.query}-${index}`} className="hover:bg-slate-50">
            <Td>{record.query}</Td>
            <Td muted>{(record.answers || []).join(', ') || 'none'}</Td>
            <Td mono>{record.resolver || 'unknown'}</Td>
            <Td muted>{record.correlation_method || 'unknown'}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function FindingTable({ findings }: { findings: FindingRecord[] }) {
  return (
    <Table empty="No findings for this filter.">
      <thead className="bg-slate-50">
        <tr>
          <Th>Finding</Th>
          <Th>Risk</Th>
          <Th>Pattern</Th>
          <Th>Evidence</Th>
        </tr>
      </thead>
      <tbody className="divide-y divide-slate-100 bg-white">
        {findings.length === 0 ? (
          <EmptyRow colSpan={4} message="No findings for this filter." />
        ) : findings.slice(0, 80).map((finding, index) => (
          <tr key={finding.finding_id || finding.detection_id || index} className="hover:bg-slate-50">
            <Td>
              <div className="font-medium text-slate-900">{finding.title || finding.classification || finding.event_type}</div>
              <div className="mt-1 text-xs text-slate-500">{finding.detection_id || finding.finding_id || 'no id'}</div>
            </Td>
            <Td><Risk value={finding.risk_score || 0} severity={finding.severity} /></Td>
            <Td muted>{(finding.detected_patterns || []).join(', ') || 'none'}</Td>
            <Td muted>{(finding.evidence || []).slice(0, 2).map((item) => `${item.type}: ${item.value}`).join(' | ') || 'none'}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function EventTable({ events }: { events: EventRecord[] }) {
  return (
    <Table empty="No events for this filter.">
      <thead className="bg-slate-50">
        <tr>
          <Th>Sequence</Th>
          <Th>Type</Th>
          <Th>Source</Th>
          <Th>Received</Th>
        </tr>
      </thead>
      <tbody className="divide-y divide-slate-100 bg-white">
        {events.length === 0 ? (
          <EmptyRow colSpan={4} message="No events for this filter." />
        ) : events.slice(0, 80).map((event) => (
          <tr key={event.event_id} className="hover:bg-slate-50">
            <Td mono>{event.sequence}</Td>
            <Td>{event.event_type}</Td>
            <Td muted>{event.source || 'unknown'}</Td>
            <Td muted>{formatDate(event.received_at_ms || event.timestamp_ms)}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function Table({ children }: { children: React.ReactNode; empty: string }) {
  return (
    <table className="min-w-full table-fixed text-left text-sm">
      {children}
    </table>
  )
}

function Th({ children }: { children: React.ReactNode }) {
  return <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">{children}</th>
}

function Td({ children, muted, mono }: { children: React.ReactNode; muted?: boolean; mono?: boolean }) {
  return (
    <td className={`max-w-[360px] truncate px-4 py-3 align-top ${muted ? 'text-slate-500' : 'text-slate-800'} ${mono ? 'font-mono text-xs' : ''}`}>
      {children}
    </td>
  )
}

function EmptyRow({ colSpan, message }: { colSpan: number; message: string }) {
  return (
    <tr>
      <td colSpan={colSpan} className="px-4 py-10 text-center text-sm text-slate-500">
        {message}
      </td>
    </tr>
  )
}

function Badge({ value }: { value: string }) {
  return <span className="rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700">{value}</span>
}

function Risk({ value, severity }: { value: number; severity?: string }) {
  const tone = value >= 70 ? 'bg-red-100 text-red-800' : value >= 40 ? 'bg-amber-100 text-amber-800' : 'bg-slate-100 text-slate-700'
  return <span className={`rounded-md px-2 py-1 text-xs font-semibold ${tone}`}>{severity || 'info'} {value}</span>
}

function socket(ip?: string, port?: number) {
  return `${ip || 'unknown'}${port ? `:${port}` : ''}`
}

function formatDate(ms?: number) {
  if (!ms) return 'unknown'
  return new Date(ms).toLocaleString()
}

function ageFromMs(ms?: number) {
  if (!ms) return 'no data'
  const seconds = Math.max(0, Math.round((Date.now() - ms) / 1000))
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  return `${Math.round(minutes / 60)}h ago`
}

function platformName(value: string) {
  const normalized = value.toLowerCase()
  if (normalized.includes('windows')) return 'Windows'
  if (normalized.includes('linux')) return 'Linux'
  if (normalized.includes('macos') || normalized.includes('darwin')) return 'macOS'
  return value || 'Unknown'
}
