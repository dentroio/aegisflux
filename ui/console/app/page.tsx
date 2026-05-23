'use client'

import { Suspense, useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { AgentsManagementPanel } from '@/components/AgentsManagementPanel'
import { InventoryPanel } from '@/components/InventoryPanel'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import { FirstValueTourBanner } from '@/components/FirstValueTour'
import {
  CopyValueButton,
  EmptyState,
  FormattedValue,
  KpiTile,
  SummaryStrip,
  WorkbenchHeader,
} from '@/components/workbench/primitives'
import { formatAgentId } from '@/shared/formatting'
import {
  Activity,
  AlertTriangle,
  Bot,
  Chrome,
  Cpu,
  LockKeyhole,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
  Sparkles,
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
  device_id?: string
  process_guid: string
  pid: number
  name: string
  path?: string
  command_line?: string
}

type FlowRecord = {
  device_id?: string
  flow_id: string
  pid?: number
  process_name?: string
  protocol: string
  direction: string
  remote_ip: string
  remote_port?: number
  remote_hostname?: string
}

type DnsRecord = {
  device_id?: string
  query: string
  answers?: string[]
  resolver?: string
  correlation_method?: string
}

type FindingRecord = {
  device_id?: string
  event_type: string
  classification?: string
  severity?: string
  title?: string
  risk_score?: number
  detected_patterns?: string[]
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
  device_id?: string
  browser: string
  profile: string
  extension_id: string
  name: string
  version: string
  permissions?: string[]
  host_permissions?: string[]
}

type SaseComponentRecord = {
  device_id?: string
  component_type: string
  vendor: string
  product: string
  name: string
  status?: string | null
  evidence?: string[]
}

type AgentPerformanceRecord = {
  device_id?: string
  os?: string
  process_cpu_percent?: number | null
  process_memory_rss_mb?: number | null
  collector_runtime_ms?: number
  collector_name?: string
  event_queue_depth?: number
  spool_bytes?: number
  pack_eval_runtime_ms?: number | null
  received_at_ms?: number
}

type VisibilityData = {
  events: EventRecord[]
  processes: ProcessRecord[]
  flows: FlowRecord[]
  dns: DnsRecord[]
  findings: FindingRecord[]
}

type CollectorStatus = {
  device_id: string
  collector: string
  status: string
  message: string
  received_at_ms: number
}

/** Mirrors ingest /v1/visibility/summary/dashboard model payload. */
type DashboardSummaryModelDTO = {
  totalDevices: number
  onlineDevices: number
  offlineDevices: number
  maxRisk: number
  aiSignals: number
  eventCount: number
  extensionCount: number
  saseCount: number
  extensions: BrowserExtensionRecord[]
  sase: SaseComponentRecord[]
  healthyCollectorPairs: number
  maxCpuPercent: number | null
  avgCpuPercent: number | null
  maxMemoryRssMb: number | null
  collectorStatuses: CollectorStatus[]
  performance: AgentPerformanceRecord[]
}



type DashboardWidgetDef = {
  id: string
  title: string
  description: string
  dataSource: string
  defaultSize: 'sm' | 'md' | 'lg'
  icon: typeof Activity
  gradient: string
}

const DASHBOARD_WIDGET_REGISTRY: DashboardWidgetDef[] = [
  {
    id: 'platform_status',
    title: 'Platform Status',
    description: 'Overall console health from endpoint freshness',
    dataSource: 'visibility.devices + derived health',
    defaultSize: 'md',
    icon: Activity,
    gradient: 'from-slate-600 to-slate-700',
  },
  {
    id: 'endpoint_freshness',
    title: 'Endpoint Freshness',
    description: 'Fresh vs total reporting endpoints',
    dataSource: 'visibility.devices last_seen',
    defaultSize: 'md',
    icon: Server,
    gradient: 'from-teal-600 to-teal-700',
  },
  {
    id: 'ai_activity',
    title: 'AI Activity',
    description: 'Heuristic AI-related findings',
    dataSource: 'visibility.findings + patterns',
    defaultSize: 'sm',
    icon: Bot,
    gradient: 'from-blue-600 to-blue-700',
  },
  {
    id: 'detection_pack_coverage',
    title: 'Detection Pack Coverage',
    description: 'Healthy collector pairs as pack-readiness proxy',
    dataSource: 'visibility events aegis.collector.status',
    defaultSize: 'md',
    icon: ShieldCheck,
    gradient: 'from-indigo-600 to-indigo-700',
  },
  {
    id: 'agent_performance_budget',
    title: 'Agent Performance Budget',
    description: 'CPU/RSS pressure from agent performance stream',
    dataSource: 'visibility events aegis.agent.performance',
    defaultSize: 'md',
    icon: Cpu,
    gradient: 'from-violet-600 to-violet-700',
  },
  {
    id: 'enterprise_control_inventory',
    title: 'Enterprise Control Inventory',
    description: 'SSE/SASE component observations',
    dataSource: 'visibility events aegis.sase_component.observed',
    defaultSize: 'sm',
    icon: LockKeyhole,
    gradient: 'from-sky-600 to-sky-700',
  },
]

function dashboardRegistryMap(): Map<string, DashboardWidgetDef> {
  return new Map(DASHBOARD_WIDGET_REGISTRY.map((widget) => [widget.id, widget]))
}

function AegisDashboardBody() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const panelParam = searchParams.get('panel')
  const nextParam = searchParams.get('next')
  const nextTarget = nextParam && nextParam.startsWith('/') && !nextParam.startsWith('//') ? nextParam : ''
  const mainPanel = panelParam === 'agents' || panelParam === 'inventory' ? panelParam : 'dashboard'
  const inventoryDeviceFilter = (searchParams.get('device') || '').trim()

  const [authChecked, setAuthChecked] = useState(false)
  const [authenticated, setAuthenticated] = useState(false)
  const [loginUser, setLoginUser] = useState('admin')
  const [loginPassword, setLoginPassword] = useState('')
  const [loginError, setLoginError] = useState('')
  const [devices, setDevices] = useState<DeviceRecord[]>([])
  const [query, setQuery] = useState('')
  const [data, setData] = useState<VisibilityData>({ events: [], processes: [], flows: [], dns: [], findings: [] })
  const [serverDashboardModel, setServerDashboardModel] = useState<DashboardSummaryModelDTO | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null)
  const [aiChip, setAiChip] = useState<string | null>(null)

  useEffect(() => {
    if (!authenticated) return undefined
    let cancelled = false
    fetch('/api/actions/platform/ai/providers/summary', { cache: 'no-store' })
      .then((res) => (res.ok ? res.json() : null))
      .then((json) => {
        if (cancelled || !json || typeof json.summary !== 'string') return
        setAiChip(json.summary as string)
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [authenticated])

  useEffect(() => {
    setAuthenticated(window.localStorage.getItem('aegisflux.labAuth') === 'admin')
    setAuthChecked(true)
  }, [])

  useEffect(() => {
    if (!authenticated || mainPanel !== 'dashboard') return undefined
    fetchDashboard()
    const interval = setInterval(() => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return
      fetchDashboard()
    }, 60000)
    return () => clearInterval(interval)
  }, [authenticated, mainPanel])

  async function fetchJson<T>(path: string): Promise<T> {
    const response = await fetch(path, { cache: 'no-store' })
    if (!response.ok) throw new Error(`${path} returned HTTP ${response.status}`)
    return response.json()
  }

  async function fetchDashboard() {
    try {
      setLoading(true)
      setError(null)
      const summary = await fetchJson<{
        ok?: boolean
        devices?: DeviceRecord[]
        model?: DashboardSummaryModelDTO
      }>('/api/visibility/summary/dashboard')
      if (!summary || summary.ok === false || !summary.model) {
        throw new Error('Dashboard summary unavailable')
      }
      setDevices(summary.devices || [])
      setServerDashboardModel(summary.model)
      setData({ events: [], processes: [], flows: [], dns: [], findings: [] })
      setLastRefresh(new Date())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load AegisFlux dashboard')
    } finally {
      setLoading(false)
    }
  }

  function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (loginUser.trim() === 'admin' && loginPassword === 'admin') {
      window.localStorage.setItem('aegisflux.labAuth', 'admin')
      setAuthenticated(true)
      setLoginError('')
      setLoginPassword('')
      if (nextTarget) router.replace(nextTarget)
      return
    }
    setLoginError('Invalid lab credentials. Use admin/admin.')
  }

  function handleLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    setAuthenticated(false)
    setDevices([])
    setServerDashboardModel(null)
    setData({ events: [], processes: [], flows: [], dns: [], findings: [] })
  }

  const model = useMemo(() => {
    if (serverDashboardModel) return adaptDashboardSummaryModel(serverDashboardModel)
    return buildDashboardModel(data, devices)
  }, [serverDashboardModel, data, devices])
  const filteredDevices = useMemo(() => {
    const needle = query.trim().toLowerCase()
    if (!needle) return devices
    return devices.filter((device) => JSON.stringify(device).toLowerCase().includes(needle))
  }, [devices, query])

  const health = model.offlineDevices > 0
    ? { label: 'Attention', tone: 'amber' as const, text: `${model.offlineDevices} endpoint${model.offlineDevices === 1 ? '' : 's'} not fresh` }
    : model.totalDevices > 0
      ? { label: 'Healthy', tone: 'emerald' as const, text: 'All reporting endpoints are fresh' }
      : { label: 'Waiting', tone: 'slate' as const, text: 'No endpoint telemetry yet' }

  if (!authChecked) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-slate-400 text-sm font-semibold">Loading AegisFlux</div>
      </div>
    )
  }

  if (!authenticated) {
    return (
      <LoginScreen
        username={loginUser}
        password={loginPassword}
        error={loginError}
        onUsernameChange={setLoginUser}
        onPasswordChange={setLoginPassword}
        onSubmit={handleLogin}
      />
    )
  }

  const shellNavId =
    mainPanel === 'agents' ? 'agents' : mainPanel === 'inventory' ? 'inventory' : 'dashboard'
  const shellBreadcrumbs = [
    {
      label: mainPanel === 'dashboard' ? 'Dashboard' : mainPanel === 'agents' ? 'Agents' : 'Telemetry',
    },
  ]

  return (
    <ConsoleShell
      activeNavId={shellNavId}
      breadcrumbs={shellBreadcrumbs}
      health={health}
      onLogout={handleLogout}
      aiHealthSummary={aiChip}
    >
      <div className={`w-full max-w-[1500px] ${mainPanel === 'dashboard' ? 'px-5 py-6' : 'min-w-0 px-4 py-4'}`}>
        {mainPanel === 'dashboard' ? (
          <DashboardScanSurface
            data={data}
            error={error}
            filteredDevices={filteredDevices}
            health={health}
            lastRefresh={lastRefresh}
            loading={loading}
            model={model}
            query={query}
            setQuery={setQuery}
            onRefresh={fetchDashboard}
          />
        ) : mainPanel === 'agents' ? (
          <div className="min-w-0 w-full">
            <AgentsManagementPanel embedded />
          </div>
        ) : (
          <div className="min-w-0 w-full">
            <InventoryPanel embedded deviceFilter={inventoryDeviceFilter} />
          </div>
        )}
      </div>
    </ConsoleShell>
  )
}

export default function AegisDashboard() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center bg-gray-50 text-sm font-semibold text-slate-600">
          Loading console…
        </div>
      }
    >
      <AegisDashboardBody />
    </Suspense>
  )
}

function DashboardScanSurface({
  data,
  error,
  filteredDevices,
  health,
  lastRefresh,
  loading,
  model,
  query,
  setQuery,
  onRefresh,
}: {
  data: VisibilityData
  error: string | null
  filteredDevices: DeviceRecord[]
  health: { label: string; tone: 'emerald' | 'amber' | 'slate'; text: string }
  lastRefresh: Date | null
  loading: boolean
  model: ReturnType<typeof buildDashboardModel>
  query: string
  setQuery: (value: string) => void
  onRefresh: () => void
}) {
  return (
    <>
      <WorkbenchHeader
        title="Dashboard"
        subtitle="Adaptive security and real-time protection across the AegisFlux fleet."
        actions={
          <>
            <span className="text-sm text-slate-500">
              {lastRefresh ? `Updated ${lastRefresh.toLocaleTimeString()}` : 'Waiting for refresh'}
            </span>
            <button
              onClick={onRefresh}
              className="inline-flex h-8 items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 text-sm font-medium text-slate-700 hover:bg-gray-50 transition-colors"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </>
        }
      />

      {error && (
        <div className="mb-4 flex items-center gap-3 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      <FirstValueTourBanner />
      <DashboardReadinessBand health={health} model={model} />

      <section className="mb-5 grid grid-cols-2 gap-4 lg:grid-cols-3 xl:grid-cols-6">
        {DASHBOARD_WIDGET_REGISTRY.map((widget) => (
          <Widget
            key={widget.id}
            icon={widget.icon}
            title={widget.title}
            gradient={widget.gradient}
            value={widgetValue(widget.id, model, data, health)}
            detail={widgetDetail(widget.id, model, data, health)}
          />
        ))}
      </section>

      <DashboardAttentionBand model={model} />
      <DashboardEndpointScan devices={filteredDevices} query={query} setQuery={setQuery} />
    </>
  )
}

function DashboardReadinessBand({
  health,
  model,
}: {
  health: { label: string; text: string }
  model: ReturnType<typeof buildDashboardModel>
}) {
  return (
    <section className="mb-5 grid gap-4 lg:grid-cols-[minmax(0,1.3fr)_minmax(320px,0.7fr)]">
      {/* Platform status card */}
      <div className="rounded-lg border border-gray-200 bg-white p-5 shadow-md flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-sm font-semibold text-gray-600">
            <Activity className="h-4 w-4 text-emerald-500" />
            Platform status
          </div>
          <div className="mt-2 text-3xl font-bold tracking-tight text-gray-900">{health.label}</div>
          <p className="mt-2 max-w-lg text-sm leading-6 text-gray-500">
            {health.text}. Observe endpoint evidence, adapt detections, and enforce only when controls are proven.
          </p>
        </div>
        <div className="shrink-0">
          <SummaryStrip>
            <KpiTile label="Endpoints" value={model.totalDevices} />
            <KpiTile label="Fresh" value={model.onlineDevices} />
            <KpiTile label="Collectors" value={model.healthyCollectorPairs} />
            <KpiTile label="Max Risk" value={model.maxRisk} />
          </SummaryStrip>
        </div>
      </div>

      {/* Signal focus dark card */}
      <div className="rounded-lg bg-slate-900 dark:bg-slate-800 border border-white/10 p-5 text-white shadow-md">
        <div className="flex items-center gap-2 text-sm font-semibold text-slate-200">
          <Sparkles className="h-4 w-4 text-clarion-teal" />
          Signal focus
        </div>
        <div className="mt-4 grid grid-cols-3 gap-3">
          <DarkStat icon={Bot} label="AI" value={model.aiSignals} />
          <DarkStat icon={Chrome} label="Ext" value={model.extensionCount} />
          <DarkStat icon={LockKeyhole} label="SASE" value={model.saseCount} />
        </div>
        <p className="mt-4 text-sm leading-6 text-slate-400">
          AI tools, browser extensions, and SASE components observed across the fleet.
        </p>
        <a
          href="/discover/abom"
          className="mt-3 inline-flex items-center gap-1.5 rounded-md border border-clarion-teal/40 bg-clarion-teal/10 px-3 py-1.5 text-xs font-semibold text-clarion-teal hover:bg-clarion-teal/20 transition-colors"
        >
          Agent Bill of Materials →
        </a>
      </div>
    </section>
  )
}

function DashboardAttentionBand({ model }: { model: ReturnType<typeof buildDashboardModel> }) {
  if (model.offlineDevices === 0 && model.maxRisk <= 70 && model.aiSignals === 0) return null
  return (
    <section className="mb-5 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
      <div className="flex items-start gap-2 font-semibold">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        Attention required
      </div>
      <div className="mt-2 flex flex-wrap gap-2 text-xs">
        {model.offlineDevices > 0 ? <span className="rounded-full bg-white/70 px-2 py-1">{model.offlineDevices} stale endpoint(s)</span> : null}
        {model.maxRisk > 70 ? <span className="rounded-full bg-white/70 px-2 py-1">High-risk finding score detected</span> : null}
        {model.aiSignals > 0 ? <span className="rounded-full bg-white/70 px-2 py-1">{model.aiSignals} AI-shaped signals in current window</span> : null}
      </div>
      <div className="mt-3 flex flex-wrap gap-3 text-xs font-semibold">
        <a href="/agents" className="text-amber-900 underline decoration-amber-400 underline-offset-2">Open Agents workbench</a>
        <a href="/detections" className="text-amber-900 underline decoration-amber-400 underline-offset-2">Open Detections</a>
        <a href="/operate/events" className="text-amber-900 underline decoration-amber-400 underline-offset-2">Open Event feed</a>
      </div>
    </section>
  )
}

function DashboardEndpointScan({
  devices,
  query,
  setQuery,
}: {
  devices: DeviceRecord[]
  query: string
  setQuery: (value: string) => void
}) {
  return (
    <section className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_320px]">
      <div className="rounded-lg border border-gray-200 bg-white shadow-md overflow-hidden">
        <div className="flex flex-col gap-3 border-b border-gray-200 px-4 py-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 className="text-base font-semibold text-gray-800">Endpoint scan list</h2>
            <p className="mt-0.5 text-xs text-gray-400">Compact view. Open Agents for deep investigation.</p>
          </div>
          <div className="relative lg:w-72">
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              className="w-full pl-9 pr-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-400 focus:border-transparent bg-gray-50 transition-colors"
              placeholder="Search endpoint id, OS, or sensor"
            />
          </div>
        </div>
        <div className="divide-y divide-gray-100 max-h-[480px] overflow-auto">
          {devices.length === 0 ? (
            <div className="p-4">
              <EmptyState title="No endpoints found" message="Try a broader query or open Agents for advanced filters." />
            </div>
          ) : (
            devices.slice(0, 12).map((device) => {
              const active = Date.now() - device.last_seen_ms < 5 * 60 * 1000
              const findings = Number(device.event_type_count?.['aegis.risk_finding.created'] || 0)
              return (
                <a
                  key={device.device_id}
                  href={`/agents/${encodeURIComponent(device.device_id)}`}
                  className="grid grid-cols-[minmax(0,1fr)_auto] gap-3 px-4 py-3 hover:bg-slate-50"
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <FreshDot active={active} />
                      <FormattedValue value={formatAgentId(device.device_id)} fullValue={device.device_id} />
                      <CopyValueButton value={device.device_id} label="Copy endpoint id" />
                      <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600">{platformName(device.source || device.device_id)}</span>
                    </div>
                    <p className="mt-1 truncate text-xs text-slate-500">{device.sensor_version} · last seen {ageFromMs(device.last_seen_ms)}</p>
                  </div>
                  <div className="self-center">
                    <CountPill label="Find" value={findings} tone={findings ? 'amber' : 'slate'} />
                  </div>
                </a>
              )
            })
          )}
        </div>
        <div className="border-t border-gray-100 px-4 py-3 text-xs text-gray-500">
          Showing up to 12 endpoints.{' '}
          <a href="/agents" className="font-semibold text-blue-600 hover:text-blue-800 transition-colors">
            Open Agents workbench →
          </a>
        </div>
      </div>

      <aside className="rounded-lg border border-gray-200 bg-white p-4 shadow-md">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-400">Next best actions</h3>
        <div className="mt-3 space-y-1.5 text-sm">
          <a href="/agents" className="block rounded-lg px-3 py-2 text-gray-600 hover:bg-gray-50 hover:text-gray-900 transition-colors">
            Review stale endpoints and collector health
          </a>
          <a href="/discover/abom" className="block rounded-lg px-3 py-2 font-medium text-clarion-blue bg-clarion-blue/10 hover:bg-clarion-blue/20 transition-colors">
            Agent Bill of Materials
          </a>
          <a href="/analyze/evidence" className="block rounded-lg px-3 py-2 font-medium text-clarion-blue bg-clarion-blue/10 hover:bg-clarion-blue/20 transition-colors">
            Trace a finding through evidence graph
          </a>
          <a href="/analyze/research" className="block rounded-lg px-3 py-2 font-medium text-clarion-blue bg-clarion-blue/10 hover:bg-clarion-blue/20 transition-colors">
            AI Research Feed
          </a>
          <a href="/inventory" className="block rounded-lg px-3 py-2 text-gray-600 hover:bg-gray-50 hover:text-gray-900 transition-colors">
            AI tool and extension inventory
          </a>
          <a href="/detections" className="block rounded-lg px-3 py-2 text-gray-600 hover:bg-gray-50 hover:text-gray-900 transition-colors">
            Detection pack coverage
          </a>
          <a href="/operate/events" className="block rounded-lg px-3 py-2 text-gray-600 hover:bg-gray-50 hover:text-gray-900 transition-colors">
            Audit event feed
          </a>
        </div>
      </aside>
    </section>
  )
}

function LoginScreen({
  username,
  password,
  error,
  onUsernameChange,
  onPasswordChange,
  onSubmit,
}: {
  username: string
  password: string
  error: string
  onUsernameChange: (value: string) => void
  onPasswordChange: (value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  return (
    <div className="relative min-h-screen w-full overflow-hidden bg-slate-900 font-sans text-slate-900 dark:text-slate-100">
      {/* Decorative gradient orbs */}
      <div className="pointer-events-none absolute inset-0 overflow-hidden">
        <div className="absolute top-[10%] left-[15%] h-96 w-96 rounded-full bg-clarion-blue/20 blur-[100px]" />
        <div className="absolute top-[30%] right-[15%] h-[28rem] w-[28rem] rounded-full bg-clarion-teal/20 blur-[100px]" />
        <div className="absolute bottom-[10%] left-[30%] h-80 w-80 rounded-full bg-clarion-purple/20 blur-[100px]" />
      </div>

      {/* Center the card */}
      <div className="relative flex min-h-screen items-center justify-center p-4">
        <div className="w-full max-w-sm rounded-2xl border border-white/10 bg-white/95 shadow-2xl backdrop-blur-sm p-8">
          {/* Logo */}
          <div className="flex items-center gap-3 mb-6">
            <div className="w-10 h-10 rounded-xl bg-slate-950 flex items-center justify-center overflow-hidden shrink-0">
              <img src="/aegisflux-shield.png" alt="" className="w-full h-full object-cover" />
            </div>
            <div>
              <h1 className="text-xl font-extrabold tracking-tight text-slate-950 leading-none">
                Aegis<span className="text-blue-600">Flux</span>
              </h1>
              <p className="text-xs text-slate-500 mt-0.5">Lab console</p>
            </div>
          </div>

          <h2 className="text-lg font-bold text-slate-900 mb-1">Sign in</h2>
          <p className="text-sm text-slate-500 mb-5">Access the AegisFlux security console.</p>

          <form className="flex flex-col gap-4" onSubmit={onSubmit}>
            <label className="flex flex-col gap-1.5 text-sm font-medium text-slate-700">
              Username
              <input
                value={username}
                onChange={(event) => onUsernameChange(event.target.value)}
                autoComplete="username"
                className="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-slate-900 outline-none focus:ring-2 focus:ring-clarion-blue/20 focus:border-clarion-blue transition-colors"
              />
            </label>
            <label className="flex flex-col gap-1.5 text-sm font-medium text-slate-700">
              Password
              <input
                value={password}
                onChange={(event) => onPasswordChange(event.target.value)}
                autoComplete="current-password"
                type="password"
                className="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-slate-900 outline-none focus:ring-2 focus:ring-clarion-blue/20 focus:border-clarion-blue transition-colors"
              />
            </label>
            {error ? (
              <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                {error}
              </div>
            ) : null}
            <button
              type="submit"
              className="mt-1 rounded-xl bg-clarion-blue px-4 py-2.5 text-sm font-semibold text-white hover:bg-clarion-blue/90 focus:outline-none focus:ring-4 focus:ring-clarion-blue/20 transition-all"
            >
              Sign in
            </button>
          </form>

          <p className="mt-4 text-center text-xs text-slate-400">
            Lab credentials: <span className="font-mono font-semibold">admin / admin</span>
          </p>
        </div>
      </div>
    </div>
  )
}

function buildDashboardModel(data: VisibilityData, devices: DeviceRecord[]) {
  const extensions = data.events
    .filter((event) => event.event_type === 'aegis.browser_extension.observed')
    .map((event) => ({ ...event.payload, device_id: event.device_id }) as BrowserExtensionRecord)
  const sase = data.events
    .filter((event) => event.event_type === 'aegis.sase_component.observed')
    .map((event) => ({ ...event.payload, device_id: event.device_id }) as SaseComponentRecord)
  const collectorStatuses = data.events
    .filter((event) => event.event_type === 'aegis.collector.status')
    .map((event) => ({
      device_id: event.device_id,
      collector: String(event.payload.collector || 'unknown'),
      status: String(event.payload.status || 'unknown'),
      message: String(event.payload.message || ''),
      received_at_ms: event.received_at_ms || event.timestamp_ms,
    }))
  const performance = data.events
    .filter((event) => event.event_type === 'aegis.agent.performance')
    .map((event) => ({
      ...event.payload,
      device_id: event.device_id,
      received_at_ms: event.received_at_ms || event.timestamp_ms,
    }) as AgentPerformanceRecord)
  const cpuSamples = performance
    .map((record) => record.process_cpu_percent)
    .filter((value): value is number => typeof value === 'number')
  const memorySamples = performance
    .map((record) => record.process_memory_rss_mb)
    .filter((value): value is number => typeof value === 'number')

  const onlineDevices = devices.filter((device) => Date.now() - device.last_seen_ms < 5 * 60 * 1000).length
  const maxRisk = data.findings.reduce((max, finding) => Math.max(max, finding.risk_score || 0), 0)
  const aiSignals = data.findings.filter((finding) =>
    `${finding.title || ''} ${finding.classification || ''} ${(finding.detected_patterns || []).join(' ')}`.toLowerCase().match(/ai|agent|browser|model/),
  ).length

  return {
    totalDevices: devices.length,
    onlineDevices,
    offlineDevices: Math.max(0, devices.length - onlineDevices),
    maxRisk,
    aiSignals,
    eventCount: data.events.length,
    extensionCount: extensions.length,
    saseCount: sase.length,
    extensions,
    sase,
    collectorStatuses,
    performance,
    maxCpuPercent: cpuSamples.length ? Math.max(...cpuSamples) : null,
    avgCpuPercent: cpuSamples.length ? cpuSamples.reduce((sum, value) => sum + value, 0) / cpuSamples.length : null,
    maxMemoryRssMb: memorySamples.length ? Math.max(...memorySamples) : null,
    healthyCollectorPairs: new Set(collectorStatuses.filter((status) => status.status === 'healthy').map((status) => `${status.device_id}:${status.collector}`)).size,
  }
}

function adaptDashboardSummaryModel(m: DashboardSummaryModelDTO): ReturnType<typeof buildDashboardModel> {
  return {
    totalDevices: m.totalDevices,
    onlineDevices: m.onlineDevices,
    offlineDevices: m.offlineDevices,
    maxRisk: m.maxRisk,
    aiSignals: m.aiSignals,
    eventCount: m.eventCount,
    extensionCount: m.extensionCount,
    saseCount: m.saseCount,
    extensions: m.extensions || [],
    sase: m.sase || [],
    collectorStatuses: m.collectorStatuses || [],
    performance: m.performance || [],
    maxCpuPercent: m.maxCpuPercent,
    avgCpuPercent: m.avgCpuPercent,
    maxMemoryRssMb: m.maxMemoryRssMb,
    healthyCollectorPairs: m.healthyCollectorPairs,
  }
}

function widgetValue(
  id: string,
  model: ReturnType<typeof buildDashboardModel>,
  data: VisibilityData,
  health: { label: string; text: string },
) {
  if (id === 'platform_status') return health.label
  if (id === 'endpoint_freshness') return `${model.onlineDevices}/${Math.max(model.totalDevices, 1)}`
  if (id === 'ai_activity') return model.aiSignals
  if (id === 'detection_pack_coverage') return model.healthyCollectorPairs
  if (id === 'agent_performance_budget') return model.maxCpuPercent === null ? 'n/a' : `${model.maxCpuPercent.toFixed(1)}%`
  if (id === 'enterprise_control_inventory') return model.saseCount
  return model.eventCount
}

function widgetDetail(
  id: string,
  model: ReturnType<typeof buildDashboardModel>,
  data: VisibilityData,
  health: { label: string; text: string },
) {
  const reg = dashboardRegistryMap().get(id)
  const base = reg?.description || ''
  if (id === 'platform_status') return `${health.text}. ${base}`
  if (id === 'endpoint_freshness')
    return `${model.offlineDevices} stale of ${model.totalDevices}. ${base}`
  if (id === 'ai_activity') return `${model.aiSignals} AI-shaped findings in window · ${base}`
  if (id === 'detection_pack_coverage')
    return `${model.healthyCollectorPairs} healthy collector pairs (rollout readiness proxy). ${base}`
  if (id === 'agent_performance_budget') {
    const avg = model.avgCpuPercent === null ? 'n/a avg' : `${model.avgCpuPercent.toFixed(1)}% avg CPU`
    const memory = model.maxMemoryRssMb === null ? 'n/a RSS' : `${model.maxMemoryRssMb.toFixed(1)} MB max RSS`
    return `${avg} · ${memory}. ${base}`
  }
  if (id === 'enterprise_control_inventory') return `${model.sase.length} SSE/SASE rows sampled. ${base}`
  return base
}

function Widget({ icon: Icon, title, gradient, value, detail }: { icon: typeof Activity; title: string; gradient: string; value: string | number; detail: string }) {
  return (
    <div className={`bg-gradient-to-br ${gradient} text-white rounded-lg shadow p-4 flex items-center justify-between gap-3 min-w-0`}>
      <div className="min-w-0 flex-1">
        <p className="text-[10px] font-semibold uppercase tracking-wider opacity-80 truncate">{title}</p>
        <p className="mt-1.5 text-2xl font-bold tabular-nums">{value}</p>
        <p className="mt-1 text-xs leading-4 opacity-70 line-clamp-2">{detail}</p>
      </div>
      <div className="shrink-0 bg-white/20 p-2.5 rounded-lg">
        <Icon className="h-5 w-5 text-white" />
      </div>
    </div>
  )
}

function DarkStat({ icon: Icon, label, value }: { icon: typeof Activity; label: string; value: number }) {
  return (
    <div className="rounded-lg bg-white/10 p-3">
      <Icon className="h-4 w-4 text-cyan-200" />
      <div className="mt-2 text-xl font-bold tabular-nums">{value}</div>
      <div className="text-[10px] uppercase tracking-wider text-slate-400">{label}</div>
    </div>
  )
}

function CountPill({ label, value, tone }: { label: string; value: number; tone: 'slate' | 'blue' | 'amber' }) {
  const tones = {
    slate: 'bg-slate-100 text-slate-600',
    blue: 'bg-blue-50 text-blue-700',
    amber: 'bg-amber-50 text-amber-800',
  }
  return <span className={`rounded-md px-2 py-0.5 text-xs font-semibold ${tones[tone]}`}>{label} {value}</span>
}

function StatusChip({ tone, label }: { tone: 'emerald' | 'amber' | 'slate'; label: string }) {
  const tones = {
    emerald: 'border-emerald-200 bg-emerald-50 text-emerald-800',
    amber: 'border-amber-200 bg-amber-50 text-amber-800',
    slate: 'border-slate-200 bg-slate-50 text-slate-700',
  }
  return <span className={`inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold ${tones[tone]}`}>{label}</span>
}

function FreshDot({ active }: { active: boolean }) {
  return <span className={`h-2 w-2 rounded-full shrink-0 ${active ? 'bg-emerald-500' : 'bg-amber-400'}`} />
}

function platformName(value: string) {
  const normalized = value.toLowerCase()
  if (normalized.includes('windows')) return 'Windows'
  if (normalized.includes('linux')) return 'Linux'
  if (normalized.includes('macos') || normalized.includes('darwin')) return 'macOS'
  return value || 'Unknown'
}

function ageFromMs(ms?: number) {
  if (!ms) return 'no data'
  const seconds = Math.max(0, Math.round((Date.now() - ms) / 1000))
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  return `${Math.round(minutes / 60)}h ago`
}
