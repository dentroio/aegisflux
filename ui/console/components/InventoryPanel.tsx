'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Bot,
  Chrome,
  Database,
  LockKeyhole,
  Monitor,
  RefreshCw,
  Server,
  Shield,
  Terminal,
  Wrench,
  Search,
} from 'lucide-react'
import {
  BoundedTable,
  DetailModal,
  EmptyState,
  FilterBar,
  KpiTile,
  SummaryStrip,
  WorkbenchHeader,
} from '@/components/workbench/primitives'
import { formatCommandLine, formatHash, formatHostname, formatPath, formatRelativeAge } from '@/shared/formatting'

type EventRecord = {
  event_id: string
  event_type: string
  timestamp_ms: number
  received_at_ms?: number
  source?: string
  device_id: string
  payload: Record<string, unknown>
}

type DnsRecord = {
  device_id?: string
  query?: string
  answers?: string[]
  correlation_method?: string
}

type ProcessRecord = {
  device_id?: string
  name?: string
  path?: string
  command_line?: string
}

type FindingRecord = {
  device_id?: string
  title?: string
  classification?: string
  detected_patterns?: string[]
}

type DeviceRecord = {
  device_id: string
  source?: string
  last_seen_ms?: number
}

const AI_PATTERN = /chatgpt|openai|anthropic|claude|gemini|copilot|mistral|ollama|litellm|vllm|mcp|modelcontextprotocol/i

function matchesInventoryQuery(row: unknown, needle: string) {
  return !needle || JSON.stringify(row).toLowerCase().includes(needle)
}

async function fetchJson<T>(path: string, fallback: T): Promise<T> {
  try {
    const response = await fetch(path, { cache: 'no-store' })
    if (!response.ok) return fallback
    return await response.json()
  } catch {
    return fallback
  }
}

export function InventoryPanel({
  deviceFilter,
  embedded = false,
}: {
  deviceFilter: string
  embedded?: boolean
}) {
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [devices, setDevices] = useState<DeviceRecord[]>([])
  const [extensionEvents, setExtensionEvents] = useState<EventRecord[]>([])
  const [saseEvents, setSaseEvents] = useState<EventRecord[]>([])
  const [dns, setDns] = useState<DnsRecord[]>([])
  const [processes, setProcesses] = useState<ProcessRecord[]>([])
  const [findings, setFindings] = useState<FindingRecord[]>([])
  const [category, setCategory] = useState<'extensions' | 'ai_tools' | 'local_models' | 'sase' | 'unknown'>('extensions')
  const [query, setQuery] = useState('')
  const [detailModal, setDetailModal] = useState<{ title: string; payload: unknown } | null>(null)

  const load = useCallback(async () => {
    setRefreshing(true)
    const res = await fetch('/api/visibility/summary/inventory', { cache: 'no-store' })
    if (!res.ok) {
      const [deviceRes, extRes, saseRes, dnsRes, procRes, findRes] = await Promise.all([
        fetchJson<{ devices?: DeviceRecord[] }>('/api/visibility/devices?limit=180', {}),
        fetchJson<{ events?: EventRecord[] }>(
          '/api/visibility/events?event_type=aegis.browser_extension.observed&limit=220',
          {},
        ),
        fetchJson<{ events?: EventRecord[] }>(
          '/api/visibility/events?event_type=aegis.sase_component.observed&limit=220',
          {},
        ),
        fetchJson<{ dns?: DnsRecord[]; observations?: DnsRecord[] }>('/api/visibility/dns?limit=220', {}),
        fetchJson<{ processes?: ProcessRecord[] }>('/api/visibility/processes?limit=220', {}),
        fetchJson<{ findings?: FindingRecord[] }>('/api/visibility/findings?limit=120', {}),
      ])
      setDevices(deviceRes.devices || [])
      setExtensionEvents(extRes.events || [])
      setSaseEvents(saseRes.events || [])
      setDns(dnsRes.dns || dnsRes.observations || [])
      setProcesses(procRes.processes || [])
      setFindings(findRes.findings || [])
      setLoading(false)
      setRefreshing(false)
      return
    }
    const bundle = await res.json()
    const mapVisEvent = (e: Record<string, unknown>): EventRecord => {
      let payload = e.payload as Record<string, unknown> | string | undefined
      if (typeof payload === 'string') {
        try {
          payload = JSON.parse(payload) as Record<string, unknown>
        } catch {
          payload = {}
        }
      }
      return {
        event_id: String(e.event_id || ''),
        event_type: String(e.event_type || ''),
        timestamp_ms: Number(e.timestamp_ms || 0),
        received_at_ms: e.received_at_ms !== undefined ? Number(e.received_at_ms) : undefined,
        source: e.source !== undefined ? String(e.source) : undefined,
        device_id: String(e.device_id || ''),
        payload: (payload && typeof payload === 'object' ? payload : {}) as Record<string, unknown>,
      }
    }
    setDevices((bundle.devices || []) as DeviceRecord[])
    setExtensionEvents((bundle.events_ext || []).map(mapVisEvent))
    setSaseEvents((bundle.events_sase || []).map(mapVisEvent))
    setDns((bundle.dns || []) as DnsRecord[])
    setProcesses(
      (bundle.processes || []).map((p: Record<string, unknown>) => ({
        device_id: p.device_id !== undefined ? String(p.device_id) : undefined,
        name: p.name !== undefined ? String(p.name) : undefined,
        path: p.path !== undefined ? String(p.path) : undefined,
        command_line: p.command_line !== undefined ? String(p.command_line) : undefined,
      })),
    )
    setFindings(
      (bundle.findings || []).map((f: Record<string, unknown>) => ({
        device_id: f.device_id !== undefined ? String(f.device_id) : undefined,
        title: f.title !== undefined ? String(f.title) : undefined,
        classification: f.classification !== undefined ? String(f.classification) : undefined,
        detected_patterns: Array.isArray(f.detected_patterns)
          ? (f.detected_patterns as string[])
          : undefined,
      })),
    )
    setLoading(false)
    setRefreshing(false)
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const model = useMemo(
    () => buildInventoryModel(extensionEvents, saseEvents, dns, processes, findings, deviceFilter),
    [extensionEvents, saseEvents, dns, processes, findings, deviceFilter],
  )

  const fleetDevicesWithExtensions = useMemo(() => {
    const ids = new Set(model.extensionAggregates.flatMap((row) => row.device_ids))
    return ids.size
  }, [model.extensionAggregates])

  const fleetDevicesWithSase = useMemo(() => {
    const ids = new Set(model.saseAggregates.flatMap((row) => row.device_ids))
    return ids.size
  }, [model.saseAggregates])

  const clearFilterHref = embedded ? '/?panel=inventory' : '/inventory'
  const queryNeedle = query.trim().toLowerCase()

  const aiToolRows = useMemo(() => {
    const processRows = processes
      .filter((row) => AI_PATTERN.test(`${row.name || ''} ${row.path || ''} ${row.command_line || ''}`))
      .map((row) => ({
        type: 'process',
        name: row.name || 'unknown-process',
        descriptor: row.path || row.command_line || 'n/a',
        device_id: row.device_id || 'unknown',
      }))
    const dnsRows = model.aiDnsAggregates.map((row) => ({
      type: 'dns',
      name: row.query,
      descriptor: `${row.device_ids.length} endpoint(s)`,
      device_id: row.device_ids[0] || 'unknown',
    }))
    return [...processRows, ...dnsRows]
  }, [processes, model.aiDnsAggregates])

  const localRuntimeRows = useMemo(
    () =>
      processes
        .filter((row) => /ollama|vllm|llama|mistral|litellm|localai/i.test(`${row.name || ''} ${row.path || ''} ${row.command_line || ''}`))
        .map((row) => ({
          name: row.name || 'runtime',
          command: row.command_line || row.path || 'n/a',
          device_id: row.device_id || 'unknown',
        })),
    [processes],
  )

  const unknownRows = useMemo(
    () =>
      findings
        .filter((row) => !AI_PATTERN.test(`${row.title || ''} ${row.classification || ''} ${(row.detected_patterns || []).join(' ')}`))
        .map((row) => ({
          title: row.title || 'unknown finding',
          classification: row.classification || 'unclassified',
          device_id: row.device_id || 'unknown',
          patterns: (row.detected_patterns || []).join(', ') || 'n/a',
        })),
    [findings],
  )

  const categoryCounts = {
    extensions: model.extensionAggregates.length,
    ai_tools: aiToolRows.length,
    local_models: localRuntimeRows.length,
    sase: model.saseAggregates.length,
    unknown: unknownRows.length,
  }

  const filteredExtensionRows = useMemo(
    () => model.extensionAggregates.filter((row) => matchesInventoryQuery(row, queryNeedle)),
    [model.extensionAggregates, queryNeedle],
  )
  const filteredAiToolRows = useMemo(
    () => aiToolRows.filter((row) => matchesInventoryQuery(row, queryNeedle)),
    [aiToolRows, queryNeedle],
  )
  const filteredLocalRuntimeRows = useMemo(
    () => localRuntimeRows.filter((row) => matchesInventoryQuery(row, queryNeedle)),
    [localRuntimeRows, queryNeedle],
  )
  const filteredSaseRows = useMemo(
    () => model.saseAggregates.filter((row) => matchesInventoryQuery(row, queryNeedle)),
    [model.saseAggregates, queryNeedle],
  )
  const filteredUnknownRows = useMemo(
    () => unknownRows.filter((row) => matchesInventoryQuery(row, queryNeedle)),
    [unknownRows, queryNeedle],
  )

  return (
    <div className={embedded ? 'bg-gray-50' : 'min-h-screen bg-gray-50'}>
      <header className={embedded ? 'border-b border-gray-200 bg-white' : 'border-b border-gray-200 bg-white'}>
        <div className={embedded ? 'px-0 py-4 sm:px-0' : 'mx-auto max-w-7xl px-4 py-5 sm:px-6 lg:px-8'}>
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex min-w-0 flex-wrap items-center gap-4">
              {!embedded ? (
                <>
                  <a href="/" className="btn btn-secondary h-9 px-3">
                    Dashboard
                  </a>
                  <a href="/?panel=agents" className="btn btn-secondary h-9 px-3">
                    <Server className="mr-2 h-4 w-4" />
                    Agents
                  </a>
                </>
              ) : null}
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <Database className="h-7 w-7 shrink-0 text-primary-600" />
                  <h1 className="text-2xl font-bold text-gray-900">Enterprise AI &amp; control inventory</h1>
                </div>
                <p className="mt-1 text-sm text-gray-500">
                  Observe-only surface from browser, SASE, DNS, process, and finding telemetry. No enforcement.
                </p>
                {deviceFilter ? (
                  <p className="mt-2 text-sm text-primary-800">
                    Filtered to device{' '}
                    <span className="font-mono font-semibold">{deviceFilter}</span>
                    {' · '}
                    <a className="font-medium underline" href={`/agents/${encodeURIComponent(deviceFilter)}`}>
                      Open agent detail
                    </a>
                    {' · '}
                    <a className="font-medium underline" href={clearFilterHref}>
                      Clear filter
                    </a>
                  </p>
                ) : null}
              </div>
            </div>
            <button type="button" onClick={() => void load()} disabled={refreshing} className="btn btn-secondary h-9 px-3">
              <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        </div>
      </header>

      <div className={embedded ? 'mx-auto max-w-[1500px] px-0 py-4' : 'mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8'}>
        <div className="mb-6 rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-900">
          <strong className="font-semibold">Observe-only.</strong> Counts reflect retained visibility samples, not a
          complete enterprise software asset baseline.
        </div>

        {loading ? (
          <EmptyState title="Loading telemetry" message="Collecting extension, DNS, process, and control telemetry." />
        ) : (
          <>
            <WorkbenchHeader
              title="Telemetry"
              subtitle="Raw AI-related observations — browser extensions, processes, DNS, and SASE components across the fleet."
              actions={
                <button type="button" onClick={() => void load()} disabled={refreshing} className="btn btn-secondary h-9 px-3">
                  <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
                  Refresh
                </button>
              }
            />

            <SummaryStrip>
              <KpiTile label="Fleet endpoints" value={devices.length} />
              <KpiTile label="Browser extensions" value={model.extensionAggregates.length} />
              <KpiTile label="AI IDE/CLI tools" value={aiToolRows.length} />
              <KpiTile label="Local runtimes" value={localRuntimeRows.length} />
              <KpiTile label="SASE/SSE controls" value={model.saseAggregates.length} />
              <KpiTile label="Unknown signals" value={unknownRows.length} />
            </SummaryStrip>

            <FilterBar>
              {([
                ['extensions', 'Browser extensions'],
                ['ai_tools', 'AI IDE/CLI tools'],
                ['local_models', 'Local model runtimes'],
                ['sase', 'SASE/SSE controls'],
                ['unknown', 'Unknown signals'],
              ] as const).map(([id, label]) => (
                <button
                  key={id}
                  type="button"
                  onClick={() => setCategory(id)}
                  className={`rounded-full border px-3 py-1 text-xs font-semibold ${
                    category === id ? 'border-primary-600 bg-primary-600 text-white' : 'border-gray-200 bg-gray-50 text-gray-700'
                  }`}
                >
                  {label} ({categoryCounts[id]})
                </button>
              ))}
              <label className="relative min-w-[220px] flex-1">
                <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-gray-400" />
                <input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  className="input h-9 pl-9"
                  placeholder="Search inventory values"
                />
              </label>
            </FilterBar>

            <section className="card p-5">
              {category === 'extensions' && (
                filteredExtensionRows.length === 0 ? (
                  <EmptyState title="No browser extensions" message="Extension telemetry will populate this category when observed." />
                ) : (
                  <BoundedTable
                    headers={['Name', 'Extension id', 'Version', 'Source', 'Evidence', 'Last seen', 'Devices', 'Action']}
                    rows={filteredExtensionRows.map((row) => ([
                        <span key={`${row.extension_id}-name`} className="text-sm font-medium text-gray-900">{formatHostname(row.name)}</span>,
                        <span key={`${row.extension_id}-id`} className="font-mono text-xs" title={row.extension_id}>{formatHash(row.extension_id)}</span>,
                        <span key={`${row.extension_id}-version`} className="text-xs text-gray-600">{row.version || 'n/a'}</span>,
                        <span key={`${row.extension_id}-source`} className="text-xs text-gray-600">{row.source || 'telemetry'}</span>,
                        <span key={`${row.extension_id}-evidence`} className="text-xs text-gray-600" title={row.evidenceSummary}>{formatCommandLine(row.evidenceSummary)}</span>,
                        <span key={`${row.extension_id}-seen`} className="text-xs text-gray-600" title={formatMs(row.last_seen_ms)}>{formatRelativeAge(row.last_seen_ms)}</span>,
                        <DeviceLinks key={`${row.extension_id}-devices`} deviceIds={row.device_ids} />,
                        <button
                          key={`${row.extension_id}-action`}
                          className="text-xs font-semibold text-primary-700 hover:text-primary-900"
                          onClick={() => setDetailModal({ title: `Extension: ${row.name}`, payload: row })}
                        >
                          View detail
                        </button>,
                      ]))}
                  />
                )
              )}

              {category === 'ai_tools' && (
                filteredAiToolRows.length === 0 ? (
                  <EmptyState title="No AI IDE/CLI signals" message="AI tool process or DNS signals are not present in this window." />
                ) : (
                  <BoundedTable
                    headers={['Type', 'Signal', 'Descriptor', 'Device', 'Action']}
                    rows={filteredAiToolRows.map((row, idx) => ([
                        <span key={`type-${idx}`} className="text-xs text-gray-600">{row.type}</span>,
                        <span key={`name-${idx}`} className="text-sm font-medium text-gray-900">{formatHostname(row.name)}</span>,
                        <span key={`desc-${idx}`} className="text-xs text-gray-600" title={row.descriptor}>{formatCommandLine(row.descriptor)}</span>,
                        <span key={`dev-${idx}`} className="font-mono text-xs">{formatHash(row.device_id)}</span>,
                        <button key={`action-${idx}`} className="text-xs font-semibold text-primary-700" onClick={() => setDetailModal({ title: `AI tool signal: ${row.name}`, payload: row })}>
                          View detail
                        </button>,
                      ]))}
                  />
                )
              )}

              {category === 'local_models' && (
                filteredLocalRuntimeRows.length === 0 ? (
                  <EmptyState title="No local model runtimes" message="No Ollama/vLLM/local runtime signals observed in this window." />
                ) : (
                  <BoundedTable
                    headers={['Runtime', 'Command', 'Device', 'Action']}
                    rows={filteredLocalRuntimeRows.map((row, idx) => ([
                        <span key={`name-${idx}`} className="text-sm font-medium text-gray-900">{row.name}</span>,
                        <span key={`cmd-${idx}`} className="text-xs text-gray-600" title={row.command}>{formatPath(row.command)}</span>,
                        <span key={`dev-${idx}`} className="font-mono text-xs">{formatHash(row.device_id)}</span>,
                        <button key={`action-${idx}`} className="text-xs font-semibold text-primary-700" onClick={() => setDetailModal({ title: `Local runtime: ${row.name}`, payload: row })}>
                          View detail
                        </button>,
                      ]))}
                  />
                )
              )}

              {category === 'sase' && (
                filteredSaseRows.length === 0 ? (
                  <EmptyState title="No SASE/SSE controls" message="Control component telemetry appears here when observed." />
                ) : (
                  <BoundedTable
                    headers={['Vendor', 'Product', 'Name', 'Type', 'Status', 'Last seen', 'Devices', 'Action']}
                    rows={filteredSaseRows.map((row) => ([
                        <span key={`${row.key}-vendor`} className="text-sm font-medium text-gray-900">{formatHostname(row.vendor)}</span>,
                        <span key={`${row.key}-product`} className="text-xs text-gray-600">{formatHostname(row.product)}</span>,
                        <span key={`${row.key}-name`} className="text-xs text-gray-600">{formatHostname(row.name)}</span>,
                        <span key={`${row.key}-type`} className="text-xs text-gray-600">{row.component_type || 'n/a'}</span>,
                        <span key={`${row.key}-status`} className="text-xs text-gray-600">{row.status || 'n/a'}</span>,
                        <span key={`${row.key}-seen`} className="text-xs text-gray-600" title={formatMs(row.last_seen_ms)}>{formatRelativeAge(row.last_seen_ms)}</span>,
                        <DeviceLinks key={`${row.key}-devices`} deviceIds={row.device_ids} />,
                        <button key={`${row.key}-action`} className="text-xs font-semibold text-primary-700" onClick={() => setDetailModal({ title: `SASE/SSE: ${row.vendor} ${row.product}`, payload: row })}>
                          View detail
                        </button>,
                      ]))}
                  />
                )
              )}

              {category === 'unknown' && (
                filteredUnknownRows.length === 0 ? (
                  <EmptyState title="No unknown signals" message="Unclassified inventory-like findings will appear here." />
                ) : (
                  <BoundedTable
                    headers={['Title', 'Classification', 'Patterns', 'Device', 'Action']}
                    rows={filteredUnknownRows.map((row, idx) => ([
                        <span key={`title-${idx}`} className="text-sm font-medium text-gray-900">{formatHostname(row.title)}</span>,
                        <span key={`class-${idx}`} className="text-xs text-gray-600">{row.classification}</span>,
                        <span key={`patterns-${idx}`} className="text-xs text-gray-600" title={row.patterns}>{formatCommandLine(row.patterns)}</span>,
                        <span key={`dev-${idx}`} className="font-mono text-xs">{formatHash(row.device_id)}</span>,
                        <button key={`action-${idx}`} className="text-xs font-semibold text-primary-700" onClick={() => setDetailModal({ title: `Unknown signal: ${row.title}`, payload: row })}>
                          View detail
                        </button>,
                      ]))}
                  />
                )
              )}
            </section>
          </>
        )}
      </div>
      <DetailModal
        open={Boolean(detailModal)}
        title={detailModal?.title || 'Inventory detail'}
        detail={detailModal?.payload || {}}
        onClose={() => setDetailModal(null)}
      />
    </div>
  )
}

function FleetCard({
  icon: Icon,
  label,
  value,
  detail,
}: {
  icon: typeof Server
  label: string
  value: number
  detail: string
}) {
  return (
    <div className="card p-4">
      <div className="flex items-start justify-between gap-2">
        <Icon className="h-5 w-5 text-primary-600" />
        <span className="badge badge-info text-[10px]">Fleet</span>
      </div>
      <p className="mt-3 text-xs font-semibold uppercase tracking-wide text-gray-500">{label}</p>
      <p className="mt-1 text-2xl font-bold text-gray-900">{value}</p>
      <p className="mt-2 text-xs leading-relaxed text-gray-500">{detail}</p>
    </div>
  )
}

function InventorySection({
  id,
  title,
  subtitle,
  icon: Icon,
  rows,
  children,
}: {
  id: string
  title: string
  subtitle: string
  icon: typeof Chrome
  rows: number
  children: React.ReactNode
}) {
  return (
    <section id={id} className="card mb-8">
      <div className="border-b border-gray-200 px-6 py-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            <Icon className="mt-0.5 h-6 w-6 text-primary-600" />
            <div>
              <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
              <p className="text-sm text-gray-500">{subtitle}</p>
            </div>
          </div>
          <span className="badge badge-info">{rows} rows</span>
        </div>
      </div>
      <div className="p-6">{children}</div>
    </section>
  )
}

function EmptyCategory({ message }: { message: string }) {
  return (
    <div className="rounded-lg border border-dashed border-gray-300 bg-gray-50 p-6 text-sm text-gray-600">
      {message}
    </div>
  )
}

function PlaceholderCategory({ icon: Icon, title, body }: { icon: typeof Wrench; title: string; body: string }) {
  return (
    <div className="card p-5">
      <div className="flex items-center gap-2 text-gray-800">
        <Icon className="h-5 w-5 text-gray-500" />
        <h3 className="text-base font-semibold">{title}</h3>
      </div>
      <p className="mt-2 text-sm text-gray-500">{body}</p>
      <p className="mt-3 text-xs font-medium uppercase tracking-wide text-gray-400">No telemetry · placeholder</p>
    </div>
  )
}

function DeviceLinks({ deviceIds }: { deviceIds: string[] }) {
  if (deviceIds.length === 0) return <span className="text-xs text-gray-400">n/a</span>
  const shown = deviceIds.slice(0, 4)
  const rest = deviceIds.length - shown.length
  return (
    <div className="flex flex-wrap gap-1.5">
      {shown.map((id) => (
        <a
          key={id}
          href={`/agents/${encodeURIComponent(id)}`}
          className="inline-flex max-w-[160px] truncate rounded border border-primary-200 bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-800 hover:bg-primary-100"
          title={id}
        >
          {id}
        </a>
      ))}
      {rest > 0 ? <span className="self-center text-xs text-gray-500">+{rest} more</span> : null}
    </div>
  )
}

type ExtensionAgg = {
  extension_id: string
  name: string
  version: string
  source: string
  evidenceSummary: string
  first_seen_ms: number
  last_seen_ms: number
  device_ids: string[]
  confidence: string
}

type BrowserAgg = {
  browser: string
  device_ids: string[]
}

type SaseAgg = {
  key: string
  vendor: string
  product: string
  name: string
  component_type: string
  version: string
  status: string
  evidence: string[]
  first_seen_ms: number
  last_seen_ms: number
  device_ids: string[]
}

type AiDnsAgg = {
  query: string
  device_ids: string[]
}

function buildInventoryModel(
  extensionEvents: EventRecord[],
  saseEvents: EventRecord[],
  dns: DnsRecord[],
  processes: ProcessRecord[],
  findings: FindingRecord[],
  deviceFilter: string,
) {
  const extFiltered = deviceFilter
    ? extensionEvents.filter((e) => e.device_id === deviceFilter)
    : extensionEvents
  const saseFiltered = deviceFilter ? saseEvents.filter((e) => e.device_id === deviceFilter) : saseEvents
  const dnsFiltered = deviceFilter ? dns.filter((d) => d.device_id === deviceFilter) : dns
  const procFiltered = deviceFilter ? processes.filter((p) => p.device_id === deviceFilter) : processes
  const findFiltered = deviceFilter ? findings.filter((f) => f.device_id === deviceFilter) : findings

  const extMap = new Map<string, ExtensionAgg & { _perm: string[]; _host: string[] }>()
  for (const event of extFiltered) {
    const p = event.payload || {}
    const extension_id = String(p.extension_id || '')
    if (!extension_id) continue
    const ts = event.received_at_ms || event.timestamp_ms || 0
    const name = String(p.name || extension_id)
    const version = String(p.version || '')
    const collection = String(p.collection_method || '')
    const source = [event.source, collection].filter(Boolean).join(' · ') || 'telemetry'
    const perms = Array.isArray(p.permissions) ? (p.permissions as string[]) : []
    const hostPerms = Array.isArray(p.host_permissions) ? (p.host_permissions as string[]) : []
    const browser = String(p.browser || '')
    const profile = String(p.profile || '')
    const evidenceSummary = [browser && `browser:${browser}`, profile && `profile:${profile}`, `perms:${perms.length}`, `host:${hostPerms.length}`]
      .filter(Boolean)
      .join(' · ')

    const existing = extMap.get(extension_id)
    if (!existing) {
      extMap.set(extension_id, {
        extension_id,
        name,
        version,
        source,
        evidenceSummary,
        first_seen_ms: ts,
        last_seen_ms: ts,
        device_ids: event.device_id ? [event.device_id] : [],
        confidence: 'observed',
        _perm: perms,
        _host: hostPerms,
      })
      continue
    }
    existing.first_seen_ms = Math.min(existing.first_seen_ms, ts)
    existing.last_seen_ms = Math.max(existing.last_seen_ms, ts)
    if (event.device_id && !existing.device_ids.includes(event.device_id)) existing.device_ids.push(event.device_id)
    if (ts >= existing.last_seen_ms) {
      existing.version = version || existing.version
      existing.name = name || existing.name
      existing.source = source
      existing.evidenceSummary = evidenceSummary || existing.evidenceSummary
      existing._perm = perms.length ? perms : existing._perm
      existing._host = hostPerms.length ? hostPerms : existing._host
    }
  }

  const extensionAggregates: ExtensionAgg[] = Array.from(extMap.values())
    .map(({ _perm, _host, ...rest }) => ({
      ...rest,
      device_ids: Array.from(new Set(rest.device_ids)),
    }))
    .sort((a, b) => b.last_seen_ms - a.last_seen_ms)

  const browserMap = new Map<string, Set<string>>()
  for (const event of extFiltered) {
    const browser = String((event.payload as { browser?: string })?.browser || '')
    if (!browser) continue
    if (!browserMap.has(browser)) browserMap.set(browser, new Set())
    if (event.device_id) browserMap.get(browser)!.add(event.device_id)
  }
  const browserAggregates: BrowserAgg[] = Array.from(browserMap.entries())
    .map(([browser, set]) => ({ browser, device_ids: Array.from(set) }))
    .sort((a, b) => b.device_ids.length - a.device_ids.length)

  const uniqueBrowsers = browserAggregates.length

  const saseMap = new Map<string, SaseAgg & { deviceSet: Set<string> }>()
  for (const event of saseFiltered) {
    const p = event.payload || {}
    const vendor = String(p.vendor || '')
    const product = String(p.product || '')
    const name = String(p.name || '')
    const component_type = String(p.component_type || '')
    const key = [vendor, product, name, component_type].join('\t')
    if (!key.trim()) continue
    const ts = event.received_at_ms || event.timestamp_ms || 0
    const evidence = Array.isArray(p.evidence) ? (p.evidence as string[]) : []
    const version = p.version != null ? String(p.version) : ''
    const status = p.status != null ? String(p.status) : ''

    const row = saseMap.get(key)
    if (!row) {
      saseMap.set(key, {
        key,
        vendor,
        product,
        name,
        component_type,
        version,
        status,
        evidence,
        first_seen_ms: ts,
        last_seen_ms: ts,
        device_ids: [],
        deviceSet: new Set(event.device_id ? [event.device_id] : []),
      })
      continue
    }
    row.first_seen_ms = Math.min(row.first_seen_ms, ts)
    row.last_seen_ms = Math.max(row.last_seen_ms, ts)
    if (event.device_id) row.deviceSet.add(event.device_id)
    if (evidence.length) row.evidence = Array.from(new Set([...row.evidence, ...evidence])).slice(0, 12)
  }

  const saseAggregates: SaseAgg[] = Array.from(saseMap.values())
    .map((row) => ({
      ...row,
      device_ids: Array.from(row.deviceSet),
    }))
    .map(({ deviceSet: _, ...rest }) => rest)
    .sort((a, b) => b.last_seen_ms - a.last_seen_ms)

  const aiDnsMap = new Map<string, Set<string>>()
  for (const row of dnsFiltered) {
    const q = row.query || ''
    if (!AI_PATTERN.test(q)) continue
    const id = row.device_id
    const key = q.toLowerCase()
    if (!aiDnsMap.has(key)) aiDnsMap.set(key, new Set())
    if (id) aiDnsMap.get(key)!.add(id)
  }
  const aiDnsAggregates: AiDnsAgg[] = Array.from(aiDnsMap.entries())
    .map(([query, set]) => ({ query, device_ids: Array.from(set) }))
    .sort((a, b) => b.device_ids.length - a.device_ids.length)

  const aiProcessCount = procFiltered.filter((p) => AI_PATTERN.test(`${p.name || ''} ${p.path || ''} ${p.command_line || ''}`)).length

  const aiFindingCount = findFiltered.filter((f) =>
    AI_PATTERN.test(`${f.title || ''} ${f.classification || ''} ${(f.detected_patterns || []).join(' ')}`),
  ).length

  return {
    extensionAggregates,
    browserAggregates,
    uniqueBrowsers,
    saseAggregates,
    aiDnsAggregates,
    aiProcessCount,
    aiFindingCount,
  }
}

function formatMs(ms: number) {
  if (!ms) return 'n/a'
  return new Date(ms).toLocaleString()
}

function truncate(s: string, max: number) {
  if (s.length <= max) return s
  return `${s.slice(0, max - 1)}…`
}
