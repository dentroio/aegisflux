'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  AlertTriangle,
  ArrowUpRight,
  Bot,
  CheckCircle2,
  Chrome,
  CircleSlash,
  Cpu,
  ExternalLink,
  Newspaper,
  RefreshCw,
  Send,
  Sparkles,
  Wrench,
} from 'lucide-react'
import {
  BoundedTable,
  EmptyState,
  FilterBar,
  KpiTile,
  SummaryStrip,
  WorkbenchHeader,
} from '@/components/workbench/primitives'
import { X } from 'lucide-react'

type Indicator = { type: string; value: string; note?: string }

type SuggestedDetection = {
  logic?: string
  scope?: string
  confidence?: string
  expected_noise?: string
  guard_rails?: string[]
}

export type ResearchItem = {
  id: string
  title: string
  source: string
  source_url?: string
  category: string
  summary: string
  indicators?: Indicator[]
  evidence_required?: string[]
  suggested_detection?: SuggestedDetection
  proposed_pack_id?: string
  linked_candidate_id?: string
  status: string
  risk_score: number
  operator_notes?: string
  published_at_ms?: number
  ingested_at_ms?: number
  updated_at_ms?: number
}

export type DetectionCandidate = {
  id: string
  source_research_id?: string
  title: string
  category?: string
  status: string
  pack_id?: string
  pack_version?: string
  rollout_status?: string
  operator_notes?: string
  reviewer_notes?: string
  expires_at_ms?: number
  rollback_plan?: string
  retirement_reason?: string
  rule?: SuggestedDetection
  quality_gate?: {
    required_evidence?: string[]
    expected_false_positives?: string
    has_simulation?: boolean
    has_reviewer_notes?: boolean
    has_expiration?: boolean
    has_rollback?: boolean
    missing_fields?: string[]
  }
  simulations?: Array<{
    id?: string
    at_ms?: number
    match_count?: number
    matched_device_count?: number
    top_indicators?: string[]
    window?: string
    confidence?: string
    notes?: string
  }>
  history?: Array<{ at_ms?: number; action?: string; from_status?: string; to_status?: string; note?: string }>
  created_at_ms?: number
  updated_at_ms?: number
}

const CANDIDATE_STAGES = [
  'candidate_new',
  'simulated',
  'reviewed',
  'signed',
  'deployed',
  'retired',
] as const

const CANDIDATE_STAGE_LABEL: Record<string, string> = {
  candidate_new: 'New',
  simulated: 'Simulated',
  reviewed: 'Reviewed',
  signed: 'Signed',
  deployed: 'Deployed',
  retired: 'Retired',
}

const CANDIDATE_STAGE_TONE: Record<string, string> = {
  candidate_new: 'border-blue-200 bg-blue-50 text-blue-900',
  simulated: 'border-amber-200 bg-amber-50 text-amber-900',
  reviewed: 'border-violet-200 bg-violet-50 text-violet-900',
  signed: 'border-emerald-200 bg-emerald-50 text-emerald-900',
  deployed: 'border-emerald-300 bg-emerald-100 text-emerald-900',
  retired: 'border-slate-300 bg-slate-100 text-slate-700',
}

type ResearchFeedResponse = {
  items?: ResearchItem[]
  total?: number
  status_counts?: Record<string, number>
  category_counts?: Record<string, number>
  generated_at_ms?: number
}

const STATUS_LABEL: Record<string, string> = {
  new: 'New',
  scoped: 'Scoped',
  ready_for_pack: 'Ready for pack',
  promoted: 'Promoted',
  declined: 'Declined',
}

const STATUS_TONE: Record<string, string> = {
  new: 'bg-blue-50 text-blue-800 border-blue-200',
  scoped: 'bg-amber-50 text-amber-900 border-amber-200',
  ready_for_pack: 'bg-emerald-50 text-emerald-800 border-emerald-200',
  promoted: 'bg-slate-100 text-slate-700 border-slate-200',
  declined: 'bg-slate-100 text-slate-500 border-slate-200',
}

const CATEGORY_ICON: Record<string, typeof Wrench> = {
  ai_general: Sparkles,
  local_model_runtime: Cpu,
  coding_agent: Bot,
  browser_ai_extension: Chrome,
  cli_agent: Bot,
  mcp_endpoint: Wrench,
}

type Filters = {
  category: string
  status: string
  search: string
}

export function ResearchFeedPanel({ embedded = false }: { embedded?: boolean }) {
  const [data, setData] = useState<ResearchFeedResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [filters, setFilters] = useState<Filters>({ category: '', status: '', search: '' })
  const [detailItem, setDetailItem] = useState<ResearchItem | null>(null)
  const [editingNotes, setEditingNotes] = useState<{ id: string; notes: string; status: string } | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [info, setInfo] = useState<string | null>(null)
  const [candidates, setCandidates] = useState<DetectionCandidate[]>([])
  const [candidateBusyId, setCandidateBusyId] = useState<string | null>(null)
  const [candidateDetail, setCandidateDetail] = useState<DetectionCandidate | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams()
      if (filters.category) params.set('category', filters.category)
      if (filters.status) params.set('status', filters.status)
      const url = `/api/actions/platform/research-feed${params.toString() ? `?${params.toString()}` : ''}`
      const res = await fetch(url, { cache: 'no-store' })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      const body = (await res.json()) as ResearchFeedResponse
      setData(body)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load research feed')
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [filters.category, filters.status])

  const loadCandidates = useCallback(async () => {
    try {
      const res = await fetch('/api/actions/platform/detection-candidates', { cache: 'no-store' })
      if (!res.ok) {
        setCandidates([])
        return
      }
      const body = (await res.json()) as { candidates?: DetectionCandidate[] }
      setCandidates(body?.candidates || [])
    } catch {
      setCandidates([])
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    void loadCandidates()
  }, [loadCandidates])

  async function candidateAction(id: string, path: string, body: Record<string, unknown> | null = null) {
    setCandidateBusyId(id)
    setError(null)
    setInfo(null)
    try {
      const res = await fetch(`/api/actions/platform/detection-candidates/${encodeURIComponent(id)}${path}`, {
        method: path === '' ? 'PATCH' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: body ? JSON.stringify(body) : '{}',
      })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      await loadCandidates()
      setInfo('Detection candidate updated.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update detection candidate')
    } finally {
      setCandidateBusyId(null)
    }
  }

  const filteredItems = useMemo(() => {
    const items = data?.items || []
    const search = filters.search.trim().toLowerCase()
    if (!search) return items
    return items.filter((item) => {
      const haystack = [
        item.title,
        item.source,
        item.summary,
        item.category,
        ...(item.indicators || []).map((indicator) => `${indicator.type} ${indicator.value}`),
      ]
        .join(' ')
        .toLowerCase()
      return haystack.includes(search)
    })
  }, [data, filters.search])

  const categoryOptions = useMemo(() => {
    return Object.keys(data?.category_counts || {}).sort()
  }, [data])

  async function patchItem(id: string, body: Record<string, unknown>) {
    setBusyId(id)
    setError(null)
    setInfo(null)
    try {
      const res = await fetch(`/api/actions/platform/research-feed/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      await load()
      setInfo('Research item updated.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update research item')
    } finally {
      setBusyId(null)
    }
  }

  async function promoteItem(item: ResearchItem) {
    if (!confirm(`Promote "${item.title}" to a governed observe-only detection opportunity?`)) {
      return
    }
    setBusyId(item.id)
    setError(null)
    setInfo(null)
    try {
      const res = await fetch(`/api/actions/platform/research-feed/${encodeURIComponent(item.id)}/promote`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      const body = await res.json()
      setInfo(`Promoted as governed pack ${body?.proposed_pack || 'pack'} — observe-only by default.`)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to promote research item')
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className={embedded ? '' : 'min-h-screen bg-slate-50'}>
      <WorkbenchHeader
        title="AI research feed"
        subtitle="Curated AI ecosystem intelligence with governed lifecycle from new to ready-for-pack."
        actions={
          <button
            type="button"
            onClick={() => void load()}
            disabled={loading}
            className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        }
      />

      <SummaryStrip>
        <KpiTile label="Total" value={data?.total || 0} />
        <KpiTile label="New" value={data?.status_counts?.new || 0} />
        <KpiTile label="Scoped" value={data?.status_counts?.scoped || 0} />
        <KpiTile label="Ready" value={data?.status_counts?.ready_for_pack || 0} />
      </SummaryStrip>

      <FilterBar>
        <CategoryButtons
          options={categoryOptions}
          counts={data?.category_counts || {}}
          activeId={filters.category}
          onSelect={(cat) => setFilters((prev) => ({ ...prev, category: prev.category === cat ? '' : cat }))}
        />
        <div className="ml-auto flex items-center gap-2">
          <select
            value={filters.status}
            onChange={(event) => setFilters((prev) => ({ ...prev, status: event.target.value }))}
            className="h-9 rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700"
          >
            <option value="">All statuses</option>
            {Object.keys(STATUS_LABEL).map((status) => (
              <option key={status} value={status}>{STATUS_LABEL[status]}</option>
            ))}
          </select>
          <input
            value={filters.search}
            onChange={(event) => setFilters((prev) => ({ ...prev, search: event.target.value }))}
            placeholder="Search title, indicator, source"
            className="h-9 w-64 rounded-md border border-slate-200 bg-white px-3 text-sm"
          />
        </div>
      </FilterBar>

      {error ? (
        <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">{error}</div>
      ) : null}
      {info ? (
        <div className="mb-4 rounded-md border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900">{info}</div>
      ) : null}

      <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        {loading && filteredItems.length === 0 ? (
          <EmptyState title="Loading research feed" message="Fetching curated AI intelligence." />
        ) : filteredItems.length === 0 ? (
          <EmptyState
            title="No research items match this view"
            message="Try clearing filters, or add a research item via API to populate this feed."
            hint="In lab/demo, run the suspicious automation scenario to seed a research item with indicators and guard rails."
            actions={
              <a
                href="/demo/scenarios"
                className="inline-flex h-8 items-center gap-1 rounded-md border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-700 hover:bg-slate-50"
              >
                View sample scenarios
              </a>
            }
          />
        ) : (
          <BoundedTable
            headers={['Title', 'Status', 'Risk', 'Source', 'Updated', 'Actions']}
            rows={filteredItems.map((item) => {
              const Icon = CATEGORY_ICON[item.category] || Newspaper
              return [
                <div key={`${item.id}-title`}>
                  <div className="flex items-center gap-2">
                    <Icon className="h-4 w-4 text-blue-600" />
                    <button
                      type="button"
                      onClick={() => setDetailItem(item)}
                      className="text-sm font-semibold text-slate-900 hover:text-blue-700"
                    >
                      {item.title}
                    </button>
                  </div>
                  <p className="mt-1 line-clamp-2 text-xs text-slate-600">{item.summary}</p>
                  {(item.indicators || []).length > 0 ? (
                    <div className="mt-1 flex flex-wrap gap-1 text-[11px] text-slate-700">
                      {(item.indicators || []).slice(0, 3).map((indicator, idx) => (
                        <span key={`${item.id}-ind-${idx}`} className="rounded bg-slate-100 px-2 py-0.5 font-mono">
                          {indicator.type}: {indicator.value}
                        </span>
                      ))}
                    </div>
                  ) : null}
                </div>,
                <span key={`${item.id}-status`} className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs ${STATUS_TONE[item.status] || STATUS_TONE.new}`}>
                  {STATUS_LABEL[item.status] || item.status}
                </span>,
                <span key={`${item.id}-risk`} className="text-xs font-semibold text-slate-700">
                  {item.risk_score}
                </span>,
                <span key={`${item.id}-source`} className="text-xs text-slate-700">
                  {item.source_url ? (
                    <a href={item.source_url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-blue-700 hover:text-blue-900">
                      {item.source}
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  ) : (
                    item.source
                  )}
                </span>,
                <span key={`${item.id}-updated`} className="text-xs text-slate-500">
                  {formatRelative(item.updated_at_ms || item.ingested_at_ms)}
                </span>,
                <div key={`${item.id}-actions`} className="flex flex-wrap gap-2 text-xs">
                  {item.status !== 'scoped' && item.status !== 'promoted' && item.status !== 'declined' ? (
                    <button
                      type="button"
                      disabled={busyId === item.id}
                      onClick={() => void patchItem(item.id, { status: 'scoped' })}
                      className="rounded-md border border-slate-200 px-2 py-1 hover:bg-slate-50"
                    >
                      Mark scoped
                    </button>
                  ) : null}
                  {item.status !== 'ready_for_pack' && item.status !== 'promoted' && item.status !== 'declined' ? (
                    <button
                      type="button"
                      disabled={busyId === item.id}
                      onClick={() => void patchItem(item.id, { status: 'ready_for_pack' })}
                      className="rounded-md border border-emerald-200 bg-emerald-50 px-2 py-1 text-emerald-800 hover:bg-emerald-100"
                    >
                      Ready for pack
                    </button>
                  ) : null}
                  {item.status !== 'promoted' && item.status !== 'declined' ? (
                    <button
                      type="button"
                      disabled={busyId === item.id}
                      onClick={() => void promoteItem(item)}
                      className="inline-flex items-center gap-1 rounded-md border border-blue-600 bg-blue-600 px-2 py-1 text-white hover:bg-blue-700"
                    >
                      <Send className="h-3 w-3" />
                      Promote
                    </button>
                  ) : null}
                  <button
                    type="button"
                    disabled={busyId === item.id}
                    onClick={() => setEditingNotes({ id: item.id, notes: item.operator_notes || '', status: item.status })}
                    className="rounded-md border border-slate-200 px-2 py-1 hover:bg-slate-50"
                  >
                    Notes
                  </button>
                  {item.status !== 'declined' && item.status !== 'promoted' ? (
                    <button
                      type="button"
                      disabled={busyId === item.id}
                      onClick={() => void patchItem(item.id, { status: 'declined' })}
                      className="inline-flex items-center gap-1 rounded-md border border-slate-200 px-2 py-1 text-slate-500 hover:bg-slate-50"
                    >
                      <CircleSlash className="h-3 w-3" />
                      Decline
                    </button>
                  ) : null}
                </div>,
              ]
            })}
          />
        )}
      </section>

      <DetectionWorkflowBoard
        candidates={candidates}
        busyId={candidateBusyId}
        onSimulate={(id) => void candidateAction(id, '/simulate')}
        onAdvance={(id, status, note) => void candidateAction(id, '', { status, note })}
        onRetire={(id, reason) => void candidateAction(id, '/retire', { reason })}
        onOpenDetail={setCandidateDetail}
      />

      {candidateDetail ? (
        <ResearchDetailModal title={`Candidate: ${candidateDetail.title}`} onClose={() => setCandidateDetail(null)}>
          <CandidateDetailContent candidate={candidateDetail} />
        </ResearchDetailModal>
      ) : null}

      {detailItem ? (
        <ResearchDetailModal title={detailItem.title} onClose={() => setDetailItem(null)}>
          <div className="grid gap-3 text-sm text-slate-800">
            <p>{detailItem.summary}</p>
            <div className="grid gap-2 md:grid-cols-2">
              <Field label="Source">
                {detailItem.source_url ? (
                  <a href={detailItem.source_url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-blue-700 hover:text-blue-900">
                    {detailItem.source}
                    <ExternalLink className="h-3 w-3" />
                  </a>
                ) : (
                  detailItem.source
                )}
              </Field>
              <Field label="Category">{detailItem.category}</Field>
              <Field label="Status">{STATUS_LABEL[detailItem.status] || detailItem.status}</Field>
              <Field label="Risk score">{detailItem.risk_score}</Field>
              <Field label="Proposed pack">{detailItem.proposed_pack_id || '—'}</Field>
              <Field label="Operator notes">{detailItem.operator_notes || '—'}</Field>
            </div>
            <Section title="Indicators">
              {(detailItem.indicators || []).length === 0 ? (
                <p className="text-xs text-slate-500">No indicators captured.</p>
              ) : (
                <ul className="grid gap-1 text-xs text-slate-700">
                  {(detailItem.indicators || []).map((indicator, idx) => (
                    <li key={`detail-ind-${idx}`} className="font-mono">
                      <span className="font-semibold text-slate-900">{indicator.type}</span>: {indicator.value}
                      {indicator.note ? <span className="ml-2 text-slate-500">{indicator.note}</span> : null}
                    </li>
                  ))}
                </ul>
              )}
            </Section>
            <Section title="Evidence required">
              <ul className="grid gap-1 pl-5 text-xs text-slate-700 list-disc">
                {(detailItem.evidence_required || []).map((req) => (
                  <li key={req}>{req}</li>
                ))}
              </ul>
            </Section>
            <Section title="Suggested detection">
              <div className="grid gap-1 text-xs text-slate-700">
                <p><span className="font-semibold">Logic:</span> <span className="font-mono">{detailItem.suggested_detection?.logic || '—'}</span></p>
                <p><span className="font-semibold">Scope:</span> <span className="font-mono">{detailItem.suggested_detection?.scope || '—'}</span></p>
                <p><span className="font-semibold">Confidence:</span> {detailItem.suggested_detection?.confidence || '—'}</p>
                <p><span className="font-semibold">Expected noise:</span> {detailItem.suggested_detection?.expected_noise || '—'}</p>
                {(detailItem.suggested_detection?.guard_rails || []).length > 0 ? (
                  <div>
                    <p className="font-semibold">Guard rails</p>
                    <ul className="grid gap-1 pl-5 list-disc">
                      {(detailItem.suggested_detection?.guard_rails || []).map((rule) => (
                        <li key={rule}>{rule}</li>
                      ))}
                    </ul>
                  </div>
                ) : null}
              </div>
            </Section>
            <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900">
              <div className="flex items-center gap-2 font-semibold">
                <AlertTriangle className="h-4 w-4" /> Governed
              </div>
              <p className="mt-1">
                Promotion creates an observe-only detection opportunity tied to a draft pack id. No detection is enabled in agents
                until reviewed and explicitly enabled.
              </p>
            </div>
          </div>
        </ResearchDetailModal>
      ) : null}

      {editingNotes ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 p-4">
          <div className="w-full max-w-xl rounded-xl border border-slate-200 bg-white p-4 shadow-xl">
            <h2 className="text-sm font-semibold text-slate-900">Operator notes & status</h2>
            <p className="mt-1 text-xs text-slate-500">Document why this item is being scoped or declined.</p>
            <div className="mt-3 grid gap-3">
              <label className="grid gap-1 text-xs font-medium text-slate-700">
                Status
                <select
                  value={editingNotes.status}
                  onChange={(event) => setEditingNotes({ ...editingNotes, status: event.target.value })}
                  className="h-9 rounded-md border border-slate-200 px-3 text-sm"
                >
                  {Object.keys(STATUS_LABEL).map((status) => (
                    <option key={status} value={status}>{STATUS_LABEL[status]}</option>
                  ))}
                </select>
              </label>
              <label className="grid gap-1 text-xs font-medium text-slate-700">
                Operator notes
                <textarea
                  value={editingNotes.notes}
                  onChange={(event) => setEditingNotes({ ...editingNotes, notes: event.target.value })}
                  className="h-32 rounded-md border border-slate-200 px-3 py-2 text-sm"
                />
              </label>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button
                type="button"
                className="h-9 rounded-md border border-slate-200 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
                onClick={() => setEditingNotes(null)}
              >
                Cancel
              </button>
              <button
                type="button"
                className="h-9 rounded-md border border-blue-600 bg-blue-600 px-3 text-sm font-semibold text-white hover:bg-blue-700"
                onClick={async () => {
                  if (!editingNotes) return
                  await patchItem(editingNotes.id, { operator_notes: editingNotes.notes, status: editingNotes.status })
                  setEditingNotes(null)
                }}
              >
                Save
              </button>
            </div>
          </div>
        </div>
      ) : null}

      <section className="mt-4 rounded-xl border border-slate-200 bg-white p-4 text-sm text-slate-700 shadow-sm">
        <h3 className="inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
          <CheckCircle2 className="h-4 w-4 text-emerald-600" />
          Governance
        </h3>
        <ul className="mt-2 grid gap-1 pl-5 text-xs text-slate-700 list-disc">
          <li>Research items are observe-only until promoted.</li>
          <li>Promotion creates a draft detection pack id; detections do not run until enabled in the pack.</li>
          <li>All transitions emit operational events for audit.</li>
        </ul>
        <p className="mt-3 text-xs text-slate-500">
          Need to feed AegisFlux from the outside? POST a research item to <span className="font-mono">/api/actions/platform/research-feed</span> with title, source,
          category, summary, and indicators.
        </p>
        <a
          href="/control/controls"
          className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-blue-700 hover:text-blue-900"
        >
          Open finding-to-control designer
          <ArrowUpRight className="h-3 w-3" />
        </a>
      </section>
    </div>
  )
}

function ResearchDetailModal({
  title,
  onClose,
  children,
}: {
  title: string
  onClose: () => void
  children: React.ReactNode
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4">
      <div className="max-h-[80vh] w-full max-w-3xl overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3">
          <h3 className="text-sm font-semibold text-slate-900">{title}</h3>
          <button
            type="button"
            className="inline-flex h-7 w-7 items-center justify-center rounded border border-slate-200 text-slate-500 hover:bg-slate-50"
            onClick={onClose}
            aria-label="Close detail"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="max-h-[calc(80vh-52px)] overflow-auto p-4">{children}</div>
      </div>
    </div>
  )
}

function CategoryButtons({
  options,
  counts,
  activeId,
  onSelect,
}: {
  options: string[]
  counts: Record<string, number>
  activeId: string
  onSelect: (id: string) => void
}) {
  if (options.length === 0) return null
  return (
    <div className="flex flex-wrap gap-2">
      {options.map((option) => {
        const Icon = CATEGORY_ICON[option] || Newspaper
        const active = activeId === option
        return (
          <button
            key={option}
            type="button"
            onClick={() => onSelect(option)}
            className={`inline-flex items-center gap-2 rounded-md border px-3 py-1 text-xs font-semibold transition ${
              active ? 'border-blue-600 bg-blue-600 text-white' : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50'
            }`}
          >
            <Icon className="h-3.5 w-3.5" />
            <span>{option.replace(/_/g, ' ')}</span>
            <span className={active ? 'text-blue-100' : 'text-slate-500'}>{counts[option] || 0}</span>
          </button>
        )
      })}
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">{title}</h3>
      <div className="mt-1">{children}</div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-xs font-semibold text-slate-500">{label}</dt>
      <dd className="text-sm text-slate-800">{children}</dd>
    </div>
  )
}

function formatRelative(ts?: number) {
  if (!ts) return 'n/a'
  const seconds = Math.max(0, Math.round((Date.now() - ts) / 1000))
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 48) return `${hours}h ago`
  return `${Math.round(hours / 24)}d ago`
}

function DetectionWorkflowBoard({
  candidates,
  busyId,
  onSimulate,
  onAdvance,
  onRetire,
  onOpenDetail,
}: {
  candidates: DetectionCandidate[]
  busyId: string | null
  onSimulate: (id: string) => void
  onAdvance: (id: string, status: string, note?: string) => void
  onRetire: (id: string, reason?: string) => void
  onOpenDetail: (candidate: DetectionCandidate) => void
}) {
  const grouped = useMemo(() => {
    const map: Record<string, DetectionCandidate[]> = {}
    for (const stage of CANDIDATE_STAGES) {
      map[stage] = []
    }
    for (const c of candidates) {
      const key = CANDIDATE_STAGES.includes(c.status as typeof CANDIDATE_STAGES[number])
        ? c.status
        : 'candidate_new'
      if (!map[key]) map[key] = []
      map[key].push(c)
    }
    return map
  }, [candidates])

  const totalCount = candidates.length

  return (
    <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex items-baseline justify-between">
        <div>
          <h2 className="text-lg font-semibold text-slate-900">Detection workflow</h2>
          <p className="text-xs text-slate-500">
            Promoted research is tracked here through simulation, review, signing, and rollout. Detections remain observe-only until signed.
          </p>
        </div>
        <span className="text-xs text-slate-500">{totalCount} candidate{totalCount === 1 ? '' : 's'}</span>
      </div>
      {totalCount === 0 ? (
        <div className="mt-4">
          <EmptyState
            title="No detection candidates yet"
            message="Promote a research item to create a governed observe-only detection candidate."
          />
        </div>
      ) : (
        <div className="mt-4 grid gap-3 lg:grid-cols-3 xl:grid-cols-6">
          {CANDIDATE_STAGES.map((stage) => {
            const list = grouped[stage] || []
            return (
              <div key={stage} className="rounded-lg border border-slate-200 bg-slate-50 p-3">
                <div className="flex items-center justify-between">
                  <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-semibold ${CANDIDATE_STAGE_TONE[stage] || 'border-slate-200 bg-white text-slate-700'}`}>
                    {CANDIDATE_STAGE_LABEL[stage] || stage}
                  </span>
                  <span className="text-xs text-slate-500">{list.length}</span>
                </div>
                <div className="mt-2 grid gap-2">
                  {list.length === 0 ? (
                    <p className="text-[11px] text-slate-400">No items.</p>
                  ) : (
                    list.map((candidate) => (
                      <CandidateCard
                        key={candidate.id}
                        candidate={candidate}
                        busy={busyId === candidate.id}
                        onSimulate={() => onSimulate(candidate.id)}
                        onAdvance={(status, note) => onAdvance(candidate.id, status, note)}
                        onRetire={(reason) => onRetire(candidate.id, reason)}
                        onOpenDetail={() => onOpenDetail(candidate)}
                      />
                    ))
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </section>
  )
}

function CandidateCard({
  candidate,
  busy,
  onSimulate,
  onAdvance,
  onRetire,
  onOpenDetail,
}: {
  candidate: DetectionCandidate
  busy: boolean
  onSimulate: () => void
  onAdvance: (status: string, note?: string) => void
  onRetire: (reason?: string) => void
  onOpenDetail: () => void
}) {
  const missing = candidate.quality_gate?.missing_fields || []
  const lastSim = (candidate.simulations || []).slice(-1)[0]
  return (
    <div className="rounded-md border border-slate-200 bg-white p-2 shadow-sm">
      <button
        type="button"
        onClick={onOpenDetail}
        className="block w-full text-left text-sm font-semibold text-slate-900 hover:text-blue-700"
      >
        {candidate.title}
      </button>
      <div className="mt-1 flex flex-wrap items-center gap-1 text-[11px] text-slate-600">
        {candidate.pack_id ? (
          <span className="rounded bg-slate-100 px-1.5 py-0.5 font-mono">{candidate.pack_id}</span>
        ) : null}
        {candidate.rollout_status && candidate.rollout_status !== 'none' ? (
          <span className="rounded bg-emerald-50 px-1.5 py-0.5 text-emerald-800">{candidate.rollout_status}</span>
        ) : null}
      </div>
      {lastSim ? (
        <p className="mt-1 text-[11px] text-slate-600">
          Last sim: {lastSim.match_count ?? 0} matches • {lastSim.matched_device_count ?? 0} devices
        </p>
      ) : null}
      {missing.length > 0 ? (
        <p className="mt-1 text-[11px] text-amber-700">Gate missing: {missing.slice(0, 3).join(', ')}{missing.length > 3 ? '…' : ''}</p>
      ) : null}
      <div className="mt-2 flex flex-wrap gap-1">
        {candidate.status === 'candidate_new' ? (
          <button
            type="button"
            disabled={busy}
            onClick={onSimulate}
            className="inline-flex items-center gap-1 rounded border border-slate-300 bg-white px-2 py-0.5 text-[11px] text-slate-700 hover:bg-slate-50 disabled:opacity-50"
          >
            Simulate
          </button>
        ) : null}
        {candidate.status === 'simulated' ? (
          <button
            type="button"
            disabled={busy}
            onClick={() => onAdvance('reviewed')}
            className="inline-flex items-center gap-1 rounded border border-slate-300 bg-white px-2 py-0.5 text-[11px] text-slate-700 hover:bg-slate-50 disabled:opacity-50"
          >
            Mark reviewed
          </button>
        ) : null}
        {candidate.status === 'reviewed' ? (
          <button
            type="button"
            disabled={busy || (candidate.quality_gate?.missing_fields?.length || 0) > 0}
            title={(candidate.quality_gate?.missing_fields?.length || 0) > 0 ? `Quality gate blocks signing: ${(candidate.quality_gate?.missing_fields || []).join(', ')}` : 'Sign and queue for rollout'}
            onClick={() => onAdvance('signed')}
            className="inline-flex items-center gap-1 rounded border border-emerald-300 bg-emerald-50 px-2 py-0.5 text-[11px] text-emerald-800 hover:bg-emerald-100 disabled:opacity-50"
          >
            Sign
          </button>
        ) : null}
        {candidate.status === 'signed' ? (
          <button
            type="button"
            disabled={busy}
            onClick={() => onAdvance('deployed')}
            className="inline-flex items-center gap-1 rounded border border-emerald-300 bg-emerald-50 px-2 py-0.5 text-[11px] text-emerald-800 hover:bg-emerald-100 disabled:opacity-50"
          >
            Mark deployed
          </button>
        ) : null}
        {candidate.status !== 'retired' ? (
          <button
            type="button"
            disabled={busy}
            onClick={() => {
              const reason = prompt('Retirement reason?', '') || ''
              onRetire(reason)
            }}
            className="inline-flex items-center gap-1 rounded border border-slate-300 bg-white px-2 py-0.5 text-[11px] text-slate-600 hover:bg-slate-50 disabled:opacity-50"
          >
            Retire
          </button>
        ) : null}
      </div>
    </div>
  )
}

function CandidateDetailContent({ candidate }: { candidate: DetectionCandidate }) {
  const lastSim = (candidate.simulations || []).slice(-1)[0]
  return (
    <div className="grid gap-3 text-sm text-slate-800">
      <div className="grid gap-2 md:grid-cols-2">
        <Field label="Status">{CANDIDATE_STAGE_LABEL[candidate.status] || candidate.status}</Field>
        <Field label="Pack id">{candidate.pack_id || '—'}</Field>
        <Field label="Pack version">{candidate.pack_version || '—'}</Field>
        <Field label="Rollout">{candidate.rollout_status || '—'}</Field>
        <Field label="Source research">{candidate.source_research_id || '—'}</Field>
        <Field label="Expires at">{candidate.expires_at_ms ? new Date(candidate.expires_at_ms).toLocaleString() : '—'}</Field>
      </div>
      <Section title="Rule">
        <div className="grid gap-1 text-xs text-slate-700">
          <p><span className="font-semibold">Logic:</span> <span className="font-mono">{candidate.rule?.logic || '—'}</span></p>
          <p><span className="font-semibold">Scope:</span> <span className="font-mono">{candidate.rule?.scope || '—'}</span></p>
          <p><span className="font-semibold">Confidence:</span> {candidate.rule?.confidence || '—'}</p>
          <p><span className="font-semibold">Expected noise:</span> {candidate.rule?.expected_noise || '—'}</p>
        </div>
      </Section>
      <Section title="Quality gate">
        <div className="grid gap-1 text-xs text-slate-700">
          <p>Has simulation: {candidate.quality_gate?.has_simulation ? 'yes' : 'no'}</p>
          <p>Reviewer notes: {candidate.quality_gate?.has_reviewer_notes ? 'yes' : 'no'}</p>
          <p>Expiration: {candidate.quality_gate?.has_expiration ? 'yes' : 'no'}</p>
          <p>Rollback plan: {candidate.quality_gate?.has_rollback ? 'yes' : 'no'}</p>
          {(candidate.quality_gate?.missing_fields || []).length > 0 ? (
            <p className="text-amber-700">Missing for signing: {(candidate.quality_gate?.missing_fields || []).join(', ')}</p>
          ) : null}
        </div>
      </Section>
      {lastSim ? (
        <Section title="Latest simulation">
          <div className="grid gap-1 text-xs text-slate-700">
            <p>Matches: {lastSim.match_count ?? 0} • Devices: {lastSim.matched_device_count ?? 0}</p>
            {lastSim.window ? <p>Window: {lastSim.window}</p> : null}
            {(lastSim.top_indicators || []).length > 0 ? (
              <p>Top indicators: {(lastSim.top_indicators || []).join(', ')}</p>
            ) : null}
            {lastSim.notes ? <p>{lastSim.notes}</p> : null}
          </div>
        </Section>
      ) : null}
      <Section title="History">
        {(candidate.history || []).length === 0 ? (
          <p className="text-xs text-slate-500">No history captured.</p>
        ) : (
          <ol className="grid gap-1 text-xs text-slate-700">
            {(candidate.history || []).slice().reverse().map((entry, idx) => (
              <li key={`hist-${idx}`} className="rounded border border-slate-200 bg-white p-2">
                <div className="flex items-center justify-between">
                  <span className="font-semibold">{entry.action || 'event'}</span>
                  <span className="text-[11px] text-slate-500">{entry.at_ms ? formatRelative(entry.at_ms) : ''}</span>
                </div>
                {entry.from_status || entry.to_status ? (
                  <p className="text-[11px] text-slate-500">
                    {entry.from_status || '—'} → {entry.to_status || '—'}
                  </p>
                ) : null}
                {entry.note ? <p className="mt-1">{entry.note}</p> : null}
              </li>
            ))}
          </ol>
        )}
      </Section>
      <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900">
        <div className="flex items-center gap-2 font-semibold">
          <AlertTriangle className="h-4 w-4" /> Observe-only
        </div>
        <p className="mt-1">
          This candidate is for governed simulation and review. Signing and deployment update platform records but never change endpoint enforcement state.
        </p>
      </div>
    </div>
  )
}
