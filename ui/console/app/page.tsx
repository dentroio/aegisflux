'use client'

import { useEffect, useMemo, useState } from 'react'
import type { CSSProperties } from 'react'
import {
  Activity,
  AlertTriangle,
  Bell,
  Bot,
  BookText,
  CheckCircle2,
  Chrome,
  Cpu,
  Database,
  Globe2,
  HardDrive,
  LayoutDashboard,
  LockKeyhole,
  LogOut,
  Moon,
  Network,
  Plug,
  RefreshCw,
  Search,
  Server,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Sparkles,
  TerminalSquare,
  UserCircle,
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

const ui: Record<string, CSSProperties> = {
  page: {
    minHeight: '100vh',
    background: '#f8fafc',
    color: '#0f172a',
    fontFamily: 'Inter, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  },
  header: {
    background: '#fff',
    borderBottom: '1px solid #e5e7eb',
    boxShadow: '0 1px 2px rgba(15, 23, 42, 0.05)',
  },
  headerInner: {
    height: 64,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '0 16px',
    gap: 16,
  },
  brandRow: { display: 'flex', alignItems: 'center', gap: 12, minWidth: 0 },
  logoBox: {
    width: 40,
    height: 40,
    borderRadius: 10,
    background: '#020617',
    color: '#fff',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
  },
  brandTitle: { fontSize: 18, fontWeight: 700, lineHeight: '20px' },
  mutedSmall: { fontSize: 12, color: '#64748b' },
  brandTagline: {
    display: 'flex',
    alignItems: 'center',
    gap: 7,
    color: '#0ea5e9',
    fontSize: 10,
    fontWeight: 800,
    lineHeight: '12px',
    letterSpacing: '0.22em',
    textTransform: 'uppercase',
  },
  brandDot: { color: '#0284c7', fontWeight: 900 },
  headerActions: { display: 'flex', alignItems: 'center', gap: 12 },
  button: {
    height: 36,
    display: 'inline-flex',
    alignItems: 'center',
    gap: 8,
    padding: '0 12px',
    border: '1px solid #d1d5db',
    borderRadius: 8,
    background: '#fff',
    color: '#334155',
    fontSize: 14,
    fontWeight: 600,
    cursor: 'pointer',
  },
  main: { maxWidth: 1500, margin: '0 auto', padding: '24px 20px' },
  titleRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-end',
    gap: 16,
    marginBottom: 20,
    flexWrap: 'wrap',
  },
  titleLine: { display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' },
  pageTitle: { margin: 0, fontSize: 28, fontWeight: 800, letterSpacing: 0 },
  subtitle: { margin: '8px 0 0', fontSize: 14, color: '#64748b' },
  mutedText: { fontSize: 14, color: '#64748b' },
  heroGrid: {
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1.25fr) minmax(360px, 0.75fr)',
    gap: 16,
    marginBottom: 20,
  },
  card: {
    background: '#fff',
    border: '1px solid #e2e8f0',
    borderRadius: 12,
    boxShadow: '0 1px 2px rgba(15, 23, 42, 0.05)',
    padding: 20,
  },
  cardNoPad: {
    background: '#fff',
    border: '1px solid #e2e8f0',
    borderRadius: 12,
    boxShadow: '0 1px 2px rgba(15, 23, 42, 0.05)',
    overflow: 'hidden',
  },
  heroContent: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 18 },
  sectionKicker: { display: 'flex', alignItems: 'center', gap: 8, color: '#334155', fontSize: 14, fontWeight: 700 },
  heroStatus: { marginTop: 12, fontSize: 44, lineHeight: 1, fontWeight: 800, letterSpacing: 0 },
  heroText: { margin: '10px 0 0', maxWidth: 720, color: '#475569', fontSize: 14, lineHeight: 1.65 },
  miniGrid: { display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 12 },
  darkCard: {
    background: '#020617',
    color: '#fff',
    border: '1px solid #1e293b',
    borderRadius: 12,
    boxShadow: '0 1px 2px rgba(15, 23, 42, 0.05)',
    padding: 20,
  },
  darkKicker: { display: 'flex', alignItems: 'center', gap: 8, color: '#e2e8f0', fontSize: 14, fontWeight: 700 },
  darkStatGrid: { display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 12, marginTop: 16 },
  darkText: { margin: '16px 0 0', color: '#cbd5e1', fontSize: 14, lineHeight: 1.6 },
  widgetGrid: { display: 'grid', gridTemplateColumns: 'repeat(4, minmax(0, 1fr))', gap: 16, marginBottom: 20 },
  contentGrid: { display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) 430px', gap: 16 },
  panelHeader: { padding: 16, borderBottom: '1px solid #e2e8f0' },
  panelTitle: { margin: 0, fontSize: 16, fontWeight: 700 },
  searchWrap: { position: 'relative', width: 320 },
  searchInput: {
    width: '100%',
    height: 38,
    borderRadius: 8,
    border: '1px solid #d1d5db',
    padding: '0 12px 0 36px',
    fontSize: 14,
    outline: 'none',
  },
  agentList: { maxHeight: 620, overflow: 'auto' },
  agentRow: {
    width: '100%',
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1fr) auto',
    gap: 16,
    padding: '14px 16px',
    border: 0,
    borderBottom: '1px solid #f1f5f9',
    background: '#fff',
    textAlign: 'left',
    cursor: 'pointer',
  },
  agentRowActive: { background: '#eff6ff' },
  minWidth0: { minWidth: 0 },
  inlineRow: { display: 'flex', alignItems: 'center', gap: 8 },
  agentName: { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 14, fontWeight: 700 },
  smallPill: { borderRadius: 6, background: '#f1f5f9', padding: '2px 8px', color: '#475569', fontSize: 12, fontWeight: 600 },
  agentMeta: { marginTop: 4, color: '#64748b', fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
  rowPills: { display: 'flex', alignItems: 'center', gap: 8 },
  detailBody: { padding: 16, display: 'grid', gap: 16 },
  spaceBetween: { display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12 },
  widget: {
    background: '#fff',
    border: '1px solid #e2e8f0',
    borderRadius: 12,
    boxShadow: '0 1px 2px rgba(15, 23, 42, 0.05)',
    padding: 16,
  },
  kpiLabel: { margin: 0, fontSize: 11, fontWeight: 800, letterSpacing: '0.12em', textTransform: 'uppercase', color: '#64748b' },
  kpiValue: { margin: '8px 0 0', fontSize: 26, fontWeight: 800, color: '#0f172a' },
  miniStat: { borderRadius: 10, border: '1px solid #e2e8f0', background: '#f8fafc', padding: 12 },
  miniValue: { marginTop: 4, fontSize: 22, fontWeight: 800, color: '#0f172a' },
  darkStat: { borderRadius: 10, background: 'rgba(255,255,255,0.10)', padding: 12 },
  darkValue: { marginTop: 8, fontSize: 22, fontWeight: 800 },
  darkLabel: { fontSize: 11, color: '#cbd5e1', letterSpacing: '0.08em', textTransform: 'uppercase' },
  detailTitle: { display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, color: '#64748b', fontSize: 11, fontWeight: 800, letterSpacing: '0.08em', textTransform: 'uppercase' },
  stackSmall: { display: 'grid', gap: 8 },
  compactRow: { border: '1px solid #e2e8f0', borderRadius: 8, padding: '10px 12px' },
  compactTitle: { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 14, fontWeight: 600, color: '#1e293b' },
  compactSub: { marginTop: 4, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 12, color: '#64748b' },
  emptyLine: { borderRadius: 8, background: '#f8fafc', padding: '10px 12px', color: '#64748b', fontSize: 14 },
  emptyPanel: { background: '#fff', border: '1px dashed #cbd5e1', borderRadius: 12, padding: 32, textAlign: 'center', color: '#64748b' },
  countPill: { borderRadius: 6, padding: '4px 8px', fontSize: 12, fontWeight: 700 },
  statusChip: { display: 'inline-flex', alignItems: 'center', borderRadius: 999, border: '1px solid', padding: '4px 10px', fontSize: 12, fontWeight: 700 },
  freshDot: { display: 'inline-block', width: 10, height: 10, borderRadius: 999, flexShrink: 0 },
  appShell: { display: 'flex', minHeight: '100vh', flexDirection: 'column' },
  bodyShell: { display: 'flex', minHeight: 0, flex: 1 },
  sidebar: {
    width: 264,
    flexShrink: 0,
    background: '#fff',
    borderRight: '1px solid #e5e7eb',
    display: 'flex',
    flexDirection: 'column',
    overflowY: 'auto',
  },
  sidebarBrand: {
    height: 64,
    display: 'flex',
    alignItems: 'center',
    gap: 12,
    padding: '0 18px',
    borderBottom: '1px solid #e5e7eb',
  },
  sideNav: { padding: 16, display: 'grid', gap: 4 },
  sideGroupLabel: {
    padding: '20px 12px 6px',
    fontSize: 11,
    fontWeight: 800,
    color: '#94a3b8',
    letterSpacing: '0.08em',
    textTransform: 'uppercase',
  },
  sideButton: {
    width: '100%',
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    border: 0,
    borderRadius: 8,
    background: 'transparent',
    color: '#475569',
    padding: '10px 12px',
    fontSize: 14,
    fontWeight: 600,
    textAlign: 'left',
    cursor: 'pointer',
  },
  sideButtonActive: { background: '#2563eb', color: '#fff' },
  sideButtonMuted: { color: '#475569', cursor: 'default' },
  sideFooter: { marginTop: 'auto', padding: 14, borderTop: '1px solid #e5e7eb' },
  sideFooterCard: { borderRadius: 10, border: '1px solid #bae6fd', background: '#f0f9ff', padding: 12 },
  contentArea: { minWidth: 0, flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' },
  scrollArea: { flex: 1, minHeight: 0, overflow: 'auto' },
  widgetHeader: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, marginBottom: 12 },
  widgetTitle: { margin: 0, fontSize: 16, fontWeight: 750, color: '#0f172a' },
  customizePanel: {
    marginBottom: 12,
    display: 'flex',
    flexWrap: 'wrap',
    gap: 8,
    border: '1px solid #e2e8f0',
    background: '#fff',
    borderRadius: 10,
    padding: 10,
  },
  widgetToggle: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 7,
    border: '1px solid #e2e8f0',
    borderRadius: 999,
    padding: '6px 10px',
    fontSize: 13,
    color: '#475569',
    background: '#f8fafc',
  },
  headerLogo: { display: 'flex', alignItems: 'center', gap: 12, minWidth: 260 },
  headerLogoMark: {
    width: 38,
    height: 38,
    borderRadius: 9,
    background: '#020617',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
  },
  logoImage: { width: '100%', height: '100%', objectFit: 'cover', display: 'block' },
  wordmarkText: {
    margin: 0,
    fontSize: 21,
    lineHeight: '23px',
    fontWeight: 850,
    letterSpacing: 0.4,
    color: '#0f172a',
  },
  wordmarkFlux: {
    color: '#2563eb',
  },
  headerSearch: {
    height: 34,
    display: 'inline-flex',
    alignItems: 'center',
    gap: 8,
    border: 0,
    borderRadius: 8,
    background: '#f1f5f9',
    color: '#64748b',
    padding: '0 10px',
    fontSize: 14,
  },
  keycap: {
    border: '1px solid #cbd5e1',
    borderRadius: 5,
    background: '#fff',
    color: '#94a3b8',
    padding: '1px 5px',
    fontSize: 11,
    fontWeight: 700,
  },
  iconButton: {
    width: 36,
    height: 36,
    border: 0,
    borderRadius: 8,
    background: 'transparent',
    color: '#64748b',
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerStatus: { display: 'inline-flex', alignItems: 'center', gap: 8, paddingLeft: 12, borderLeft: '1px solid #e5e7eb' },
  userBadge: { display: 'inline-flex', alignItems: 'center', gap: 8, paddingLeft: 12, borderLeft: '1px solid #e5e7eb' },
  rolePill: { border: '1px solid #bfdbfe', background: '#eff6ff', color: '#1d4ed8', borderRadius: 5, padding: '2px 6px', fontSize: 11, fontWeight: 700 },
  breadcrumb: {
    height: 38,
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    padding: '0 20px',
    borderBottom: '1px solid #f1f5f9',
    background: '#fff',
    color: '#64748b',
    fontSize: 13,
  },
}

const pillStyles: Record<'slate' | 'blue' | 'amber', CSSProperties> = {
  slate: { background: '#f1f5f9', color: '#475569' },
  blue: { background: '#eff6ff', color: '#1d4ed8' },
  amber: { background: '#fffbeb', color: '#92400e' },
}

const statusStyles: Record<'emerald' | 'amber' | 'slate', CSSProperties> = {
  emerald: { borderColor: '#a7f3d0', background: '#ecfdf5', color: '#065f46' },
  amber: { borderColor: '#fde68a', background: '#fffbeb', color: '#92400e' },
  slate: { borderColor: '#e2e8f0', background: '#f8fafc', color: '#334155' },
}

const navGroups = [
  {
    label: 'Overview',
    items: [
      { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard, active: true },
    ],
  },
  {
    label: 'Discover',
    items: [
      { id: 'agents', label: 'Agents', icon: Server },
      { id: 'inventory', label: 'Inventory', icon: Database },
      { id: 'activity', label: 'AI Activity', icon: Bot },
    ],
  },
  {
    label: 'Analyze',
    items: [
      { id: 'evidence', label: 'Evidence Graph', icon: Network },
      { id: 'detections', label: 'Detection Packs', icon: Sparkles },
      { id: 'activity-log', label: 'Findings', icon: Activity },
    ],
  },
  {
    label: 'Control',
    items: [
      { id: 'controls', label: 'Controls', icon: ShieldCheck },
      { id: 'simulation', label: 'Policy Simulation', icon: TerminalSquare },
    ],
  },
  {
    label: 'Operate',
    items: [
      { id: 'monitoring', label: 'Monitoring', icon: Cpu },
      { id: 'reports', label: 'Reports', icon: BookText },
    ],
  },
  {
    label: 'Configure',
    items: [
      { id: 'connectors', label: 'Connectors', icon: Plug },
      { id: 'settings', label: 'Settings', icon: Settings },
    ],
  },
] as const

const widgetCatalog = [
  { id: 'evidence', icon: Database, title: 'Evidence', detail: 'process, flow, and DNS records' },
  { id: 'browser', icon: Chrome, title: 'Browser Surface', detail: 'extension and profile observations' },
  { id: 'sase', icon: LockKeyhole, title: 'Enterprise Controls', detail: 'SSE/SASE endpoint components' },
  { id: 'budget', icon: Cpu, title: 'Agent Budget', detail: 'near-zero idle, bounded collectors' },
] as const

export default function AegisDashboard() {
  const [devices, setDevices] = useState<DeviceRecord[]>([])
  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [query, setQuery] = useState('')
  const [data, setData] = useState<VisibilityData>({ events: [], processes: [], flows: [], dns: [], findings: [] })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null)
  const [customizeOpen, setCustomizeOpen] = useState(false)
  const [hiddenWidgets, setHiddenWidgets] = useState<string[]>([])

  useEffect(() => {
    fetchDashboard()
    const interval = setInterval(fetchDashboard, 15000)
    return () => clearInterval(interval)
  }, [])

  async function fetchJson<T>(path: string): Promise<T> {
    const response = await fetch(path, { cache: 'no-store' })
    if (!response.ok) throw new Error(`${path} returned HTTP ${response.status}`)
    return response.json()
  }

  async function fetchDashboard() {
    try {
      setLoading(true)
      setError(null)
      const [deviceResponse, events, extensionEvents, saseEvents, collectorEvents, processes, flows, dns, findings] = await Promise.all([
        fetchJson<{ devices?: DeviceRecord[] }>('/api/visibility/devices?limit=100'),
        fetchJson<{ events?: EventRecord[] }>('/api/visibility/events?limit=240'),
        fetchJson<{ events?: EventRecord[] }>('/api/visibility/events?event_type=aegis.browser_extension.observed&limit=160'),
        fetchJson<{ events?: EventRecord[] }>('/api/visibility/events?event_type=aegis.sase_component.observed&limit=160'),
        fetchJson<{ events?: EventRecord[] }>('/api/visibility/events?event_type=aegis.collector.status&limit=220'),
        fetchJson<{ processes?: ProcessRecord[] }>('/api/visibility/processes?limit=180'),
        fetchJson<{ flows?: FlowRecord[] }>('/api/visibility/flows?limit=180'),
        fetchJson<{ dns?: DnsRecord[] }>('/api/visibility/dns?limit=180'),
        fetchJson<{ findings?: FindingRecord[] }>('/api/visibility/findings?limit=120'),
      ])

      const nextDevices = deviceResponse.devices || []
      setDevices(nextDevices)
      setSelectedDeviceId((current) => current || nextDevices[0]?.device_id || '')
      setData({
        events: uniqueEvents([
          ...(events.events || []),
          ...(extensionEvents.events || []),
          ...(saseEvents.events || []),
          ...(collectorEvents.events || []),
        ]),
        processes: processes.processes || [],
        flows: flows.flows || [],
        dns: dns.dns || [],
        findings: findings.findings || [],
      })
      setLastRefresh(new Date())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load AegisFlux dashboard')
    } finally {
      setLoading(false)
    }
  }

  const model = useMemo(() => buildDashboardModel(data, devices), [data, devices])
  const selectedDevice = devices.find((device) => device.device_id === selectedDeviceId) || devices[0]
  const selectedDetail = useMemo(
    () => selectedDevice ? buildDeviceDetail(selectedDevice, data, model) : null,
    [selectedDevice, data, model],
  )
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
  const visibleWidgets = widgetCatalog.filter((widget) => !hiddenWidgets.includes(widget.id))

  return (
    <div className="min-h-screen bg-gray-50 text-slate-900" style={ui.page}>
      <div style={ui.appShell}>
        <header className="border-b border-gray-200 bg-white shadow-sm" style={ui.header}>
          <div className="flex h-16 items-center justify-between px-5" style={ui.headerInner}>
            <div style={ui.headerLogo}>
              <div style={ui.headerLogoMark}>
                <img src="/aegisflux-icon.svg" alt="AegisFlux" style={ui.logoImage} />
              </div>
              <div style={{ display: 'grid', gap: 1 }}>
                <div style={ui.wordmarkText}>
                  Aegis<span style={ui.wordmarkFlux}>Flux</span>
                </div>
                <div style={ui.brandTagline}>
                  <span>Observe</span>
                  <span style={ui.brandDot}>.</span>
                  <span>Adapt</span>
                  <span style={ui.brandDot}>.</span>
                  <span>Enforce</span>
                  <span style={ui.brandDot}>.</span>
                </div>
              </div>
            </div>

            <div className="flex items-center gap-3" style={ui.headerActions}>
              <button style={ui.headerSearch} title="Search">
                <Search className="h-4 w-4" />
                <span>Search</span>
                <span style={ui.keycap}>Cmd K</span>
              </button>
              <button style={ui.iconButton} title="Notifications">
                <Bell className="h-5 w-5" />
              </button>
              <button style={ui.iconButton} title="Theme">
                <Moon className="h-5 w-5" />
              </button>
              <button style={ui.iconButton} title="Documentation">
                <BookText className="h-5 w-5" />
              </button>
              <button style={ui.iconButton} title="AI Assistant">
                <Bot className="h-5 w-5" />
              </button>
              <div style={ui.headerStatus}>
                <FreshDot active={health.tone === 'emerald'} />
                <span style={{ fontSize: 14, fontWeight: 650, color: health.tone === 'emerald' ? '#334155' : '#92400e' }}>
                  {health.label}
                </span>
              </div>
              <div style={ui.userBadge}>
                <UserCircle className="h-5 w-5 text-slate-500" />
                <span style={{ fontSize: 14, fontWeight: 650, color: '#334155' }}>operator</span>
                <span style={ui.rolePill}>Admin</span>
                <LogOut className="h-4 w-4 text-slate-400" />
              </div>
            </div>
          </div>
        </header>

        <div style={ui.bodyShell}>
          <aside style={ui.sidebar}>
            <nav style={ui.sideNav}>
              {navGroups.map((group) => (
                <div key={group.label}>
                  <div style={ui.sideGroupLabel}>{group.label}</div>
                  {group.items.map((item) => {
                    const Icon = item.icon
                    const active = 'active' in item && item.active
                    return (
                      <button
                        key={item.id}
                        type="button"
                        style={{
                          ...ui.sideButton,
                          ...(active ? ui.sideButtonActive : ui.sideButtonMuted),
                        }}
                      >
                        <Icon className="h-4 w-4" />
                        {item.label}
                      </button>
                    )
                  })}
                </div>
              ))}
            </nav>
            <div style={ui.sideFooter}>
              <div style={ui.sideFooterCard}>
                <div style={{ fontSize: 13, fontWeight: 750, color: '#075985' }}>Observe-only</div>
                <div style={{ marginTop: 4, fontSize: 12, lineHeight: 1.45, color: '#0369a1' }}>
                  Controls remain staged until approval and rollback are ready.
                </div>
              </div>
            </div>
          </aside>

          <div style={ui.contentArea}>
            <div style={ui.breadcrumb}>
              <span>AegisFlux</span>
              <span>/</span>
              <span style={{ color: '#0f172a', fontWeight: 650 }}>Dashboard</span>
            </div>
            <div style={ui.scrollArea}>
              <main className="mx-auto max-w-[1500px] px-5 py-6" style={ui.main}>
                <div className="mb-5 flex flex-col gap-3 md:flex-row md:items-end md:justify-between" style={ui.titleRow}>
                  <div>
                    <div className="flex flex-wrap items-center gap-2" style={ui.titleLine}>
                      <h1 className="text-2xl font-bold text-slate-950" style={ui.pageTitle}>Dashboard</h1>
                      <StatusChip tone={health.tone} label={health.text} />
                    </div>
                    <p className="mt-2 text-sm text-slate-500" style={ui.subtitle}>
                      Adaptive security and real-time protection across the AegisFlux fleet.
                    </p>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
                    <div className="text-sm text-slate-500" style={ui.mutedText}>
                      {lastRefresh ? `Last updated ${lastRefresh.toLocaleTimeString()}` : 'Waiting for refresh'}
                    </div>
                    <button
                      onClick={fetchDashboard}
                      className="inline-flex h-9 items-center gap-2 rounded-md border border-gray-300 bg-white px-3 text-sm font-medium text-slate-700 hover:bg-gray-50"
                      style={ui.button}
                    >
                      <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                      Refresh
                    </button>
                  </div>
                </div>

                {error && (
                  <div className="mb-4 flex items-center gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
                    <AlertTriangle className="h-4 w-4" />
                    {error}
                  </div>
                )}

        <section className="mb-5 grid gap-4 lg:grid-cols-[minmax(0,1.25fr)_minmax(360px,0.75fr)]" style={ui.heroGrid}>
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm" style={ui.card}>
            <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between" style={ui.heroContent}>
              <div>
                <div className="flex items-center gap-2 text-sm font-semibold text-slate-700" style={ui.sectionKicker}>
                  <Activity className="h-4 w-4 text-emerald-600" />
                  Platform status
                </div>
                <div className="mt-3 text-4xl font-bold tracking-tight text-slate-950" style={ui.heroStatus}>{health.label}</div>
                <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600" style={ui.heroText}>{health.text}. Observe endpoint evidence, adapt detections continuously, and enforce only when controls are proven.</p>
              </div>
              <div className="grid grid-cols-2 gap-3 sm:min-w-[360px]" style={ui.miniGrid}>
                <MiniStat label="Endpoints" value={model.totalDevices} />
                <MiniStat label="Fresh" value={model.onlineDevices} />
                <MiniStat label="Collectors" value={model.healthyCollectorPairs} />
                <MiniStat label="Max Risk" value={model.maxRisk} />
              </div>
            </div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-slate-950 p-5 text-white shadow-sm" style={ui.darkCard}>
            <div className="flex items-center gap-2 text-sm font-semibold text-slate-200" style={ui.darkKicker}>
              <Sparkles className="h-4 w-4 text-cyan-300" />
              Signal focus
            </div>
            <div className="mt-4 grid grid-cols-3 gap-3" style={ui.darkStatGrid}>
              <DarkStat icon={Bot} label="AI" value={model.aiSignals} />
              <DarkStat icon={Chrome} label="Ext" value={model.extensionCount} />
              <DarkStat icon={LockKeyhole} label="SASE" value={model.saseCount} />
            </div>
            <p className="mt-4 text-sm leading-6 text-slate-300" style={ui.darkText}>
              The next product leap is turning these signals into an Agent Bill of Materials per endpoint.
            </p>
          </div>
        </section>

        <section>
          <div style={ui.widgetHeader}>
            <h2 style={ui.widgetTitle}>Dashboard widgets</h2>
            <button style={ui.button} onClick={() => setCustomizeOpen((value) => !value)}>
              <SlidersHorizontal className="h-4 w-4" />
              Customize
            </button>
          </div>
          {customizeOpen && (
            <div style={ui.customizePanel}>
              {widgetCatalog.map((widget) => (
                <label key={widget.id} style={ui.widgetToggle}>
                  <input
                    type="checkbox"
                    checked={!hiddenWidgets.includes(widget.id)}
                    onChange={() => setHiddenWidgets((current) =>
                      current.includes(widget.id)
                        ? current.filter((id) => id !== widget.id)
                        : [...current, widget.id],
                    )}
                  />
                  {widget.title}
                </label>
              ))}
            </div>
          )}
        </section>

        <section className="mb-5 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4" style={ui.widgetGrid}>
          {visibleWidgets.map((widget) => (
            <Widget
              key={widget.id}
              icon={widget.icon}
              title={widget.title}
              value={widgetValue(widget.id, model, data)}
              detail={widget.id === 'evidence' ? `${data.processes.length} process, ${data.flows.length} flow, ${data.dns.length} DNS records` : widget.detail}
            />
          ))}
        </section>

        <section className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_430px]" style={ui.contentGrid}>
          <div className="rounded-xl border border-slate-200 bg-white shadow-sm" style={ui.cardNoPad}>
            <div className="flex flex-col gap-3 border-b border-slate-200 px-4 py-4 lg:flex-row lg:items-center lg:justify-between" style={ui.panelHeader}>
              <div>
                <h2 className="text-base font-semibold text-slate-950" style={ui.panelTitle}>Agents</h2>
                <p className="mt-1 text-sm text-slate-500" style={ui.mutedText}>Select an endpoint to dig into coverage, evidence, and inventory.</p>
              </div>
              <div className="relative lg:w-80" style={ui.searchWrap}>
                <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                <input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  className="h-9 w-full rounded-md border border-gray-300 bg-white pl-9 pr-3 text-sm outline-none focus:border-slate-600"
                  style={ui.searchInput}
                  placeholder="Search agents"
                />
              </div>
            </div>
            <AgentList
              devices={filteredDevices}
              selectedDeviceId={selectedDevice?.device_id || ''}
              model={model}
              onSelect={setSelectedDeviceId}
            />
          </div>

          <AgentDetailPanel detail={selectedDetail} />
        </section>
      </main>
        </div>
      </div>
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
    healthyCollectorPairs: new Set(collectorStatuses.filter((status) => status.status === 'healthy').map((status) => `${status.device_id}:${status.collector}`)).size,
  }
}

function buildDeviceDetail(device: DeviceRecord, data: VisibilityData, model: ReturnType<typeof buildDashboardModel>) {
  const deviceId = device.device_id
  return {
    device,
    processes: data.processes.filter((record) => record.device_id === deviceId),
    flows: data.flows.filter((record) => record.device_id === deviceId),
    dns: data.dns.filter((record) => record.device_id === deviceId),
    findings: data.findings.filter((record) => record.device_id === deviceId),
    extensions: model.extensions.filter((record) => record.device_id === deviceId),
    sase: model.sase.filter((record) => record.device_id === deviceId),
    collectors: model.collectorStatuses.filter((record) => record.device_id === deviceId),
  }
}

function uniqueEvents(events: EventRecord[]) {
  const byId = new Map<string, EventRecord>()
  for (const event of events) byId.set(event.event_id, event)
  return Array.from(byId.values()).sort((left, right) =>
    (right.received_at_ms || right.timestamp_ms) - (left.received_at_ms || left.timestamp_ms),
  )
}

function widgetValue(id: string, model: ReturnType<typeof buildDashboardModel>, data: VisibilityData) {
  if (id === 'evidence') return model.eventCount
  if (id === 'browser') return model.extensionCount
  if (id === 'sase') return model.saseCount
  if (id === 'budget') return 'Low'
  return data.events.length
}

function AgentList({
  devices,
  selectedDeviceId,
  model,
  onSelect,
}: {
  devices: DeviceRecord[]
  selectedDeviceId: string
  model: ReturnType<typeof buildDashboardModel>
  onSelect: (deviceId: string) => void
}) {
  return (
    <div className="divide-y divide-slate-100" style={ui.agentList}>
      {devices.length === 0 ? (
        <div className="px-4 py-12 text-center text-sm text-slate-500">No agents match this filter.</div>
      ) : devices.map((device) => {
        const active = Date.now() - device.last_seen_ms < 5 * 60 * 1000
        const findings = Number(device.event_type_count?.['aegis.risk_finding.created'] || 0)
        const extensions = model.extensions.filter((record) => record.device_id === device.device_id).length
        return (
          <button
            key={device.device_id}
            onClick={() => onSelect(device.device_id)}
            className={`grid w-full grid-cols-[minmax(0,1fr)_auto] gap-4 px-4 py-3 text-left transition-colors hover:bg-slate-50 ${
              selectedDeviceId === device.device_id ? 'bg-blue-50/60' : ''
            }`}
            style={{
              ...ui.agentRow,
              ...(selectedDeviceId === device.device_id ? ui.agentRowActive : null),
            }}
          >
            <div className="min-w-0" style={ui.minWidth0}>
              <div className="flex items-center gap-2" style={ui.inlineRow}>
                <FreshDot active={active} />
                <div className="truncate text-sm font-semibold text-slate-950" style={ui.agentName}>{device.device_id}</div>
                <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600" style={ui.smallPill}>{platformName(device.source || device.device_id)}</span>
              </div>
              <div className="mt-1 truncate text-xs text-slate-500" style={ui.agentMeta}>
                {device.sensor_version} · last seen {ageFromMs(device.last_seen_ms)}
              </div>
            </div>
            <div className="flex items-center gap-2 text-xs" style={ui.rowPills}>
              <CountPill label="Find" value={findings} tone={findings ? 'amber' : 'slate'} />
              <CountPill label="Ext" value={extensions} tone="blue" />
            </div>
          </button>
        )
      })}
    </div>
  )
}

function AgentDetailPanel({ detail }: { detail: ReturnType<typeof buildDeviceDetail> | null }) {
  if (!detail) {
    return (
      <div className="rounded-xl border border-dashed border-slate-300 bg-white p-8 text-center text-sm text-slate-500" style={ui.emptyPanel}>
        Select an agent to view detail.
      </div>
    )
  }
  const { device } = detail
  const active = Date.now() - device.last_seen_ms < 5 * 60 * 1000

  return (
    <aside className="rounded-xl border border-slate-200 bg-white shadow-sm" style={ui.cardNoPad}>
      <div className="border-b border-slate-200 p-4" style={ui.panelHeader}>
        <div className="flex items-start justify-between gap-3" style={ui.spaceBetween}>
          <div className="min-w-0" style={ui.minWidth0}>
            <div className="flex items-center gap-2" style={ui.inlineRow}>
              <FreshDot active={active} />
              <h2 className="truncate text-base font-semibold text-slate-950" style={ui.panelTitle}>{device.device_id}</h2>
            </div>
            <p className="mt-1 text-sm text-slate-500" style={ui.mutedText}>{platformName(device.source)} · {device.sensor_version}</p>
          </div>
          <StatusChip tone={active ? 'emerald' : 'amber'} label={active ? 'Fresh' : 'Stale'} />
        </div>
      </div>

      <div className="space-y-4 p-4" style={ui.detailBody}>
        <div className="grid grid-cols-2 gap-3" style={ui.miniGrid}>
          <MiniStat label="Processes" value={detail.processes.length} />
          <MiniStat label="Flows" value={detail.flows.length} />
          <MiniStat label="DNS" value={detail.dns.length} />
          <MiniStat label="Findings" value={detail.findings.length} />
        </div>

        <DetailSection icon={Activity} title="Collector health">
          {detail.collectors.length === 0 ? (
            <EmptyLine text="No collector status in current window." />
          ) : detail.collectors.slice(0, 6).map((collector) => (
            <div key={`${collector.collector}-${collector.received_at_ms}`} className="rounded-md border border-slate-200 px-3 py-2">
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm font-medium text-slate-800">{collector.collector}</span>
                <StatusChip tone={collector.status === 'healthy' ? 'emerald' : 'amber'} label={collector.status} />
              </div>
              <p className="mt-1 text-xs leading-5 text-slate-500">{collector.message}</p>
            </div>
          ))}
        </DetailSection>

        <DetailSection icon={Chrome} title="Browser inventory">
          {detail.extensions.length === 0 ? (
            <EmptyLine text="No extensions observed in current window." />
          ) : detail.extensions.slice(0, 5).map((extension) => (
            <CompactRow key={`${extension.extension_id}-${extension.profile}`} title={extension.name} subtitle={`${extension.browser} · ${extension.profile}`} />
          ))}
        </DetailSection>

        <DetailSection icon={LockKeyhole} title="SSE/SASE">
          {detail.sase.length === 0 ? (
            <EmptyLine text="No SSE/SASE component detected." />
          ) : detail.sase.slice(0, 5).map((component) => (
            <CompactRow key={`${component.component_type}-${component.name}`} title={`${component.vendor} · ${component.name}`} subtitle={component.product} />
          ))}
        </DetailSection>

        <DetailSection icon={Globe2} title="Recent AI destinations">
          {detail.dns.filter((record) => /chatgpt|openai|anthropic|claude|gemini|copilot|mistral/i.test(record.query)).slice(0, 5).map((record) => (
            <CompactRow key={record.query} title={record.query} subtitle={record.correlation_method || 'DNS/browser evidence'} />
          ))}
          {detail.dns.filter((record) => /chatgpt|openai|anthropic|claude|gemini|copilot|mistral/i.test(record.query)).length === 0 && (
            <EmptyLine text="No AI destination in current DNS window." />
          )}
        </DetailSection>
      </div>
    </aside>
  )
}

function Widget({ icon: Icon, title, value, detail }: { icon: typeof Activity; title: string; value: string | number; detail: string }) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm" style={ui.widget}>
      <div className="flex items-start justify-between gap-3" style={ui.spaceBetween}>
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500" style={ui.kpiLabel}>{title}</p>
          <p className="mt-2 text-2xl font-semibold text-slate-950" style={ui.kpiValue}>{value}</p>
          <p className="mt-2 text-sm leading-5 text-slate-500" style={ui.mutedText}>{detail}</p>
        </div>
        <Icon className="h-5 w-5 text-slate-400" />
      </div>
    </div>
  )
}

function MiniStat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-3" style={ui.miniStat}>
      <div className="text-xs font-semibold uppercase tracking-wide text-slate-500" style={ui.kpiLabel}>{label}</div>
      <div className="mt-1 text-xl font-semibold text-slate-950" style={ui.miniValue}>{value}</div>
    </div>
  )
}

function DarkStat({ icon: Icon, label, value }: { icon: typeof Activity; label: string; value: number }) {
  return (
    <div className="rounded-lg bg-white/10 p-3" style={ui.darkStat}>
      <Icon className="h-4 w-4 text-cyan-200" />
      <div className="mt-2 text-xl font-semibold" style={ui.darkValue}>{value}</div>
      <div className="text-xs uppercase tracking-wide text-slate-300" style={ui.darkLabel}>{label}</div>
    </div>
  )
}

function DetailSection({ icon: Icon, title, children }: { icon: typeof Activity; title: string; children: React.ReactNode }) {
  return (
    <section>
      <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-slate-500" style={ui.detailTitle}>
        <Icon className="h-4 w-4" />
        {title}
      </div>
      <div className="space-y-2" style={ui.stackSmall}>{children}</div>
    </section>
  )
}

function CompactRow({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div className="rounded-md border border-slate-200 px-3 py-2" style={ui.compactRow}>
      <div className="truncate text-sm font-medium text-slate-800" style={ui.compactTitle}>{title}</div>
      <div className="mt-1 truncate text-xs text-slate-500" style={ui.compactSub}>{subtitle}</div>
    </div>
  )
}

function EmptyLine({ text }: { text: string }) {
  return <div className="rounded-md bg-slate-50 px-3 py-2 text-sm text-slate-500" style={ui.emptyLine}>{text}</div>
}

function CountPill({ label, value, tone }: { label: string; value: number; tone: 'slate' | 'blue' | 'amber' }) {
  const tones = {
    slate: 'bg-slate-100 text-slate-600',
    blue: 'bg-blue-50 text-blue-700',
    amber: 'bg-amber-50 text-amber-800',
  }
  return <span className={`rounded-md px-2 py-1 font-medium ${tones[tone]}`} style={{ ...ui.countPill, ...pillStyles[tone] }}>{label} {value}</span>
}

function StatusChip({ tone, label }: { tone: 'emerald' | 'amber' | 'slate'; label: string }) {
  const tones = {
    emerald: 'border-emerald-200 bg-emerald-50 text-emerald-800',
    amber: 'border-amber-200 bg-amber-50 text-amber-800',
    slate: 'border-slate-200 bg-slate-50 text-slate-700',
  }
  return <span className={`inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold ${tones[tone]}`} style={{ ...ui.statusChip, ...statusStyles[tone] }}>{label}</span>
}

function FreshDot({ active }: { active: boolean }) {
  return <span className={`h-2.5 w-2.5 rounded-full ${active ? 'bg-emerald-500' : 'bg-amber-500'}`} style={{ ...ui.freshDot, background: active ? '#10b981' : '#f59e0b' }} />
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
