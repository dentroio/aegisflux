'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  AlertTriangle,
  ArrowRight,
  ChevronDown,
  ChevronRight,
  Globe2,
  Network,
  RefreshCw,
  Search,
  ShieldCheck,
  Server,
  Sparkles,
  Terminal,
  TerminalSquare,
  Wrench,
} from 'lucide-react'
import {
  EmptyState,
  KpiTile,
  SummaryStrip,
  WorkbenchHeader,
} from '@/components/workbench/primitives'

type Node = {
  id: string
  type: string
  label: string
  operator_label?: string
  detail?: string
  evidence_id?: string
  confidence: 'high' | 'medium' | 'low' | string
  confidence_reason?: string
  attributes?: Record<string, string>
  missing?: boolean
  related_abom_id?: string
  related_abom_label?: string
}

type Edge = {
  from: string
  to: string
  label: string
  confidence: string
}

type Narrative = {
  what_happened?: string
  why_it_matters?: string
  what_we_know?: string[]
  what_is_missing?: string[]
  recommended_next_step?: string
}

type EvidencePathResponse = {
  ok?: boolean
  generated_at_ms?: number
  subject?: { type?: string; id?: string; device_id?: string; agent_id?: string }
  summary?: string
  narrative?: Narrative
  nodes?: Node[]
  edges?: Edge[]
  missing_evidence?: string[]
  confidence_overall?: string
  confidence_reason?: string
  draft_controls?: Array<{
    control_id?: string
    title?: string
    action?: string
    target?: string
    scope?: string
    reason?: string
    blast_radius?: string[]
    rollback?: string[]
    evidence?: string[]
  }>
  raw_processes?: any[]
  raw_flows?: any[]
  raw_dns?: any[]
  raw_findings?: any[]
  empty_help?: string
}

type EvidenceGraphPanelProps = {
  initialDeviceId?: string
  initialFindingId?: string
  initialProcessGUID?: string
  embedded?: boolean
  autoLoad?: boolean
}

const NODE_TYPE_ORDER = [
  'finding',
  'detection_pack',
  'parent_process',
  'process',
  'flow',
  'dns',
  'endpoint',
  'draft_control',
] as const

const NODE_TYPE_LABEL: Record<string, string> = {
  endpoint: 'Endpoint',
  process: 'Process',
  parent_process: 'Parent process',
  flow: 'Network flow',
  dns: 'DNS lookup',
  finding: 'Finding',
  detection_pack: 'Detection pack',
  draft_control: 'Draft control',
  user_session: 'User session',
}

const NODE_TYPE_ICON: Record<string, typeof Server> = {
  endpoint: Server,
  process: TerminalSquare,
  parent_process: Terminal,
  flow: Network,
  dns: Globe2,
  finding: AlertTriangle,
  detection_pack: Sparkles,
  draft_control: ShieldCheck,
  user_session: Wrench,
}

function confidenceTone(confidence: string): { dot: string; label: string } {
  if (confidence === 'high') return { dot: 'bg-emerald-500', label: 'High' }
  if (confidence === 'medium') return { dot: 'bg-amber-500', label: 'Medium' }
  return { dot: 'bg-slate-400', label: 'Low' }
}

function formatRelative(ts?: number) {
  if (!ts) return 'n/a'
  const seconds = Math.max(0, Math.round((Date.now() - ts) / 1000))
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  return `${Math.round(minutes / 60)}h ago`
}

export function EvidenceGraphPanel({
  initialDeviceId = '',
  initialFindingId = '',
  initialProcessGUID = '',
  embedded = false,
  autoLoad = true,
}: EvidenceGraphPanelProps) {
  const [findingId, setFindingId] = useState(initialFindingId)
  const [deviceId, setDeviceId] = useState(initialDeviceId)
  const [processGUID, setProcessGUID] = useState(initialProcessGUID)
  const [data, setData] = useState<EvidencePathResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showRaw, setShowRaw] = useState(false)

  const requestPath = useMemo(() => {
    const params = new URLSearchParams()
    if (findingId.trim()) params.set('finding_id', findingId.trim())
    if (deviceId.trim()) params.set('device_id', deviceId.trim())
    if (processGUID.trim()) params.set('process_guid', processGUID.trim())
    return `/api/visibility/evidence-path?${params.toString()}`
  }, [findingId, deviceId, processGUID])

  const load = useCallback(async () => {
    if (!findingId.trim() && !deviceId.trim()) {
      setError('Provide a finding id or a device id to build an evidence path.')
      return
    }
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(requestPath, { cache: 'no-store' })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      const body = (await res.json()) as EvidencePathResponse
      setData(body)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load evidence path')
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [findingId, deviceId, requestPath])

  useEffect(() => {
    if (autoLoad && (initialFindingId || initialDeviceId)) {
      void load()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const nodesByType = useMemo(() => {
    const map = new Map<string, Node>()
    for (const node of data?.nodes || []) {
      if (!map.has(node.type)) map.set(node.type, node)
    }
    return map
  }, [data])

  const orderedNodes = useMemo(() => {
    const seen = new Set<string>()
    const ordered: Node[] = []
    for (const type of NODE_TYPE_ORDER) {
      const node = nodesByType.get(type)
      if (node) {
        ordered.push(node)
        seen.add(node.id)
      }
    }
    for (const node of data?.nodes || []) {
      if (!seen.has(node.id)) ordered.push(node)
    }
    return ordered
  }, [data, nodesByType])

  const overallTone = confidenceTone(data?.confidence_overall || 'low')

  const subjectLabel = (() => {
    const subj = data?.subject
    if (!subj) return ''
    if (subj.type === 'finding') return `Finding ${subj.id || 'unknown'}`
    if (subj.type === 'process') return `Process ${subj.id || 'unknown'}`
    return `Endpoint ${subj.device_id || subj.id || 'unknown'}`
  })()

  return (
    <div className={embedded ? '' : 'min-h-screen bg-gray-50'}>
      <WorkbenchHeader
        title="Evidence path"
        subtitle="A trusted path from finding through process, flow, DNS, and endpoint to draft control."
        actions={
          <div className="flex items-center gap-2">
            <span className="text-xs text-slate-500">
              {data?.generated_at_ms ? `Updated ${formatRelative(data.generated_at_ms)}` : 'Idle'}
            </span>
            <button
              type="button"
              onClick={() => void load()}
              disabled={loading}
              className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        }
      />

      <section className="mb-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="grid gap-3 md:grid-cols-3">
          <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Finding id
            <input
              value={findingId}
              onChange={(event) => setFindingId(event.target.value)}
              className="mt-1 h-9 w-full rounded-md border border-slate-200 px-3 text-sm normal-case tracking-normal text-slate-900 outline-none focus:border-blue-500"
              placeholder="optional"
            />
          </label>
          <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Device id
            <input
              value={deviceId}
              onChange={(event) => setDeviceId(event.target.value)}
              className="mt-1 h-9 w-full rounded-md border border-slate-200 px-3 text-sm normal-case tracking-normal text-slate-900 outline-none focus:border-blue-500"
              placeholder="optional"
            />
          </label>
          <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Process GUID
            <input
              value={processGUID}
              onChange={(event) => setProcessGUID(event.target.value)}
              className="mt-1 h-9 w-full rounded-md border border-slate-200 px-3 text-sm normal-case tracking-normal text-slate-900 outline-none focus:border-blue-500"
              placeholder="optional"
            />
          </label>
        </div>
        <div className="mt-3 flex items-center justify-between gap-3">
          <button
            type="button"
            onClick={() => void load()}
            disabled={loading}
            className="inline-flex h-9 items-center gap-2 rounded-md border border-blue-600 bg-blue-600 px-3 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-60"
          >
            <Search className="h-4 w-4" />
            Build path
          </button>
          <span className="text-xs text-slate-500">{subjectLabel}</span>
        </div>
      </section>

      {error ? (
        <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      ) : null}

      {!data && !loading && !error ? (
        <EmptyState
          title="No path requested yet"
          message="Enter a finding id, device id, or process GUID and click Build path to assemble an evidence trail."
          hint="Not sure where to start? The sample scenarios include subject ids that exercise the evidence path end to end."
          actions={
            <a
              href="/demo/scenarios"
              className="inline-flex h-8 items-center gap-1 rounded-md border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-700 hover:bg-slate-50"
            >
              View sample scenarios
            </a>
          }
        />
      ) : null}

      {data && (data.nodes || []).length === 0 ? (
        <EmptyState
          title="No evidence yet"
          message={data.empty_help || 'No process, flow, DNS, or finding evidence yet for this subject.'}
          hint="In the lab, run the matching sample scenario so the agent emits the expected process, DNS, and finding evidence."
          actions={
            <a
              href="/demo/scenarios"
              className="inline-flex h-8 items-center gap-1 rounded-md border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-700 hover:bg-slate-50"
            >
              View sample scenarios
            </a>
          }
        />
      ) : null}

      {data && (data.nodes || []).length > 0 ? (
        <>
          <SummaryStrip>
            <KpiTile label="Nodes" value={(data.nodes || []).length} />
            <KpiTile label="Edges" value={(data.edges || []).length} />
            <KpiTile label="Missing" value={(data.missing_evidence || []).length} />
            <KpiTile label="Confidence" value={overallTone.label} />
          </SummaryStrip>

          {data.narrative ? (
            <NarrativeBlock
              narrative={data.narrative}
              overallTone={overallTone}
              confidenceReason={data.confidence_reason}
              draftControls={data.draft_controls || []}
              subject={data.subject}
            />
          ) : data.summary ? (
            <section className="mb-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Path summary</h3>
              <p className="mt-2 text-sm leading-6 text-slate-800">{data.summary}</p>
            </section>
          ) : null}

          <section className="mb-4 grid gap-3">
            {orderedNodes.map((node, idx) => {
              const Icon = NODE_TYPE_ICON[node.type] || Wrench
              const tone = confidenceTone(node.confidence)
              const next = orderedNodes[idx + 1]
              return (
                <div key={node.id}>
                  <article
                    className={`flex items-start gap-3 rounded-xl border p-4 shadow-sm ${
                      node.missing ? 'border-amber-200 bg-amber-50' : 'border-slate-200 bg-white'
                    }`}
                  >
                    <div className={`mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-md ${node.missing ? 'bg-amber-100 text-amber-800' : 'bg-blue-50 text-blue-700'}`}>
                      <Icon className="h-4 w-4" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                          {node.operator_label || NODE_TYPE_LABEL[node.type] || node.type}
                        </span>
                        <span className="text-[10px] uppercase tracking-wide text-slate-400">
                          {NODE_TYPE_LABEL[node.type] || node.type}
                        </span>
                        <span className={`inline-flex items-center gap-1 text-xs ${node.missing ? 'text-amber-800' : 'text-slate-600'}`}>
                          <span className={`h-2 w-2 rounded-full ${tone.dot}`} />
                          {tone.label}
                        </span>
                      </div>
                      <p className="mt-1 truncate text-sm font-semibold text-slate-900" title={node.label}>
                        {node.label}
                      </p>
                      {node.detail ? (
                        <p className="mt-1 truncate text-xs text-slate-600" title={node.detail}>
                          {node.detail}
                        </p>
                      ) : null}
                      {node.confidence_reason ? (
                        <p className={`mt-2 text-xs leading-relaxed ${node.missing ? 'text-amber-900' : 'text-slate-600'}`}>
                          {node.confidence_reason}
                        </p>
                      ) : null}
                      {node.attributes && Object.keys(node.attributes).length > 0 ? (
                        <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-600">
                          {Object.entries(node.attributes).map(([key, value]) => (
                            <div key={key} className="min-w-0">
                              <dt className="truncate font-semibold text-slate-500">{key.replace(/_/g, ' ')}</dt>
                              <dd className="truncate" title={value}>{value}</dd>
                            </div>
                          ))}
                        </dl>
                      ) : null}
                      {node.related_abom_id ? (
                        <a
                          href={`/discover/abom`}
                          className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-blue-700 hover:text-blue-900"
                        >
                          Open related ABOM item ({node.related_abom_label || node.related_abom_id})
                          <ArrowRight className="h-3 w-3" />
                        </a>
                      ) : null}
                      {node.evidence_id ? (
                        <p className="mt-2 truncate font-mono text-[11px] text-slate-500" title={node.evidence_id}>
                          {node.evidence_id}
                        </p>
                      ) : null}
                    </div>
                  </article>
                  {next ? (
                    <div className="my-1 flex items-center gap-2 pl-12 text-xs font-semibold uppercase tracking-wide text-slate-400">
                      <ArrowRight className="h-3 w-3" />
                      {edgeLabelBetween(data.edges || [], node.id, next.id)}
                    </div>
                  ) : null}
                </div>
              )
            })}
          </section>

          <RawEvidenceBlock data={data} open={showRaw} onToggle={() => setShowRaw((value) => !value)} />
        </>
      ) : null}

    </div>
  )
}

function NarrativeBlock({
  narrative,
  overallTone,
  confidenceReason,
  draftControls,
  subject,
}: {
  narrative: Narrative
  overallTone: { dot: string; label: string }
  confidenceReason?: string
  draftControls: NonNullable<EvidencePathResponse['draft_controls']>
  subject?: EvidencePathResponse['subject']
}) {
  const designerHref = useMemo(() => {
    const params = new URLSearchParams()
    const findingId = subject?.type === 'finding' ? subject.id : undefined
    if (findingId) params.set('finding_id', findingId)
    if (subject?.device_id) params.set('device_id', subject.device_id)
    return `/control/controls?${params.toString()}`
  }, [subject])

  return (
    <section className="mb-4 grid gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex flex-wrap items-center gap-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Investigation explanation</h3>
        <span className="inline-flex items-center gap-1 text-xs text-slate-600">
          <span className={`h-2 w-2 rounded-full ${overallTone.dot}`} />
          Overall confidence {overallTone.label}
        </span>
      </div>
      {confidenceReason ? (
        <p className="text-xs leading-relaxed text-slate-600">{confidenceReason}</p>
      ) : null}

      <div className="grid gap-3 md:grid-cols-2">
        <NarrativeCard title="What happened" emphasis>
          <p className="text-sm leading-6 text-slate-800">{narrative.what_happened || 'No description yet.'}</p>
        </NarrativeCard>
        <NarrativeCard title="Why it matters" emphasis>
          <p className="text-sm leading-6 text-slate-800">{narrative.why_it_matters || 'No commentary yet.'}</p>
        </NarrativeCard>
        <NarrativeCard title="What we know">
          <BulletList items={narrative.what_we_know} fallback="No bulletable evidence yet." />
        </NarrativeCard>
        <NarrativeCard title="What is missing" tone="amber">
          <BulletList items={narrative.what_is_missing} fallback="No critical gaps." />
        </NarrativeCard>
      </div>

      <div className="rounded-md border border-blue-200 bg-blue-50 p-3 text-sm text-blue-900">
        <p className="font-semibold">Recommended next step</p>
        <p className="mt-1 leading-6">{narrative.recommended_next_step || 'No recommendation yet.'}</p>
        <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
          <a
            href={designerHref}
            className="inline-flex h-8 items-center gap-1 rounded-md border border-blue-600 bg-blue-600 px-3 font-semibold text-white hover:bg-blue-700"
          >
            <ShieldCheck className="h-4 w-4" />
            Design observe-only control
          </a>
          {draftControls.length > 0 && draftControls[0].control_id ? (
            <span className="text-blue-900/80">
              Seeded draft: <span className="font-mono">{draftControls[0].control_id}</span>
            </span>
          ) : null}
        </div>
      </div>
    </section>
  )
}

function NarrativeCard({
  title,
  emphasis,
  tone,
  children,
}: {
  title: string
  emphasis?: boolean
  tone?: 'amber'
  children: React.ReactNode
}) {
  const toneClass =
    tone === 'amber'
      ? 'border-amber-200 bg-amber-50'
      : emphasis
        ? 'border-slate-200 bg-slate-50'
        : 'border-slate-200 bg-white'
  return (
    <article className={`rounded-md border p-3 shadow-sm ${toneClass}`}>
      <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">{title}</h4>
      <div className="mt-1">{children}</div>
    </article>
  )
}

function BulletList({ items, fallback }: { items?: string[]; fallback: string }) {
  if (!items || items.length === 0) {
    return <p className="text-sm text-slate-600">{fallback}</p>
  }
  return (
    <ul className="grid gap-1 pl-5 text-sm leading-6 text-slate-800 list-disc">
      {items.map((item, idx) => (
        <li key={`${idx}-${item}`}>{item}</li>
      ))}
    </ul>
  )
}

function edgeLabelBetween(edges: Edge[], from: string, to: string): string {
  const direct = edges.find((edge) => edge.from === from && edge.to === to)
  if (direct) return direct.label.replace(/_/g, ' ')
  const reverse = edges.find((edge) => edge.from === to && edge.to === from)
  if (reverse) return `${reverse.label.replace(/_/g, ' ')} (reverse)`
  return 'related to'
}

function RawEvidenceBlock({
  data,
  open,
  onToggle,
}: {
  data: EvidencePathResponse
  open: boolean
  onToggle: () => void
}) {
  const totalRaw =
    (data.raw_processes?.length || 0) +
    (data.raw_flows?.length || 0) +
    (data.raw_dns?.length || 0) +
    (data.raw_findings?.length || 0)
  return (
    <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center justify-between text-left text-xs font-semibold uppercase tracking-wide text-slate-600"
      >
        <span className="inline-flex items-center gap-2">
          {open ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          Raw evidence ({totalRaw})
        </span>
        <span className="text-slate-400">Bounded JSON view</span>
      </button>
      {open ? (
        <div className="mt-3 space-y-3 text-xs">
          <RawSection title={`Processes (${data.raw_processes?.length || 0})`} rows={(data.raw_processes || []).slice(0, 6)} />
          <RawSection title={`Flows (${data.raw_flows?.length || 0})`} rows={(data.raw_flows || []).slice(0, 6)} />
          <RawSection title={`DNS (${data.raw_dns?.length || 0})`} rows={(data.raw_dns || []).slice(0, 6)} />
          <RawSection title={`Findings (${data.raw_findings?.length || 0})`} rows={(data.raw_findings || []).slice(0, 6)} />
          <RawSection title={`Draft controls (${data.draft_controls?.length || 0})`} rows={data.draft_controls || []} />
        </div>
      ) : null}
    </section>
  )
}

function RawSection({ title, rows }: { title: string; rows: any[] }) {
  return (
    <div>
      <h4 className="font-semibold text-slate-700">{title}</h4>
      {rows.length === 0 ? (
        <p className="text-slate-500">none</p>
      ) : (
        <pre className="mt-1 max-h-64 overflow-auto whitespace-pre-wrap rounded bg-slate-900 p-3 font-mono text-[11px] text-slate-100">
{JSON.stringify(rows, null, 2)}
        </pre>
      )}
    </div>
  )
}
