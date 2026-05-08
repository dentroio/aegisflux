'use client'

import { Suspense, useEffect, useMemo, useState } from 'react'
import type { CSSProperties, FormEvent } from 'react'
import { useSearchParams } from 'next/navigation'
import { AgentsManagementPanel } from '@/components/AgentsManagementPanel'
import { InventoryPanel } from '@/components/InventoryPanel'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import {
  CopyValueButton,
  EmptyState,
  FilterBar,
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
  SlidersHorizontal,
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
  appShell: { display: 'flex', height: '100vh', flexDirection: 'column', overflow: 'hidden' },
  bodyShell: { display: 'flex', minHeight: 0, flex: 1, overflow: 'hidden' },
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
  headerLogo: { display: 'flex', alignItems: 'center', gap: 12, minWidth: 230 },
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
  loginPage: {
    minHeight: '100vh',
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1fr) 440px',
    background: '#020617',
    color: '#fff',
    overflow: 'hidden',
  },
  loginVisual: {
    position: 'relative',
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 56,
    background:
      'radial-gradient(circle at 64% 36%, rgba(14,165,233,0.18), transparent 34%), radial-gradient(circle at 18% 78%, rgba(37,99,235,0.16), transparent 30%), linear-gradient(135deg, #020617 0%, #07111f 56%, #020617 100%)',
  },
  loginGridOverlay: {
    position: 'absolute',
    inset: 0,
    opacity: 0.22,
    backgroundImage:
      'linear-gradient(rgba(14,165,233,0.18) 1px, transparent 1px), linear-gradient(90deg, rgba(14,165,233,0.18) 1px, transparent 1px)',
    backgroundSize: '42px 42px',
  },
  loginBrandBlock: { position: 'relative', zIndex: 1, width: 'min(720px, 92%)' },
  loginHeroArtwork: {
    position: 'relative',
    zIndex: 1,
    width: 'min(780px, 94%)',
    maxHeight: '88vh',
    objectFit: 'contain',
    display: 'block',
    filter: 'drop-shadow(0 30px 70px rgba(14, 165, 233, 0.28))',
  },
  loginWordmark: { margin: 0, fontSize: 74, lineHeight: 1, fontWeight: 900, letterSpacing: 0.2 },
  loginMotto: {
    marginTop: 30,
    display: 'flex',
    gap: 22,
    color: '#e2e8f0',
    fontSize: 20,
    fontWeight: 700,
    letterSpacing: '0.34em',
    textTransform: 'uppercase',
    flexWrap: 'wrap',
  },
  loginSubMotto: {
    marginTop: 24,
    color: '#38bdf8',
    fontSize: 15,
    fontWeight: 700,
    letterSpacing: '0.26em',
    textTransform: 'uppercase',
  },
  loginRule: {
    width: 360,
    maxWidth: '100%',
    height: 2,
    margin: '34px 0',
    background: 'linear-gradient(90deg, transparent, #0ea5e9, #2563eb, transparent)',
  },
  loginStatement: {
    maxWidth: 660,
    margin: 0,
    color: '#cbd5e1',
    fontSize: 18,
    lineHeight: 1.7,
  },
  loginPanelWrap: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 40,
    background: 'linear-gradient(180deg, #f8fafc 0%, #eef6ff 100%)',
  },
  loginCard: {
    width: 'min(380px, 100%)',
    borderRadius: 14,
    border: '1px solid #e2e8f0',
    background: '#fff',
    color: '#0f172a',
    boxShadow: '0 24px 70px rgba(15, 23, 42, 0.16)',
    padding: 30,
  },
  loginCardHeader: { display: 'flex', alignItems: 'center', gap: 12, marginBottom: 26 },
  loginMiniIcon: { width: 42, height: 42, borderRadius: 10, background: '#020617', overflow: 'hidden', flexShrink: 0 },
  loginTitle: { margin: 0, fontSize: 24, fontWeight: 850, letterSpacing: 0 },
  loginCopy: { margin: '8px 0 0', color: '#64748b', fontSize: 14, lineHeight: 1.6 },
  loginForm: { display: 'grid', gap: 14 },
  loginLabel: { display: 'grid', gap: 7, color: '#334155', fontSize: 13, fontWeight: 700 },
  loginInput: {
    width: '100%',
    height: 42,
    borderRadius: 8,
    border: '1px solid #cbd5e1',
    padding: '0 12px',
    color: '#0f172a',
    fontSize: 14,
    outline: 'none',
  },
  loginSubmit: {
    height: 42,
    border: 0,
    borderRadius: 8,
    background: '#2563eb',
    color: '#fff',
    fontSize: 14,
    fontWeight: 800,
    cursor: 'pointer',
  },
  loginError: {
    borderRadius: 8,
    border: '1px solid #fecaca',
    background: '#fef2f2',
    color: '#991b1b',
    padding: '10px 12px',
    fontSize: 13,
    fontWeight: 600,
  },
  loginHint: { marginTop: 16, color: '#64748b', fontSize: 12, lineHeight: 1.55 },
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

const DASHBOARD_WIDGET_STORAGE = 'aegisflux.dashboard.widgets.v1'

type DashboardWidgetDef = {
  id: string
  title: string
  description: string
  dataSource: string
  defaultSize: 'sm' | 'md' | 'lg'
  icon: typeof Activity
}

const DASHBOARD_WIDGET_REGISTRY: DashboardWidgetDef[] = [
  {
    id: 'platform_status',
    title: 'Platform Status',
    description: 'Overall console health from endpoint freshness',
    dataSource: 'visibility.devices + derived health',
    defaultSize: 'md',
    icon: Activity,
  },
  {
    id: 'endpoint_freshness',
    title: 'Endpoint Freshness',
    description: 'Fresh vs total reporting endpoints',
    dataSource: 'visibility.devices last_seen',
    defaultSize: 'md',
    icon: Server,
  },
  {
    id: 'ai_activity',
    title: 'AI Activity',
    description: 'Heuristic AI-related findings',
    dataSource: 'visibility.findings + patterns',
    defaultSize: 'sm',
    icon: Bot,
  },
  {
    id: 'detection_pack_coverage',
    title: 'Detection Pack Coverage',
    description: 'Healthy collector pairs as pack-readiness proxy',
    dataSource: 'visibility events aegis.collector.status',
    defaultSize: 'md',
    icon: ShieldCheck,
  },
  {
    id: 'agent_performance_budget',
    title: 'Agent Performance Budget',
    description: 'CPU/RSS pressure from agent performance stream',
    dataSource: 'visibility events aegis.agent.performance',
    defaultSize: 'md',
    icon: Cpu,
  },
  {
    id: 'enterprise_control_inventory',
    title: 'Enterprise Control Inventory',
    description: 'SSE/SASE component observations',
    dataSource: 'visibility events aegis.sase_component.observed',
    defaultSize: 'sm',
    icon: LockKeyhole,
  },
]

function dashboardRegistryMap(): Map<string, DashboardWidgetDef> {
  return new Map(DASHBOARD_WIDGET_REGISTRY.map((widget) => [widget.id, widget]))
}

function AegisDashboardBody() {
  const searchParams = useSearchParams()
  const panelParam = searchParams.get('panel')
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
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null)
  const [customizeOpen, setCustomizeOpen] = useState(false)
  const [hiddenWidgets, setHiddenWidgets] = useState<string[]>([])
  const [widgetOrder, setWidgetOrder] = useState<string[]>(() => DASHBOARD_WIDGET_REGISTRY.map((widget) => widget.id))
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
    if (!authChecked || !authenticated || typeof window === 'undefined') return undefined
    window.localStorage.setItem(
      DASHBOARD_WIDGET_STORAGE,
      JSON.stringify({ hidden: hiddenWidgets, order: widgetOrder }),
    )
    return undefined
  }, [hiddenWidgets, widgetOrder, authChecked, authenticated])

  useEffect(() => {
    setAuthenticated(window.localStorage.getItem('aegisflux.labAuth') === 'admin')
    setAuthChecked(true)
    try {
      const raw = window.localStorage.getItem(DASHBOARD_WIDGET_STORAGE)
      if (!raw) return undefined
      const parsed = JSON.parse(raw) as { hidden?: string[]; order?: string[] }
      const regIds = new Set(DASHBOARD_WIDGET_REGISTRY.map((w) => w.id))
      if (Array.isArray(parsed.hidden)) setHiddenWidgets(parsed.hidden.filter((id) => regIds.has(id)))
      if (Array.isArray(parsed.order)) {
        const next = parsed.order.filter((id) => regIds.has(id))
        regIds.forEach((id) => {
          if (!next.includes(id)) next.push(id)
        })
        setWidgetOrder(next)
      }
      return undefined
    } catch {
      return undefined
    }
  }, [])

  useEffect(() => {
    if (!authenticated || mainPanel !== 'dashboard') return undefined
    fetchDashboard()
    const interval = setInterval(fetchDashboard, 60000)
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
      const [deviceResponse, events, extensionEvents, saseEvents, collectorEvents, performanceEvents, findings] = await Promise.all([
        fetchJson<{ devices?: DeviceRecord[] }>('/api/visibility/devices?limit=80'),
        fetchJson<{ events?: EventRecord[] }>('/api/visibility/events?limit=120'),
        fetchJson<{ events?: EventRecord[] }>('/api/visibility/events?event_type=aegis.browser_extension.observed&limit=80'),
        fetchJson<{ events?: EventRecord[] }>('/api/visibility/events?event_type=aegis.sase_component.observed&limit=80'),
        fetchJson<{ events?: EventRecord[] }>('/api/visibility/events?event_type=aegis.collector.status&limit=120'),
        fetchJson<{ events?: EventRecord[] }>('/api/visibility/events?event_type=aegis.agent.performance&limit=120'),
        fetchJson<{ findings?: FindingRecord[] }>('/api/visibility/findings?limit=80'),
      ])

      const nextDevices = deviceResponse.devices || []
      setDevices(nextDevices)
      setData({
        events: uniqueEvents([
          ...(events.events || []),
          ...(extensionEvents.events || []),
          ...(saseEvents.events || []),
          ...(collectorEvents.events || []),
          ...(performanceEvents.events || []),
        ]),
        processes: [],
        flows: [],
        dns: [],
        findings: findings.findings || [],
      })
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
      return
    }
    setLoginError('Invalid lab credentials. Use admin/admin.')
  }

  function handleLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    setAuthenticated(false)
    setDevices([])
    setData({ events: [], processes: [], flows: [], dns: [], findings: [] })
  }

  const model = useMemo(() => buildDashboardModel(data, devices), [data, devices])
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
  const regMap = dashboardRegistryMap()
  const visibleWidgets = widgetOrder
    .map((wid) => regMap.get(wid))
    .filter((widget): widget is DashboardWidgetDef => Boolean(widget && !hiddenWidgets.includes(widget.id)))

  if (!authChecked) {
    return (
      <div style={{ ...ui.loginPage, gridTemplateColumns: '1fr', placeItems: 'center' }}>
        <div style={{ color: '#cbd5e1', fontSize: 14, fontWeight: 700 }}>Loading AegisFlux</div>
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
      label: mainPanel === 'dashboard' ? 'Dashboard' : mainPanel === 'agents' ? 'Agents' : 'Inventory',
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
              <main
                className={`mx-auto max-w-[1500px] ${mainPanel === 'dashboard' ? 'px-5 py-6' : 'min-w-0 px-4 py-4'}`}
                style={ui.main}
              >
                {mainPanel === 'dashboard' ? (
                  <DashboardScanSurface
                    data={data}
                    error={error}
                    filteredDevices={filteredDevices}
                    health={health}
                    hiddenWidgets={hiddenWidgets}
                    lastRefresh={lastRefresh}
                    loading={loading}
                    model={model}
                    query={query}
                    regMap={regMap}
                    setCustomizeOpen={setCustomizeOpen}
                    setHiddenWidgets={setHiddenWidgets}
                    setQuery={setQuery}
                    setWidgetOrder={setWidgetOrder}
                    visibleWidgets={visibleWidgets}
                    widgetOrder={widgetOrder}
                    customizeOpen={customizeOpen}
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
              </main>
    </ConsoleShell>
  )
}

export default function AegisDashboard() {
  return (
    <Suspense
      fallback={
        <div
          className="flex min-h-screen items-center justify-center bg-slate-50 text-sm font-semibold text-slate-600"
          style={ui.page}
        >
          Loading console…
        </div>
      }
    >
      <AegisDashboardBody />
    </Suspense>
  )
}

function DashboardScanSurface({
  customizeOpen,
  data,
  error,
  filteredDevices,
  health,
  hiddenWidgets,
  lastRefresh,
  loading,
  model,
  query,
  regMap,
  setCustomizeOpen,
  setHiddenWidgets,
  setQuery,
  setWidgetOrder,
  visibleWidgets,
  widgetOrder,
  onRefresh,
}: {
  customizeOpen: boolean
  data: VisibilityData
  error: string | null
  filteredDevices: DeviceRecord[]
  health: { label: string; tone: 'emerald' | 'amber' | 'slate'; text: string }
  hiddenWidgets: string[]
  lastRefresh: Date | null
  loading: boolean
  model: ReturnType<typeof buildDashboardModel>
  query: string
  regMap: Map<string, DashboardWidgetDef>
  setCustomizeOpen: (updater: (value: boolean) => boolean) => void
  setHiddenWidgets: (updater: (current: string[]) => string[]) => void
  setQuery: (value: string) => void
  setWidgetOrder: (updater: (current: string[]) => string[]) => void
  visibleWidgets: DashboardWidgetDef[]
  widgetOrder: string[]
  onRefresh: () => void
}) {
  return (
    <>
      <WorkbenchHeader
        title="Dashboard"
        subtitle="Adaptive security and real-time protection across the AegisFlux fleet."
        actions={
          <>
            <div className="text-sm text-slate-500" style={ui.mutedText}>
              {lastRefresh ? `Last updated ${lastRefresh.toLocaleTimeString()}` : 'Waiting for refresh'}
            </div>
            <button
              onClick={onRefresh}
              className="inline-flex h-9 items-center gap-2 rounded-md border border-gray-300 bg-white px-3 text-sm font-medium text-slate-700 hover:bg-gray-50"
              style={ui.button}
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </>
        }
      />
      <div className="mb-3">
        <StatusChip tone={health.tone} label={health.text} />
      </div>

      {error && (
        <div className="mb-4 flex items-center gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          <AlertTriangle className="h-4 w-4" />
          {error}
        </div>
      )}

      <DashboardReadinessBand health={health} model={model} />
      <DashboardWidgetControls
        customizeOpen={customizeOpen}
        hiddenWidgets={hiddenWidgets}
        regMap={regMap}
        setCustomizeOpen={setCustomizeOpen}
        setHiddenWidgets={setHiddenWidgets}
        setWidgetOrder={setWidgetOrder}
        widgetOrder={widgetOrder}
      />

      <section className="mb-5 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        {visibleWidgets.map((widget) => (
          <Widget
            key={widget.id}
            icon={widget.icon}
            title={widget.title}
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
    <section className="mb-5 grid gap-4 lg:grid-cols-[minmax(0,1.25fr)_minmax(360px,0.75fr)]">
      <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm" style={ui.card}>
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between" style={ui.heroContent}>
          <div>
            <div className="flex items-center gap-2 text-sm font-semibold text-slate-700" style={ui.sectionKicker}>
              <Activity className="h-4 w-4 text-emerald-600" />
              Platform status
            </div>
            <div className="mt-3 text-4xl font-bold tracking-tight text-slate-950" style={ui.heroStatus}>{health.label}</div>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600" style={ui.heroText}>
              {health.text}. Observe endpoint evidence, adapt detections continuously, and enforce only when controls are proven.
            </p>
          </div>
          <div className="sm:min-w-[360px]">
            <SummaryStrip>
              <KpiTile label="Endpoints" value={model.totalDevices} />
              <KpiTile label="Fresh" value={model.onlineDevices} />
              <KpiTile label="Collectors" value={model.healthyCollectorPairs} />
              <KpiTile label="Max Risk" value={model.maxRisk} />
            </SummaryStrip>
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
  )
}

function DashboardWidgetControls({
  customizeOpen,
  hiddenWidgets,
  regMap,
  setCustomizeOpen,
  setHiddenWidgets,
  setWidgetOrder,
  widgetOrder,
}: {
  customizeOpen: boolean
  hiddenWidgets: string[]
  regMap: Map<string, DashboardWidgetDef>
  setCustomizeOpen: (updater: (value: boolean) => boolean) => void
  setHiddenWidgets: (updater: (current: string[]) => string[]) => void
  setWidgetOrder: (updater: (current: string[]) => string[]) => void
  widgetOrder: string[]
}) {
  return (
    <section>
      <div style={ui.widgetHeader}>
        <h2 style={ui.widgetTitle}>Dashboard widgets</h2>
        <button style={ui.button} onClick={() => setCustomizeOpen((value) => !value)}>
          <SlidersHorizontal className="h-4 w-4" />
          Customize
        </button>
      </div>
      {customizeOpen && (
        <FilterBar>
          {widgetOrder.map((wid, idx) => {
            const widget = regMap.get(wid)
            if (!widget) return null
            return (
              <div key={wid} style={{ ...ui.widgetToggle, flexWrap: 'wrap' }}>
                <label style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
                  <input
                    type="checkbox"
                    checked={!hiddenWidgets.includes(wid)}
                    onChange={() =>
                      setHiddenWidgets((current) =>
                        current.includes(wid)
                          ? current.filter((hid) => hid !== wid)
                          : [...current, wid],
                      )
                    }
                  />
                  <span>{widget.title}</span>
                </label>
                <span style={{ display: 'inline-flex', gap: 6 }}>
                  <button
                    type="button"
                    style={ui.button}
                    disabled={idx === 0}
                    onClick={() =>
                      setWidgetOrder((ord) => {
                        if (idx === 0) return ord
                        const next = [...ord]
                        ;[next[idx - 1], next[idx]] = [next[idx], next[idx - 1]]
                        return next
                      })
                    }
                  >
                    Up
                  </button>
                  <button
                    type="button"
                    style={ui.button}
                    disabled={idx === widgetOrder.length - 1}
                    onClick={() =>
                      setWidgetOrder((ord) => {
                        if (idx >= ord.length - 1) return ord
                        const next = [...ord]
                        ;[next[idx + 1], next[idx]] = [next[idx], next[idx + 1]]
                        return next
                      })
                    }
                  >
                    Down
                  </button>
                </span>
              </div>
            )
          })}
          <span style={{ fontSize: 12, color: '#64748b' }}>Choices persist locally in your browser.</span>
        </FilterBar>
      )}
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
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm" style={ui.cardNoPad}>
        <div className="flex flex-col gap-3 border-b border-slate-200 px-4 py-4 lg:flex-row lg:items-center lg:justify-between" style={ui.panelHeader}>
          <div>
            <h2 className="text-base font-semibold text-slate-950" style={ui.panelTitle}>Endpoint scan list</h2>
            <p className="mt-1 text-sm text-slate-500" style={ui.mutedText}>Compact view only. Use agent detail route for deep investigation.</p>
          </div>
          <div className="relative lg:w-80" style={ui.searchWrap}>
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              className="h-9 w-full rounded-md border border-gray-300 bg-white pl-9 pr-3 text-sm outline-none focus:border-slate-600"
              style={ui.searchInput}
              placeholder="Search endpoint id, OS, or sensor"
            />
          </div>
        </div>
        <div className="divide-y divide-slate-100" style={ui.agentList}>
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
        <div className="border-t border-slate-200 px-4 py-3 text-xs text-slate-500">
          Showing up to 12 endpoints. <a href="/agents" className="font-semibold text-blue-700 hover:text-blue-900">Open Agents workbench for full list and actions.</a>
        </div>
      </div>

      <aside className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm" style={ui.card}>
        <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Next best actions</h3>
        <div className="mt-3 space-y-2 text-sm text-slate-700">
          <a href="/agents" className="block rounded-md border border-slate-200 px-3 py-2 hover:bg-slate-50">Review stale endpoints and collector health</a>
          <a href="/inventory" className="block rounded-md border border-slate-200 px-3 py-2 hover:bg-slate-50">Inspect AI tool and extension inventory</a>
          <a href="/detections" className="block rounded-md border border-slate-200 px-3 py-2 hover:bg-slate-50">Check detection-pack coverage and rollout</a>
          <a href="/control/controls" className="block rounded-md border border-slate-200 px-3 py-2 hover:bg-slate-50">Draft observe-only controls from evidence</a>
          <a href="/operate/events" className="block rounded-md border border-slate-200 px-3 py-2 hover:bg-slate-50">Audit operational events and changes</a>
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
    <div style={ui.loginPage}>
      <section style={ui.loginVisual}>
        <div style={ui.loginGridOverlay} />
        <img src="/aegisflux-login-hero.png" alt="AegisFlux Observe Adapt Enforce" style={ui.loginHeroArtwork} />
      </section>

      <section style={ui.loginPanelWrap}>
        <div style={ui.loginCard}>
          <div style={ui.loginCardHeader}>
            <div style={ui.loginMiniIcon}>
              <img src="/aegisflux-shield.png" alt="" style={ui.logoImage} />
            </div>
            <div>
              <h2 style={ui.loginTitle}>Sign in</h2>
              <p style={ui.loginCopy}>Access the AegisFlux lab console.</p>
            </div>
          </div>

          <form style={ui.loginForm} onSubmit={onSubmit}>
            <label style={ui.loginLabel}>
              Username
              <input
                value={username}
                onChange={(event) => onUsernameChange(event.target.value)}
                autoComplete="username"
                style={ui.loginInput}
              />
            </label>
            <label style={ui.loginLabel}>
              Password
              <input
                value={password}
                onChange={(event) => onPasswordChange(event.target.value)}
                autoComplete="current-password"
                type="password"
                style={ui.loginInput}
              />
            </label>
            {error ? <div style={ui.loginError}>{error}</div> : null}
            <button type="submit" style={ui.loginSubmit}>Sign in</button>
          </form>
          <div style={ui.loginHint}>Lab credentials: admin / admin. This is a local development gate, not production authentication.</div>
        </div>
      </section>
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

function uniqueEvents(events: EventRecord[]) {
  const byId = new Map<string, EventRecord>()
  for (const event of events) byId.set(event.event_id, event)
  return Array.from(byId.values()).sort((left, right) =>
    (right.received_at_ms || right.timestamp_ms) - (left.received_at_ms || left.timestamp_ms),
  )
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
  return data.events.length
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

function DarkStat({ icon: Icon, label, value }: { icon: typeof Activity; label: string; value: number }) {
  return (
    <div className="rounded-lg bg-white/10 p-3" style={ui.darkStat}>
      <Icon className="h-4 w-4 text-cyan-200" />
      <div className="mt-2 text-xl font-semibold" style={ui.darkValue}>{value}</div>
      <div className="text-xs uppercase tracking-wide text-slate-300" style={ui.darkLabel}>{label}</div>
    </div>
  )
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
