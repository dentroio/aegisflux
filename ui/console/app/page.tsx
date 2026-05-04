'use client'

import { useEffect, useMemo, useState } from 'react'
import {
  Activity,
  AlertTriangle,
  ArrowDownUp,
  Bot,
  Box,
  Chrome,
  CheckCircle2,
  Cpu,
  Database,
  FileText,
  Globe2,
  HardDrive,
  LayoutDashboard,
  LockKeyhole,
  Network,
  Plug,
  RefreshCw,
  Search,
  Server,
  Settings,
  Shield,
  ShieldCheck,
  SlidersHorizontal,
  TerminalSquare,
  Undo2,
} from 'lucide-react'

type EventRecord = {
  event_id: string
  event_type: string
  timestamp_ms: number
  received_at_ms?: number
  source?: string
  device_id: string
  agent_id: string
  sensor_version?: string
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
  remote_hostname?: string
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

type BrowserExtensionRecord = {
  browser: string
  profile: string
  extension_id: string
  name: string
  version: string
  permissions?: string[]
  host_permissions?: string[]
  collection_method?: string
}

type SaseComponentRecord = {
  component_type: string
  vendor: string
  product: string
  name: string
  version?: string | null
  status?: string | null
  source: string
  evidence?: string[]
  collection_method?: string
}

type DraftControl = {
  control_id: string
  title: string
  mode: string
  status: string
  action: string
  target: string
  scope: string
  reason: string
  evidence: string[]
  blast_radius: string[]
  rollback: string[]
}

type InvestigationData = {
  ok: boolean
  device_id: string
  counts: {
    processes: number
    flows: number
    dns: number
    findings: number
    draft_controls?: number
  }
  processes: ProcessRecord[]
  flows: FlowRecord[]
  dns: DnsRecord[]
  findings: FindingRecord[]
  draft_controls?: DraftControl[]
}

type InvestigationSelection = {
  label: string
  process_guid?: string
  pid?: number
}

type VisibilityData = {
  events: EventRecord[]
  processes: ProcessRecord[]
  flows: FlowRecord[]
  dns: DnsRecord[]
  findings: FindingRecord[]
}

const navGroups = [
  {
    label: 'Dashboard',
    items: [{ id: 'overview', label: 'Overview', icon: LayoutDashboard }],
  },
  {
    label: 'Analyze',
    items: [
      { id: 'activity', label: 'AI Activity', icon: Bot },
      { id: 'evidence', label: 'Evidence', icon: Database },
      { id: 'inventory', label: 'Inventory', icon: Box },
      { id: 'findings', label: 'Findings', icon: AlertTriangle },
    ],
  },
  {
    label: 'Operate',
    items: [
      { id: 'controls', label: 'Controls', icon: Shield },
      { id: 'devices', label: 'Devices', icon: Server },
    ],
  },
  {
    label: 'Configure',
    items: [
      { id: 'connectors', label: 'Connectors', icon: Plug },
      { id: 'management', label: 'Management', icon: Settings },
    ],
  },
] as const

const evidenceTabs = [
  { id: 'processes', label: 'Processes', icon: TerminalSquare },
  { id: 'flows', label: 'Flows', icon: Network },
  { id: 'dns', label: 'DNS', icon: Globe2 },
  { id: 'events', label: 'Events', icon: Database },
] as const

export default function VisibilityConsole() {
  const [devices, setDevices] = useState<DeviceRecord[]>([])
  const [selectedDevice, setSelectedDevice] = useState('')
  const [activeSection, setActiveSection] = useState('overview')
  const [activeEvidenceTab, setActiveEvidenceTab] = useState<(typeof evidenceTabs)[number]['id']>('processes')
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
  const [investigation, setInvestigation] = useState<InvestigationData | null>(null)
  const [investigationSelection, setInvestigationSelection] = useState<InvestigationSelection | null>(null)
  const [investigationLoading, setInvestigationLoading] = useState(false)

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
    if (!response.ok) throw new Error(`${path} returned HTTP ${response.status}`)
    return response.json()
  }

  async function fetchDevices() {
    try {
      setLoading(true)
      setError(null)
      const response = await fetchJson<{ devices?: DeviceRecord[] }>('/api/visibility/devices?limit=50')
      const nextDevices = response.devices || []
      setDevices(nextDevices)
      if (!selectedDevice && nextDevices.length > 0) setSelectedDevice(nextDevices[0].device_id)
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
      const params = `device_id=${encodeURIComponent(selectedDevice)}&limit=160`
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

  async function fetchInvestigation(selection: InvestigationSelection) {
    if (!selectedDevice) return
    try {
      setInvestigationLoading(true)
      setInvestigationSelection(selection)
      const params = new URLSearchParams({ device_id: selectedDevice, limit: '20' })
      if (selection.process_guid) params.set('process_guid', selection.process_guid)
      if (selection.pid !== undefined) params.set('pid', selection.pid.toString())
      const response = await fetchJson<InvestigationData>(`/api/visibility/investigation?${params.toString()}`)
      setInvestigation(response)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load investigation path')
    } finally {
      setInvestigationLoading(false)
    }
  }

  const derived = useMemo(() => deriveVisibility(data), [data])
  const filtered = useMemo(() => filterVisibility(data, derived.extensions, derived.saseComponents, query), [data, derived, query])
  const selectedDeviceRecord = devices.find((device) => device.device_id === selectedDevice)
  const stats = useMemo(() => buildStats(data, derived, selectedDeviceRecord), [data, derived, selectedDeviceRecord])

  return (
    <div className="flex min-h-screen bg-gray-50 text-slate-900">
      <aside className="hidden w-72 shrink-0 border-r border-gray-200 bg-white lg:flex lg:flex-col">
        <div className="flex h-16 items-center gap-3 border-b border-gray-200 px-5">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-950">
            <ShieldCheck className="h-5 w-5 text-white" />
          </div>
          <div>
            <div className="text-base font-semibold text-slate-950">Aegis</div>
            <div className="text-xs text-slate-500">AI endpoint governance</div>
          </div>
        </div>

        <nav className="flex-1 overflow-y-auto px-3 py-4">
          {navGroups.map((group) => (
            <div key={group.label} className="mb-4">
              <div className="px-3 pb-1 text-xs font-semibold uppercase tracking-wide text-slate-400">{group.label}</div>
              <div className="space-y-1">
                {group.items.map((item) => {
                  const Icon = item.icon
                  const active = activeSection === item.id
                  return (
                    <button
                      key={item.id}
                      onClick={() => setActiveSection(item.id)}
                      className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                        active ? 'bg-slate-950 text-white' : 'text-slate-700 hover:bg-gray-100'
                      }`}
                    >
                      <Icon className="h-4 w-4" />
                      {item.label}
                    </button>
                  )
                })}
              </div>
            </div>
          ))}
        </nav>

        <div className="border-t border-gray-200 p-4">
          <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-3">
            <div className="flex items-center gap-2 text-sm font-semibold text-emerald-800">
              <CheckCircle2 className="h-4 w-4" />
              Observe-only
            </div>
            <p className="mt-1 text-xs leading-5 text-emerald-700">Controls stay staged until approval and rollback are wired.</p>
          </div>
        </div>
      </aside>

      <div className="min-w-0 flex-1">
        <header className="border-b border-gray-200 bg-white">
          <div className="flex flex-col gap-4 px-5 py-4 xl:flex-row xl:items-center xl:justify-between">
            <div>
              <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
                <ShieldCheck className="h-4 w-4" />
                Aegis Management Console
              </div>
              <h1 className="mt-1 text-2xl font-semibold text-slate-950">{sectionTitle(activeSection)}</h1>
              <p className="mt-1 text-sm text-slate-500">{sectionSubtitle(activeSection)}</p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <select
                value={selectedDevice}
                onChange={(event) => setSelectedDevice(event.target.value)}
                className="h-10 rounded-md border border-gray-300 bg-white px-3 text-sm font-medium text-slate-700 outline-none focus:border-slate-600"
              >
                {devices.length === 0 ? (
                  <option value="">No reporting devices</option>
                ) : devices.map((device) => (
                  <option key={device.device_id} value={device.device_id}>
                    {platformName(device.source || device.device_id)} · {device.device_id}
                  </option>
                ))}
              </select>
              <button
                onClick={fetchVisibility}
                className="inline-flex h-10 items-center gap-2 rounded-md border border-gray-300 bg-white px-3 text-sm font-medium text-slate-700 hover:bg-gray-50"
              >
                <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            </div>
          </div>
        </header>

        <main className="px-5 py-5">
          {error && (
            <div className="mb-4 flex items-center gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
              <AlertTriangle className="h-4 w-4" />
              {error}
            </div>
          )}

          <div className="mb-5 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
            <KpiCard icon={Server} label="Device" value={stats.deviceLabel} detail={selectedDevice || 'waiting for telemetry'} />
            <KpiCard icon={Activity} label="Latest" value={stats.latestAge} detail={lastRefresh ? `UI ${lastRefresh.toLocaleTimeString()}` : 'waiting'} />
            <KpiCard icon={Bot} label="AI Signals" value={stats.aiSignals.toString()} detail={`${data.findings.length} findings`} />
            <KpiCard icon={Chrome} label="Extensions" value={derived.extensions.length.toString()} detail="browser inventory" />
            <KpiCard icon={LockKeyhole} label="SSE/SASE" value={derived.saseComponents.length.toString()} detail="endpoint controls" />
            <KpiCard icon={AlertTriangle} label="Max Risk" value={stats.risk.toString()} detail="observe-only" />
          </div>

          <div className="mb-5 grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_390px]">
            <section className="min-w-0">
              {activeSection === 'overview' && (
                <OverviewPanel stats={stats} derived={derived} data={data} setSection={setActiveSection} />
              )}
              {activeSection === 'activity' && (
                <ActivityPanel findings={filtered.findings} dns={filtered.dns} flows={filtered.flows} onInvestigate={fetchInvestigation} />
              )}
              {activeSection === 'evidence' && (
                <EvidencePanel
                  query={query}
                  setQuery={setQuery}
                  activeTab={activeEvidenceTab}
                  setActiveTab={setActiveEvidenceTab}
                  filtered={filtered}
                  onInvestigate={fetchInvestigation}
                />
              )}
              {activeSection === 'inventory' && (
                <InventoryPanel extensions={filtered.extensions} saseComponents={filtered.saseComponents} />
              )}
              {activeSection === 'findings' && (
                <Panel title="Findings" subtitle="Explainable detections and risk records.">
                  <FindingTable findings={filtered.findings} onInvestigate={fetchInvestigation} />
                </Panel>
              )}
              {activeSection === 'controls' && (
                <ControlsPanel investigation={investigation} />
              )}
              {activeSection === 'devices' && (
                <DevicesPanel devices={devices} selectedDevice={selectedDevice} selectDevice={setSelectedDevice} collectorStatuses={derived.collectorStatuses} />
              )}
              {activeSection === 'connectors' && (
                <ConnectorsPanel saseComponents={derived.saseComponents} />
              )}
              {activeSection === 'management' && (
                <ManagementPanel collectorStatuses={derived.collectorStatuses} />
              )}
            </section>

            <aside className="space-y-4">
              <InvestigationPanel loading={investigationLoading} selection={investigationSelection} investigation={investigation} />
              <AgentBudgetPanel />
              <ReadinessPanel collectorStatuses={derived.collectorStatuses} />
            </aside>
          </div>
        </main>
      </div>
    </div>
  )
}

function deriveVisibility(data: VisibilityData) {
  const extensions = data.events
    .filter((event) => event.event_type === 'aegis.browser_extension.observed')
    .map((event) => event.payload as BrowserExtensionRecord)

  const saseComponents = data.events
    .filter((event) => event.event_type === 'aegis.sase_component.observed')
    .map((event) => event.payload as SaseComponentRecord)

  const collectorStatuses = data.events
    .filter((event) => event.event_type === 'aegis.collector.status')
    .map((event) => ({
      collector: String(event.payload.collector || 'unknown'),
      status: String(event.payload.status || 'unknown'),
      message: String(event.payload.message || ''),
      received_at_ms: event.received_at_ms || event.timestamp_ms,
    }))

  return { extensions, saseComponents, collectorStatuses }
}

function filterVisibility(data: VisibilityData, extensions: BrowserExtensionRecord[], saseComponents: SaseComponentRecord[], query: string) {
  const needle = query.trim().toLowerCase()
  const includesNeedle = (value: unknown) => !needle || JSON.stringify(value || '').toLowerCase().includes(needle)
  return {
    events: data.events.filter(includesNeedle),
    processes: data.processes.filter(includesNeedle),
    flows: data.flows.filter(includesNeedle),
    dns: data.dns.filter(includesNeedle),
    findings: data.findings.filter(includesNeedle),
    extensions: extensions.filter(includesNeedle),
    saseComponents: saseComponents.filter(includesNeedle),
  }
}

function buildStats(data: VisibilityData, derived: ReturnType<typeof deriveVisibility>, device?: DeviceRecord) {
  const latest = data.events[0]
  const risk = data.findings.reduce((max, finding) => Math.max(max, finding.risk_score || 0), 0)
  const aiSignals = data.findings.filter((finding) =>
    `${finding.classification || ''} ${finding.title || ''} ${(finding.detected_patterns || []).join(' ')}`.toLowerCase().includes('ai')
    || `${finding.classification || ''} ${finding.title || ''}`.toLowerCase().includes('agent')
  ).length
  return {
    deviceLabel: platformName(device?.source || device?.device_id || ''),
    latestAge: latest ? ageFromMs(latest.received_at_ms || latest.timestamp_ms) : 'no data',
    risk,
    aiSignals,
    collectorHealthy: derived.collectorStatuses.filter((status) => status.status === 'healthy').length,
  }
}

function OverviewPanel({
  stats,
  derived,
  data,
  setSection,
}: {
  stats: ReturnType<typeof buildStats>
  derived: ReturnType<typeof deriveVisibility>
  data: VisibilityData
  setSection: (section: string) => void
}) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <StatusCard tone="emerald" icon={CheckCircle2} title="Collector Coverage" value={`${stats.collectorHealthy}/${derived.collectorStatuses.length}`} detail="healthy collectors in latest event window" onClick={() => setSection('devices')} />
        <StatusCard tone="blue" icon={Database} title="Evidence Volume" value={`${data.events.length}`} detail={`${data.processes.length} processes, ${data.flows.length} flows, ${data.dns.length} DNS rows`} onClick={() => setSection('evidence')} />
        <StatusCard tone="amber" icon={Shield} title="Controls" value="Observe" detail="draft controls require staged approval and rollback" onClick={() => setSection('controls')} />
      </div>
      <Panel title="Operational Summary" subtitle="Clarion-style triage view for AI endpoint governance.">
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <SummaryItem icon={Bot} title="AI activity" text="Browser AI, CLI agents, IDE tooling, local model runtimes, and tool bridges become first-class endpoint evidence." />
          <SummaryItem icon={Chrome} title="Enterprise browser evidence" text="Chromium profiles, extension manifests, policies, history, and vendor APIs fill gaps left by DNS or flow-only detection." />
          <SummaryItem icon={LockKeyhole} title="SSE/SASE awareness" text="Endpoint clients, services, adapters, proxy/PAC, and browser extensions show where traffic should also be visible in enterprise controls." />
          <SummaryItem icon={Cpu} title="Low-resource agent" text="Collectors are budgeted, bounded, low cadence where possible, and designed to feel invisible on Windows and Linux endpoints." />
        </div>
      </Panel>
    </div>
  )
}

function ActivityPanel({
  findings,
  dns,
  flows,
  onInvestigate,
}: {
  findings: FindingRecord[]
  dns: DnsRecord[]
  flows: FlowRecord[]
  onInvestigate: (selection: InvestigationSelection) => void
}) {
  const aiDns = dns.filter((record) => /chatgpt|openai|anthropic|claude|gemini|copilot|mistral|perplexity/i.test(record.query))
  const aiFlows = flows.filter((flow) => /chatgpt|openai|anthropic|claude|gemini|copilot/i.test(flow.remote_hostname || ''))

  return (
    <div className="space-y-4">
      <Panel title="AI Activity" subtitle="Known model destinations and agentic behavior evidence.">
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
          <MiniMetric label="Findings" value={findings.length} />
          <MiniMetric label="AI DNS" value={aiDns.length} />
          <MiniMetric label="AI Flows" value={aiFlows.length} />
        </div>
      </Panel>
      <Panel title="Recent Findings" subtitle="Click a row to open investigation context.">
        <FindingTable findings={findings} onInvestigate={onInvestigate} />
      </Panel>
    </div>
  )
}

function EvidencePanel({
  query,
  setQuery,
  activeTab,
  setActiveTab,
  filtered,
  onInvestigate,
}: {
  query: string
  setQuery: (query: string) => void
  activeTab: (typeof evidenceTabs)[number]['id']
  setActiveTab: (tab: (typeof evidenceTabs)[number]['id']) => void
  filtered: ReturnType<typeof filterVisibility>
  onInvestigate: (selection: InvestigationSelection) => void
}) {
  return (
    <Panel title="Evidence" subtitle="Process, flow, DNS, and raw event evidence for the selected endpoint.">
      <div className="mb-3 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex flex-wrap gap-2">
          {evidenceTabs.map((tab) => {
            const Icon = tab.icon
            const count = filtered[tab.id].length
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`inline-flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium ${
                  activeTab === tab.id ? 'bg-slate-950 text-white' : 'bg-gray-100 text-slate-700 hover:bg-gray-200'
                }`}
              >
                <Icon className="h-4 w-4" />
                {tab.label}
                <span className={activeTab === tab.id ? 'text-slate-300' : 'text-slate-500'}>{count}</span>
              </button>
            )
          })}
        </div>
        <SearchBox value={query} onChange={setQuery} />
      </div>
      <div className="max-h-[680px] overflow-auto rounded-md border border-gray-200">
        {activeTab === 'processes' && <ProcessTable processes={filtered.processes} onInvestigate={onInvestigate} />}
        {activeTab === 'flows' && <FlowTable flows={filtered.flows} />}
        {activeTab === 'dns' && <DnsTable dns={filtered.dns} />}
        {activeTab === 'events' && <EventTable events={filtered.events} />}
      </div>
    </Panel>
  )
}

function InventoryPanel({
  extensions,
  saseComponents,
}: {
  extensions: BrowserExtensionRecord[]
  saseComponents: SaseComponentRecord[]
}) {
  return (
    <div className="space-y-4">
      <Panel title="Browser Extension Inventory" subtitle="Chromium extension manifests across discovered browser profiles.">
        <ExtensionTable extensions={extensions} />
      </Panel>
      <Panel title="SSE/SASE Inventory" subtitle="Endpoint security access components, proxy/PAC state, tunnel adapters, and services.">
        <SaseTable components={saseComponents} />
      </Panel>
    </div>
  )
}

function ControlsPanel({ investigation }: { investigation: InvestigationData | null }) {
  const draft = investigation?.draft_controls?.[0]
  return (
    <Panel title="Controls" subtitle="Observe-only recommendations before enforcement adapters are enabled.">
      {draft ? <DraftControlPanel draft={draft} /> : (
        <EmptyState icon={Shield} title="No staged control selected" text="Open an investigation from a process or finding to generate evidence-backed draft control context." />
      )}
    </Panel>
  )
}

function DevicesPanel({
  devices,
  selectedDevice,
  selectDevice,
  collectorStatuses,
}: {
  devices: DeviceRecord[]
  selectedDevice: string
  selectDevice: (device: string) => void
  collectorStatuses: ReturnType<typeof deriveVisibility>['collectorStatuses']
}) {
  return (
    <div className="space-y-4">
      <Panel title="Devices" subtitle="Agent inventory and endpoint freshness.">
        <Table>
          <thead className="bg-gray-50">
            <tr>
              <Th>Device</Th>
              <Th>Platform</Th>
              <Th>Version</Th>
              <Th>Last Seen</Th>
              <Th>Events</Th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100 bg-white">
            {devices.length === 0 ? <EmptyRow colSpan={5} message="No reporting devices." /> : devices.map((device) => (
              <tr key={device.device_id} onClick={() => selectDevice(device.device_id)} className={`cursor-pointer hover:bg-gray-50 ${selectedDevice === device.device_id ? 'bg-slate-50' : ''}`}>
                <Td>{device.device_id}</Td>
                <Td>{platformName(device.source || device.device_id)}</Td>
                <Td muted>{device.sensor_version || 'unknown'}</Td>
                <Td muted>{ageFromMs(device.last_seen_ms)}</Td>
                <Td mono>{device.event_count}</Td>
              </tr>
            ))}
          </tbody>
        </Table>
      </Panel>
      <Panel title="Collector Health" subtitle="Latest collector status events for the selected endpoint.">
        <CollectorStatusList statuses={collectorStatuses} />
      </Panel>
    </div>
  )
}

function ConnectorsPanel({ saseComponents }: { saseComponents: SaseComponentRecord[] }) {
  const vendors = Array.from(new Set(saseComponents.map((component) => component.vendor))).filter(Boolean)
  return (
    <Panel title="Connectors" subtitle="Future Clarion, SASE/SSE, identity, firewall, proxy, and model-gateway integrations.">
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <ConnectorCard icon={ShieldCheck} title="Clarion" status="Planned" text="Share Aegis endpoint truth with Clarion enterprise context and policy orchestration." />
        <ConnectorCard icon={LockKeyhole} title="SSE/SASE" status={vendors.length ? vendors.join(', ') : 'No endpoint client detected'} text="Use endpoint inventory to choose Zscaler, Palo Alto, Cisco, Netskope, Cloudflare, or other enrichment paths." />
        <ConnectorCard icon={Globe2} title="Proxy and DNS" status="Planned" text="Enrich direct endpoint evidence with resolver, proxy, SNI, and gateway labels." />
        <ConnectorCard icon={Bot} title="Model Gateways" status="Planned" text="Separate approved model access from unsanctioned local or direct AI usage." />
      </div>
    </Panel>
  )
}

function ManagementPanel({ collectorStatuses }: { collectorStatuses: ReturnType<typeof deriveVisibility>['collectorStatuses'] }) {
  return (
    <Panel title="Management" subtitle="Agent budget, collector cadence, detection packs, and platform settings.">
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <ManagementCard icon={Cpu} title="Agent Resource Budget" text="CPU, memory, disk, and network budgets must be explicit per collector." />
        <ManagementCard icon={SlidersHorizontal} title="Collector Cadence" text="Fast snapshots, inventory collectors, and heavy enrichment need separate schedules." />
        <ManagementCard icon={Database} title="Detection Packs" text="Dynamic AI indicators should update independently of agent binaries." />
        <ManagementCard icon={HardDrive} title="Spool and Backpressure" text="Bounded local JSONL spool today; later compression, batch sizing, and queue-depth health." />
      </div>
      <div className="mt-4">
        <CollectorStatusList statuses={collectorStatuses} />
      </div>
    </Panel>
  )
}

function KpiCard({ icon: Icon, label, value, detail }: { icon: typeof Activity; label: string; value: string; detail: string }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</div>
          <div className="mt-2 truncate text-2xl font-semibold text-slate-950">{value || 'Unknown'}</div>
          <div className="mt-1 truncate text-xs text-slate-500">{detail}</div>
        </div>
        <Icon className="h-5 w-5 shrink-0 text-slate-500" />
      </div>
    </div>
  )
}

function Panel({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white shadow-sm">
      <div className="border-b border-gray-200 px-4 py-3">
        <h2 className="text-base font-semibold text-slate-950">{title}</h2>
        <p className="mt-1 text-sm text-slate-500">{subtitle}</p>
      </div>
      <div className="p-4">{children}</div>
    </div>
  )
}

function StatusCard({ icon: Icon, title, value, detail, tone, onClick }: { icon: typeof Activity; title: string; value: string; detail: string; tone: 'emerald' | 'blue' | 'amber'; onClick: () => void }) {
  const styles = {
    emerald: 'border-emerald-200 bg-emerald-50 text-emerald-800',
    blue: 'border-blue-200 bg-blue-50 text-blue-800',
    amber: 'border-amber-200 bg-amber-50 text-amber-800',
  }
  return (
    <button onClick={onClick} className={`rounded-lg border p-4 text-left ${styles[tone]}`}>
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-sm font-semibold">{title}</div>
          <div className="mt-2 text-3xl font-semibold">{value}</div>
          <div className="mt-1 text-sm opacity-80">{detail}</div>
        </div>
        <Icon className="h-6 w-6" />
      </div>
    </button>
  )
}

function SummaryItem({ icon: Icon, title, text }: { icon: typeof Activity; title: string; text: string }) {
  return (
    <div className="rounded-lg border border-gray-200 p-4">
      <div className="flex items-center gap-2 text-sm font-semibold text-slate-950">
        <Icon className="h-4 w-4 text-slate-500" />
        {title}
      </div>
      <p className="mt-2 text-sm leading-6 text-slate-600">{text}</p>
    </div>
  )
}

function MiniMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-gray-50 p-4">
      <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</div>
      <div className="mt-2 text-2xl font-semibold text-slate-950">{value}</div>
    </div>
  )
}

function SearchBox({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  return (
    <div className="relative min-w-0 lg:w-80">
      <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="h-9 w-full rounded-md border border-gray-300 bg-white pl-9 pr-3 text-sm outline-none focus:border-slate-600"
        placeholder="Filter evidence"
      />
    </div>
  )
}

function ProcessTable({ processes, onInvestigate }: { processes: ProcessRecord[]; onInvestigate: (selection: InvestigationSelection) => void }) {
  return (
    <Table>
      <thead className="bg-gray-50"><tr><Th>PID</Th><Th>Name</Th><Th>Path</Th><Th>Command</Th></tr></thead>
      <tbody className="divide-y divide-gray-100 bg-white">
        {processes.length === 0 ? <EmptyRow colSpan={4} message="No process evidence." /> : processes.slice(0, 80).map((process) => (
          <tr key={process.process_guid || `${process.pid}-${process.name}`} onClick={() => onInvestigate({ label: `${process.name} pid ${process.pid}`, process_guid: process.process_guid, pid: process.pid })} className="cursor-pointer hover:bg-gray-50">
            <Td mono>{process.pid}</Td><Td>{process.name}</Td><Td muted>{process.path || 'unknown'}</Td><Td muted>{process.command_line || 'not collected'}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function FlowTable({ flows }: { flows: FlowRecord[] }) {
  return (
    <Table>
      <thead className="bg-gray-50"><tr><Th>Process</Th><Th>Direction</Th><Th>Local</Th><Th>Remote</Th><Th>Host</Th></tr></thead>
      <tbody className="divide-y divide-gray-100 bg-white">
        {flows.length === 0 ? <EmptyRow colSpan={5} message="No flow evidence." /> : flows.slice(0, 80).map((flow) => (
          <tr key={flow.flow_id || `${flow.local_ip}-${flow.remote_ip}-${flow.remote_port}`} className="hover:bg-gray-50">
            <Td>{flow.process_name || (flow.pid ? `pid ${flow.pid}` : 'unknown')}</Td><Td><Badge value={flow.direction} /></Td><Td mono>{socket(flow.local_ip, flow.local_port)}</Td><Td mono>{socket(flow.remote_ip, flow.remote_port)}</Td><Td muted>{flow.remote_hostname || 'unknown'}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function DnsTable({ dns }: { dns: DnsRecord[] }) {
  return (
    <Table>
      <thead className="bg-gray-50"><tr><Th>Query</Th><Th>Answers</Th><Th>Resolver</Th><Th>Method</Th></tr></thead>
      <tbody className="divide-y divide-gray-100 bg-white">
        {dns.length === 0 ? <EmptyRow colSpan={4} message="No DNS evidence." /> : dns.slice(0, 80).map((record, index) => (
          <tr key={`${record.query}-${index}`} className="hover:bg-gray-50">
            <Td>{record.query}</Td><Td muted>{(record.answers || []).join(', ') || 'none'}</Td><Td mono>{record.resolver || 'unknown'}</Td><Td muted>{record.correlation_method || 'unknown'}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function FindingTable({ findings, onInvestigate }: { findings: FindingRecord[]; onInvestigate: (selection: InvestigationSelection) => void }) {
  return (
    <Table>
      <thead className="bg-gray-50"><tr><Th>Finding</Th><Th>Risk</Th><Th>Pattern</Th><Th>Evidence</Th></tr></thead>
      <tbody className="divide-y divide-gray-100 bg-white">
        {findings.length === 0 ? <EmptyRow colSpan={4} message="No findings." /> : findings.slice(0, 80).map((finding, index) => (
          <tr key={finding.finding_id || finding.detection_id || index} onClick={() => onInvestigate({ label: finding.title || finding.classification || finding.event_type, process_guid: finding.process_guid })} className="cursor-pointer hover:bg-gray-50">
            <Td><div className="font-medium text-slate-900">{finding.title || finding.classification || finding.event_type}</div><div className="mt-1 text-xs text-slate-500">{finding.detection_id || finding.finding_id || 'no id'}</div></Td>
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
    <Table>
      <thead className="bg-gray-50"><tr><Th>Sequence</Th><Th>Type</Th><Th>Source</Th><Th>Received</Th></tr></thead>
      <tbody className="divide-y divide-gray-100 bg-white">
        {events.length === 0 ? <EmptyRow colSpan={4} message="No events." /> : events.slice(0, 100).map((event) => (
          <tr key={event.event_id} className="hover:bg-gray-50">
            <Td mono>{event.sequence}</Td><Td>{event.event_type}</Td><Td muted>{event.source || 'unknown'}</Td><Td muted>{formatDate(event.received_at_ms || event.timestamp_ms)}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function ExtensionTable({ extensions }: { extensions: BrowserExtensionRecord[] }) {
  return (
    <Table>
      <thead className="bg-gray-50"><tr><Th>Extension</Th><Th>Browser/Profile</Th><Th>Version</Th><Th>Permissions</Th><Th>Host Permissions</Th></tr></thead>
      <tbody className="divide-y divide-gray-100 bg-white">
        {extensions.length === 0 ? <EmptyRow colSpan={5} message="No browser extensions observed." /> : extensions.slice(0, 100).map((extension) => (
          <tr key={`${extension.browser}-${extension.profile}-${extension.extension_id}-${extension.version}`} className="hover:bg-gray-50">
            <Td><div className="font-medium text-slate-900">{extension.name}</div><div className="mt-1 font-mono text-xs text-slate-500">{extension.extension_id}</div></Td>
            <Td muted>{extension.browser}<br />{extension.profile}</Td>
            <Td mono>{extension.version}</Td>
            <Td muted>{(extension.permissions || []).join(', ') || 'none'}</Td>
            <Td muted>{(extension.host_permissions || []).join(', ') || 'none'}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function SaseTable({ components }: { components: SaseComponentRecord[] }) {
  return (
    <Table>
      <thead className="bg-gray-50"><tr><Th>Component</Th><Th>Vendor</Th><Th>Product</Th><Th>Status</Th><Th>Evidence</Th></tr></thead>
      <tbody className="divide-y divide-gray-100 bg-white">
        {components.length === 0 ? <EmptyRow colSpan={5} message="No SSE/SASE components observed on this endpoint." /> : components.slice(0, 100).map((component, index) => (
          <tr key={`${component.component_type}-${component.name}-${index}`} className="hover:bg-gray-50">
            <Td><div className="font-medium text-slate-900">{component.name}</div><div className="mt-1 text-xs text-slate-500">{component.component_type}</div></Td>
            <Td>{component.vendor}</Td><Td muted>{component.product}</Td><Td><Badge value={component.status || 'observed'} /></Td><Td muted>{(component.evidence || []).join(' | ') || component.source}</Td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function InvestigationPanel({ loading, selection, investigation }: { loading: boolean; selection: InvestigationSelection | null; investigation: InvestigationData | null }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="mb-3 flex items-center justify-between gap-3">
        <div className="flex items-center gap-2"><Network className="h-5 w-5 text-slate-700" /><h2 className="text-base font-semibold">Investigation Path</h2></div>
        {loading && <RefreshCw className="h-4 w-4 animate-spin text-slate-500" />}
      </div>
      {!selection ? (
        <div className="rounded-md border border-dashed border-gray-300 p-4 text-sm leading-5 text-slate-600">Select a process or finding to trace linked process, flow, DNS, and finding evidence.</div>
      ) : (
        <div className="space-y-4">
          <div><div className="text-sm font-semibold text-slate-950">{selection.label}</div><div className="mt-1 text-xs text-slate-500">{selection.process_guid || (selection.pid !== undefined ? `pid ${selection.pid}` : 'device scope')}</div></div>
          {investigation && (
            <>
              <div className="grid grid-cols-4 gap-2 text-center">
                <MiniCount label="Proc" value={investigation.counts.processes} /><MiniCount label="Flow" value={investigation.counts.flows} /><MiniCount label="DNS" value={investigation.counts.dns} /><MiniCount label="Find" value={investigation.counts.findings} />
              </div>
              <PathSection title="Flows" items={investigation.flows.slice(0, 4).map((flow) => `${flow.process_name || 'unknown'} to ${socket(flow.remote_ip, flow.remote_port)}`)} />
              <PathSection title="DNS" items={investigation.dns.slice(0, 4).map((record) => `${record.query} to ${(record.answers || []).join(', ') || record.resolver || 'unknown'}`)} />
              {investigation.draft_controls?.[0] && <DraftControlPanel draft={investigation.draft_controls[0]} />}
            </>
          )}
        </div>
      )}
    </div>
  )
}

function DraftControlPanel({ draft }: { draft: DraftControl }) {
  return (
    <div className="space-y-3">
      <div className="rounded-md border border-gray-200 px-3 py-3">
        <div className="flex items-center justify-between gap-3">
          <div className="text-sm font-semibold text-slate-950">{draft.title}</div>
          <Badge value={displayMode(draft.mode)} />
        </div>
        <p className="mt-2 text-sm leading-5 text-slate-600">{draft.reason}</p>
      </div>
      <DraftList icon={FileText} title="Evidence Used" items={draft.evidence} />
      <DraftList icon={Undo2} title="Rollback Plan" items={draft.rollback} />
      <button disabled className="inline-flex w-full items-center justify-center gap-2 rounded-md border border-gray-300 bg-gray-100 px-3 py-2 text-sm font-medium text-slate-500">
        <FileText className="h-4 w-4" /> Policy staging API pending
      </button>
    </div>
  )
}

function AgentBudgetPanel() {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="mb-3 flex items-center gap-2"><Cpu className="h-5 w-5 text-slate-700" /><h2 className="text-base font-semibold">Agent Budget</h2></div>
      <div className="space-y-2 text-sm text-slate-700">
        <BudgetRow label="CPU" value="near 0% idle" />
        <BudgetRow label="Memory" value="target < 50 MB" />
        <BudgetRow label="Disk" value="bounded spool" />
        <BudgetRow label="Network" value="outbound only" />
      </div>
    </div>
  )
}

function ReadinessPanel({ collectorStatuses }: { collectorStatuses: ReturnType<typeof deriveVisibility>['collectorStatuses'] }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="mb-3 flex items-center gap-2"><Activity className="h-5 w-5 text-slate-700" /><h2 className="text-base font-semibold">Readiness</h2></div>
      <CollectorStatusList statuses={collectorStatuses.slice(0, 6)} />
    </div>
  )
}

function CollectorStatusList({ statuses }: { statuses: ReturnType<typeof deriveVisibility>['collectorStatuses'] }) {
  if (statuses.length === 0) return <div className="rounded-md bg-gray-50 px-3 py-3 text-sm text-slate-500">No collector status in the current event window.</div>
  return (
    <div className="space-y-2">
      {statuses.slice(0, 12).map((status, index) => (
        <div key={`${status.collector}-${index}`} className="rounded-md border border-gray-200 px-3 py-2">
          <div className="flex items-center justify-between gap-3">
            <div className="text-sm font-medium text-slate-900">{status.collector}</div>
            <Risk value={status.status === 'healthy' ? 0 : 50} severity={status.status} />
          </div>
          <div className="mt-1 text-xs leading-5 text-slate-500">{status.message}</div>
        </div>
      ))}
    </div>
  )
}

function ConnectorCard({ icon: Icon, title, status, text }: { icon: typeof Activity; title: string; status: string; text: string }) {
  return <SummaryItem icon={Icon} title={`${title}: ${status}`} text={text} />
}

function ManagementCard({ icon: Icon, title, text }: { icon: typeof Activity; title: string; text: string }) {
  return <SummaryItem icon={Icon} title={title} text={text} />
}

function EmptyState({ icon: Icon, title, text }: { icon: typeof Activity; title: string; text: string }) {
  return (
    <div className="rounded-lg border border-dashed border-gray-300 p-8 text-center">
      <Icon className="mx-auto h-8 w-8 text-slate-400" />
      <div className="mt-3 text-sm font-semibold text-slate-900">{title}</div>
      <p className="mx-auto mt-2 max-w-md text-sm leading-6 text-slate-500">{text}</p>
    </div>
  )
}

function DraftList({ icon: Icon, title, items }: { icon: typeof Activity; title: string; items: string[] }) {
  return (
    <div>
      <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-slate-500"><Icon className="h-4 w-4" />{title}</div>
      <div className="space-y-2">{items.map((item, index) => <div key={`${title}-${index}`} className="rounded-md border border-gray-200 px-3 py-2 text-sm leading-5 text-slate-700">{item}</div>)}</div>
    </div>
  )
}

function MiniCount({ label, value }: { label: string; value: number }) {
  return <div className="rounded-md bg-gray-100 px-2 py-2"><div className="text-lg font-semibold text-slate-950">{value}</div><div className="text-xs text-slate-500">{label}</div></div>
}

function PathSection({ title, items }: { title: string; items: string[] }) {
  return <div><div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500">{title}</div>{items.length === 0 ? <div className="rounded-md bg-gray-50 px-3 py-2 text-sm text-slate-500">No linked evidence</div> : <div className="space-y-2">{items.map((item, index) => <div key={`${title}-${index}`} className="rounded-md border border-gray-200 px-3 py-2 text-sm text-slate-700">{item}</div>)}</div>}</div>
}

function BudgetRow({ label, value }: { label: string; value: string }) {
  return <div className="flex items-center justify-between rounded-md bg-gray-50 px-3 py-2"><span className="font-medium">{label}</span><span className="text-slate-500">{value}</span></div>
}

function Table({ children }: { children: React.ReactNode }) {
  return <table className="min-w-full table-fixed text-left text-sm">{children}</table>
}

function Th({ children }: { children: React.ReactNode }) {
  return <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">{children}</th>
}

function Td({ children, muted, mono }: { children: React.ReactNode; muted?: boolean; mono?: boolean }) {
  return <td className={`max-w-[360px] truncate px-4 py-3 align-top ${muted ? 'text-slate-500' : 'text-slate-800'} ${mono ? 'font-mono text-xs' : ''}`}>{children}</td>
}

function EmptyRow({ colSpan, message }: { colSpan: number; message: string }) {
  return <tr><td colSpan={colSpan} className="px-4 py-10 text-center text-sm text-slate-500">{message}</td></tr>
}

function Badge({ value }: { value: string }) {
  return <span className="rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-slate-700">{value}</span>
}

function Risk({ value, severity }: { value: number; severity?: string }) {
  const tone = value >= 70 ? 'bg-red-100 text-red-800' : value >= 40 ? 'bg-amber-100 text-amber-800' : 'bg-emerald-50 text-emerald-700'
  return <span className={`rounded-md px-2 py-1 text-xs font-semibold ${tone}`}>{severity || 'info'} {value}</span>
}

function sectionTitle(section: string) {
  const titles: Record<string, string> = {
    overview: 'Operational Overview',
    activity: 'AI Activity',
    evidence: 'Endpoint Evidence',
    inventory: 'Inventory',
    findings: 'Findings',
    controls: 'Controls',
    devices: 'Devices',
    connectors: 'Connectors',
    management: 'Management',
  }
  return titles[section] || 'Aegis'
}

function sectionSubtitle(section: string) {
  const subtitles: Record<string, string> = {
    overview: 'Current endpoint posture, AI evidence, collector coverage, and readiness.',
    activity: 'Known AI destinations, agentic behavior, and user/application context.',
    evidence: 'Raw process, flow, DNS, and event records from the selected endpoint.',
    inventory: 'Browser extensions, enterprise browsers, and SSE/SASE endpoint components.',
    findings: 'Explainable detections with evidence and risk scoring.',
    controls: 'Observe-only draft controls, blast-radius notes, and rollback plans.',
    devices: 'Agent freshness, platform visibility, and collector health.',
    connectors: 'Clarion, SASE/SSE, proxy, firewall, identity, and model gateway integration paths.',
    management: 'Agent budgets, collector cadence, detection packs, and platform settings.',
  }
  return subtitles[section] || ''
}

function displayMode(value: string) {
  return value === 'observe-only' ? 'Observe-only' : value || 'Draft'
}

function socket(ip?: string, port?: number) {
  return `${ip || 'unknown'}${port ? `:${port}` : ''}`
}

function formatDate(ms?: number) {
  return ms ? new Date(ms).toLocaleString() : 'unknown'
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
