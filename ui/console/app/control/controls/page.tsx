'use client'

import { Suspense, useEffect, useState } from 'react'
import Link from 'next/link'
import { useRouter, useSearchParams } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import { BoundedTable, DetailModal, EmptyState, FilterBar, KpiTile, SummaryStrip } from '@/components/workbench/primitives'
import { formatHash } from '@/shared/formatting'
import { FindingToControlPanel } from '@/components/FindingToControlPanel'

type Draft = {
  id: string
  status: string
  source_finding_id: string
  source_finding_title?: string
  source_device_id?: string
  proposed_action: string
  simulation_match_count?: number
  confidence?: string
  expected_breakage_risk?: string
  operator_notes?: string
  blast_radius?: string
  rollback_plan?: string
  evidence_refs?: string[]
  scope_selectors?: string[]
  blast_radius_notes?: string[]
  rollback_steps?: string[]
}

function DraftControlsPageInner() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const initialFindingId = (searchParams.get('finding_id') || '').trim()
  const initialDeviceId = (searchParams.get('device_id') || '').trim()
  const [gate, setGate] = useState(false)
  const [drafts, setDrafts] = useState<Draft[]>([])
  const [findingId, setFindingId] = useState('lab-finding-001')
  const [action, setAction] = useState('Observe-only DNS sinkhole review on matching queries')
  const [deviceSim, setDeviceSim] = useState('')
  const [query, setQuery] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [detailModal, setDetailModal] = useState<{ title: string; payload: unknown } | null>(null)
  const [showRaw, setShowRaw] = useState(false)
  const [activeMode, setActiveMode] = useState<'designer' | 'queue'>('designer')
  const [editing, setEditing] = useState<{ id: string; notes: string; status: string } | null>(null)
  const [savingNotes, setSavingNotes] = useState(false)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/')
      return
    }
    setGate(true)
  }, [router])

  async function refresh() {
    const r = await fetch('/api/actions/platform/draft-controls', { cache: 'no-store' })
    if (!r.ok) return
    const j = await r.json()
    setDrafts(j.drafts || [])
  }

  useEffect(() => {
    if (!gate) return undefined
    refresh()
    return undefined
  }, [gate])

  async function createDraft() {
    await fetch('/api/actions/platform/draft-controls', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source_finding_id: findingId, proposed_action: action }),
    })
    await refresh()
  }

  async function simulate(id: string) {
    if (!deviceSim.trim()) return
    await fetch(`/api/actions/platform/draft-controls/${encodeURIComponent(id)}/simulate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ device_id: deviceSim.trim() }),
    })
    await refresh()
  }

  function onLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  const health = { label: 'Observe-only', tone: 'amber' as HealthTone, text: 'No enforcement' }
  const queueCount = drafts.length
  const simulatedCount = drafts.filter((draft) => typeof draft.simulation_match_count === 'number').length
  const needingSimulation = drafts.filter((draft) => typeof draft.simulation_match_count !== 'number').length
  const filteredDrafts = drafts.filter((draft) =>
    !query.trim() || JSON.stringify(draft).toLowerCase().includes(query.trim().toLowerCase())
  )

  if (!gate) {
    return <div className="flex min-h-screen items-center justify-center text-sm">Loading…</div>
  }

  return (
    <ConsoleShell activeNavId="controls" breadcrumbs={[{ label: 'Draft controls' }]} health={health} onLogout={onLogout}>
      <main className="mx-auto max-w-5xl px-4 py-6">
        <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          <strong>Observe-only.</strong> Drafts and simulations do not enforce policy. They project historical matches for review.
        </div>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <Link href="/" className="text-sm font-semibold text-blue-700">Dashboard</Link>
            <Link href="/control/audit-bundles" className="text-sm font-semibold text-blue-700">
              Audit-mode bundles
            </Link>
          </div>
          <div className="inline-flex rounded-md border border-slate-200 bg-white p-0.5 text-xs font-semibold">
            <button
              type="button"
              onClick={() => setActiveMode('designer')}
              className={`h-8 rounded-md px-3 ${activeMode === 'designer' ? 'bg-blue-600 text-white' : 'text-slate-700 hover:bg-slate-50'}`}
            >
              Finding-to-control designer
            </button>
            <button
              type="button"
              onClick={() => setActiveMode('queue')}
              className={`h-8 rounded-md px-3 ${activeMode === 'queue' ? 'bg-blue-600 text-white' : 'text-slate-700 hover:bg-slate-50'}`}
            >
              Draft queue
            </button>
          </div>
        </div>

        {activeMode === 'designer' ? (
          <FindingToControlPanel
            embedded
            initialFindingId={initialFindingId}
            initialDeviceId={initialDeviceId}
            onDraftCreated={() => {
              void refresh()
            }}
          />
        ) : null}

        {activeMode === 'queue' ? (
          <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
            <span className="text-sm text-slate-500">{queueCount} drafts in the queue</span>
            <button type="button" className="btn btn-primary h-9 px-3 text-sm" onClick={() => setCreateOpen(true)}>
              New manual draft
            </button>
          </div>
        ) : null}

        {activeMode === 'queue' ? (
        <>
        <SummaryStrip>
          <KpiTile label="Draft queue" value={queueCount} />
          <KpiTile label="Simulated" value={simulatedCount} />
          <KpiTile label="Needs simulation" value={needingSimulation} />
        </SummaryStrip>

        <FilterBar>
          <input value={query} onChange={(e) => setQuery(e.target.value)} className="input h-9 min-w-[220px] max-w-md" placeholder="Filter draft id, finding, action..." />
          <input
            className="input h-9 min-w-[220px] max-w-md"
            value={deviceSim}
            onChange={(e) => setDeviceSim(e.target.value)}
            placeholder="Device id for simulation (win-lab / linux-lab)"
          />
        </FilterBar>

        <section className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          {filteredDrafts.length === 0 ? (
            <EmptyState title="No drafts" message={drafts.length === 0 ? 'Create a draft to start the queue.' : 'No drafts match the current filter.'} />
          ) : (
            <BoundedTable
              headers={['Draft', 'Action', 'Status', 'Simulation', 'Actions']}
              rows={filteredDrafts.map((d) => ([
                <div key={`${d.id}-draft`}>
                  <div className="font-mono text-xs text-gray-500">{formatHash(d.id)}</div>
                  <div className="text-xs text-gray-600">Finding: {d.source_finding_title || formatHash(d.source_finding_id)}</div>
                  {d.source_device_id ? <div className="text-[11px] text-gray-500">Device: {d.source_device_id}</div> : null}
                </div>,
                <span key={`${d.id}-action`} className="text-sm text-gray-900">{d.proposed_action}</span>,
                <div key={`${d.id}-status`} className="flex flex-col text-xs text-gray-600">
                  <span>{d.status}</span>
                  {d.confidence ? <span>conf: {d.confidence}</span> : null}
                </div>,
                <span key={`${d.id}-sim`} className="text-xs text-gray-600">
                  {typeof d.simulation_match_count === 'number' ? `${d.simulation_match_count} matches` : 'Not simulated'}
                </span>,
                <div key={`${d.id}-actions`} className="flex flex-wrap gap-2">
                  <button type="button" className="btn btn-secondary h-8 px-2 text-xs" onClick={() => simulate(d.id)} disabled={!deviceSim.trim()}>
                    Simulate
                  </button>
                  <button
                    type="button"
                    className="btn btn-secondary h-8 px-2 text-xs"
                    onClick={() => setEditing({ id: d.id, notes: d.operator_notes || '', status: d.status })}
                  >
                    Notes
                  </button>
                  <button
                    type="button"
                    className="btn btn-secondary h-8 px-2 text-xs"
                    onClick={() => {
                      setShowRaw(false)
                      setDetailModal({ title: `Draft ${d.id}`, payload: d })
                    }}
                  >
                    Detail
                  </button>
                </div>,
              ]))}
            />
          )}
        </section>
        </>
        ) : null}
      </main>

      <DetailModal
        open={Boolean(detailModal)}
        title={detailModal?.title || 'Draft detail'}
        detail={showRaw ? detailModal?.payload || {} : summarizeDraft(detailModal?.payload)}
        onClose={() => {
          setDetailModal(null)
          setShowRaw(false)
        }}
      />
      {detailModal ? (
        <div className="fixed bottom-4 right-4 z-10">
          <button type="button" className="btn btn-secondary h-8 px-2 text-xs" onClick={() => setShowRaw((v) => !v)}>
            {showRaw ? 'Show summary' : 'Show raw'}
          </button>
        </div>
      ) : null}

      {editing ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 p-4">
          <div className="w-full max-w-xl rounded-xl border border-gray-200 bg-white p-4 shadow-xl">
            <h2 className="text-sm font-semibold text-gray-900">Operator notes & status</h2>
            <p className="mt-1 text-xs text-gray-500">Document why this scope was chosen, what review is pending, and what would change before promotion.</p>
            <div className="mt-3 grid gap-3">
              <label className="grid gap-1 text-xs font-medium text-gray-700">
                Operator notes
                <textarea className="input h-32 px-3 py-2 text-sm" value={editing.notes} onChange={(e) => setEditing({ ...editing, notes: e.target.value })} />
              </label>
              <label className="grid gap-1 text-xs font-medium text-gray-700">
                Status
                <select
                  className="input h-9"
                  value={editing.status}
                  onChange={(e) => setEditing({ ...editing, status: e.target.value })}
                >
                  <option value="draft_observe_only">draft_observe_only</option>
                  <option value="draft_in_review">draft_in_review</option>
                  <option value="draft_archived">draft_archived</option>
                </select>
              </label>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button type="button" className="btn btn-secondary h-9 px-3 text-sm" onClick={() => setEditing(null)}>Cancel</button>
              <button
                type="button"
                className="btn btn-primary h-9 px-3 text-sm"
                disabled={savingNotes}
                onClick={async () => {
                  if (!editing) return
                  setSavingNotes(true)
                  try {
                    await fetch(`/api/actions/platform/draft-controls/${encodeURIComponent(editing.id)}`, {
                      method: 'PATCH',
                      headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({ operator_notes: editing.notes, status: editing.status }),
                    })
                    await refresh()
                    setEditing(null)
                  } finally {
                    setSavingNotes(false)
                  }
                }}
              >
                {savingNotes ? 'Saving…' : 'Save notes'}
              </button>
            </div>
          </div>
        </div>
      ) : null}

      {createOpen ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 p-4">
          <div className="w-full max-w-xl rounded-xl border border-gray-200 bg-white p-4 shadow-xl">
            <h2 className="text-sm font-semibold text-gray-900">Create draft from finding id</h2>
            <div className="mt-3 grid gap-3 md:grid-cols-2">
              <label className="grid gap-1 text-xs font-medium text-gray-700">
                Source finding id
                <input className="input h-9" value={findingId} onChange={(e) => setFindingId(e.target.value)} />
              </label>
              <label className="grid gap-1 text-xs font-medium text-gray-700 md:col-span-2">
                Proposed observe-only action
                <input className="input h-9" value={action} onChange={(e) => setAction(e.target.value)} />
              </label>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button type="button" className="btn btn-secondary h-9 px-3 text-sm" onClick={() => setCreateOpen(false)}>Cancel</button>
              <button
                type="button"
                className="btn btn-primary h-9 px-3 text-sm"
                onClick={async () => {
                  await createDraft()
                  setCreateOpen(false)
                }}
              >
                Save draft
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </ConsoleShell>
  )
}

function summarizeDraft(payload: unknown) {
  if (!payload || typeof payload !== 'object') return payload
  const d = payload as Draft
  return {
    id: d.id,
    status: d.status,
    confidence: d.confidence,
    expected_breakage_risk: d.expected_breakage_risk,
    source_finding_id: d.source_finding_id,
    source_finding_title: d.source_finding_title,
    source_device_id: d.source_device_id,
    proposed_action: d.proposed_action,
    scope_selectors: d.scope_selectors,
    blast_radius: d.blast_radius,
    blast_radius_notes: d.blast_radius_notes,
    rollback_plan: d.rollback_plan,
    rollback_steps: d.rollback_steps,
    operator_notes: d.operator_notes,
    simulation_match_count: d.simulation_match_count ?? 'not simulated',
    evidence_refs: d.evidence_refs,
  }
}

export default function DraftControlsPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center bg-slate-50 text-sm text-slate-600">
          Loading draft controls…
        </div>
      }
    >
      <DraftControlsPageInner />
    </Suspense>
  )
}
