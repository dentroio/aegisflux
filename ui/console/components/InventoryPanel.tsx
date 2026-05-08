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
} from 'lucide-react'

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

  const load = useCallback(async () => {
    setRefreshing(true)
    const [deviceRes, extRes, saseRes, dnsRes, procRes, findRes] = await Promise.all([
      fetchJson<{ devices?: DeviceRecord[] }>('/api/visibility/devices?limit=300', {}),
      fetchJson<{ events?: EventRecord[] }>(
        '/api/visibility/events?event_type=aegis.browser_extension.observed&limit=400',
        {},
      ),
      fetchJson<{ events?: EventRecord[] }>(
        '/api/visibility/events?event_type=aegis.sase_component.observed&limit=400',
        {},
      ),
      fetchJson<{ dns?: DnsRecord[]; observations?: DnsRecord[] }>('/api/visibility/dns?limit=400', {}),
      fetchJson<{ processes?: ProcessRecord[] }>('/api/visibility/processes?limit=400', {}),
      fetchJson<{ findings?: FindingRecord[] }>('/api/visibility/findings?limit=200', {}),
    ])
    setDevices(deviceRes.devices || [])
    setExtensionEvents(extRes.events || [])
    setSaseEvents(saseRes.events || [])
    setDns(dnsRes.dns || dnsRes.observations || [])
    setProcesses(procRes.processes || [])
    setFindings(findRes.findings || [])
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
          <p className="text-center text-gray-500">Loading visibility inventory…</p>
        ) : (
          <>
            <section className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <FleetCard icon={Server} label="Fleet endpoints" value={devices.length} detail="From visibility devices API" />
              <FleetCard
                icon={Chrome}
                label="Browsers observed"
                value={model.uniqueBrowsers}
                detail="Distinct browser products in extension telemetry"
              />
              <FleetCard
                icon={Database}
                label="Extension SKUs"
                value={model.extensionAggregates.length}
                detail={`Across ${fleetDevicesWithExtensions} endpoint${fleetDevicesWithExtensions === 1 ? '' : 's'}`}
              />
              <FleetCard
                icon={LockKeyhole}
                label="SASE / SSE components"
                value={model.saseAggregates.length}
                detail={`Across ${fleetDevicesWithSase} endpoint${fleetDevicesWithSase === 1 ? '' : 's'}`}
              />
              <FleetCard
                icon={Bot}
                label="AI DNS destinations"
                value={model.aiDnsAggregates.length}
                detail="Distinct AI-related queries in DNS window"
              />
              <FleetCard
                icon={Terminal}
                label="AI process signals"
                value={model.aiProcessCount}
                detail="Processes matching AI/tooling heuristics"
              />
              <FleetCard
                icon={Shield}
                label="AI-related findings"
                value={model.aiFindingCount}
                detail="Findings matching AI heuristics"
              />
              <FleetCard
                icon={Monitor}
                label="Raw extension events"
                value={extensionEvents.length}
                detail="Ingest window for this view"
              />
            </section>

            <InventorySection
              id="browser-extensions"
              title="Browser extensions"
              subtitle="From aegis.browser_extension.observed"
              icon={Chrome}
              rows={model.extensionAggregates.length}
            >
              {model.extensionAggregates.length === 0 ? (
                <EmptyCategory message="Deploy the Windows agent lab profile and browse with extensions enabled to populate this category." />
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 text-sm">
                    <thead>
                      <tr className="text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                        <th className="py-2 pr-4">Name</th>
                        <th className="py-2 pr-4">Extension ID</th>
                        <th className="py-2 pr-4">Version</th>
                        <th className="py-2 pr-4">Source</th>
                        <th className="py-2 pr-4">Evidence</th>
                        <th className="py-2 pr-4">First / last seen</th>
                        <th className="py-2 pr-4">Devices</th>
                        <th className="py-2 pr-4">Confidence</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {model.extensionAggregates.map((row) => (
                        <tr key={row.extension_id}>
                          <td className="py-3 pr-4 font-medium text-gray-900">{row.name}</td>
                          <td className="max-w-[140px] truncate py-3 pr-4 font-mono text-xs text-gray-700">{row.extension_id}</td>
                          <td className="py-3 pr-4 text-gray-700">{row.version}</td>
                          <td className="py-3 pr-4 text-gray-600">{row.source}</td>
                          <td className="max-w-xs py-3 pr-4 text-xs text-gray-600" title={row.evidenceSummary}>
                            {truncate(row.evidenceSummary, 80)}
                          </td>
                          <td className="whitespace-nowrap py-3 pr-4 text-xs text-gray-600">
                            {formatMs(row.first_seen_ms)} → {formatMs(row.last_seen_ms)}
                          </td>
                          <td className="py-3 pr-4">
                            <DeviceLinks deviceIds={row.device_ids} />
                          </td>
                          <td className="py-3 pr-4 text-xs text-gray-600">{row.confidence}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </InventorySection>

            <InventorySection
              id="enterprise-browsers"
              title="Enterprise browsers"
              subtitle="Distinct browser hosts from extension telemetry"
              icon={Monitor}
              rows={model.browserAggregates.length}
            >
              {model.browserAggregates.length === 0 ? (
                <EmptyCategory message="Browser family is inferred from extension observations only." />
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 text-sm">
                    <thead>
                      <tr className="text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                        <th className="py-2 pr-4">Browser</th>
                        <th className="py-2 pr-4">Endpoints</th>
                        <th className="py-2 pr-4">Devices</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {model.browserAggregates.map((row) => (
                        <tr key={row.browser}>
                          <td className="py-3 pr-4 font-medium text-gray-900">{row.browser}</td>
                          <td className="py-3 pr-4 text-gray-700">{row.device_ids.length}</td>
                          <td className="py-3 pr-4">
                            <DeviceLinks deviceIds={row.device_ids} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </InventorySection>

            <InventorySection
              id="sase-sse"
              title="SASE / SSE components"
              subtitle="From aegis.sase_component.observed (Palo Alto, Zscaler, Cisco, …)"
              icon={LockKeyhole}
              rows={model.saseAggregates.length}
            >
              {model.saseAggregates.length === 0 ? (
                <EmptyCategory message="When endpoint agents observe Zscaler, GlobalProtect, or similar components, they appear here." />
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 text-sm">
                    <thead>
                      <tr className="text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                        <th className="py-2 pr-4">Vendor</th>
                        <th className="py-2 pr-4">Product</th>
                        <th className="py-2 pr-4">Name</th>
                        <th className="py-2 pr-4">Type</th>
                        <th className="py-2 pr-4">Version</th>
                        <th className="py-2 pr-4">Status</th>
                        <th className="py-2 pr-4">Evidence</th>
                        <th className="py-2 pr-4">First / last seen</th>
                        <th className="py-2 pr-4">Devices</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {model.saseAggregates.map((row) => (
                        <tr key={row.key}>
                          <td className="py-3 pr-4 font-medium text-gray-900">{row.vendor}</td>
                          <td className="py-3 pr-4 text-gray-700">{row.product}</td>
                          <td className="py-3 pr-4 text-gray-800">{row.name}</td>
                          <td className="py-3 pr-4 text-gray-600">{row.component_type}</td>
                          <td className="py-3 pr-4 text-gray-600">{row.version || 'n/a'}</td>
                          <td className="py-3 pr-4 text-gray-600">{row.status || 'n/a'}</td>
                          <td className="max-w-xs py-3 pr-4 text-xs text-gray-600">{truncate(row.evidence.join('; '), 96)}</td>
                          <td className="whitespace-nowrap py-3 pr-4 text-xs text-gray-600">
                            {formatMs(row.first_seen_ms)} → {formatMs(row.last_seen_ms)}
                          </td>
                          <td className="py-3 pr-4">
                            <DeviceLinks deviceIds={row.device_ids} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </InventorySection>

            <InventorySection
              id="ai-destinations"
              title="AI destinations (DNS)"
              subtitle="Heuristic match on DNS queries in the visibility window"
              icon={Bot}
              rows={model.aiDnsAggregates.length}
            >
              {model.aiDnsAggregates.length === 0 ? (
                <EmptyCategory message="DNS observations with vendor AI hostnames will roll up here for fleet context." />
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 text-sm">
                    <thead>
                      <tr className="text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                        <th className="py-2 pr-4">Query</th>
                        <th className="py-2 pr-4">Endpoints</th>
                        <th className="py-2 pr-4">Devices</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {model.aiDnsAggregates.map((row) => (
                        <tr key={row.query}>
                          <td className="py-3 pr-4 font-mono text-xs font-medium text-gray-900">{row.query}</td>
                          <td className="py-3 pr-4 text-gray-700">{row.device_ids.length}</td>
                          <td className="py-3 pr-4">
                            <DeviceLinks deviceIds={row.device_ids} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </InventorySection>

            <div className="grid gap-6 lg:grid-cols-2">
              <PlaceholderCategory
                icon={Wrench}
                title="IDE extensions"
                body="No IDE extension telemetry is wired yet. This category will list VS Code / JetBrains style signals when collectors emit them."
              />
              <PlaceholderCategory
                icon={Terminal}
                title="CLI AI tools"
                body="CLI inventory will roll up from process and shell evidence when dedicated events land."
              />
              <PlaceholderCategory
                icon={Server}
                title="Local model runtimes"
                body="Ollama, vLLM, and similar local runtimes will appear here from process and network heuristics."
              />
              <PlaceholderCategory
                icon={Bot}
                title="MCP clients / servers"
                body="Model Context Protocol endpoints will be classified when MCP-specific telemetry is available."
              />
              <PlaceholderCategory
                icon={Shield}
                title="EDR / MDM / security agents"
                body="Third-party security agent inventory is out of scope for this slice; use agent and process views for raw evidence."
              />
            </div>
          </>
        )}
      </div>
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
