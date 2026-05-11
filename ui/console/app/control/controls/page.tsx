'use client'

import { Suspense, useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import { DetailModal } from '@/components/workbench/primitives'
import { formatHash } from '@/shared/formatting'
import { FindingToControlPanel } from '@/components/FindingToControlPanel'
import { AlertTriangle, ArrowRight, RefreshCw, Search } from 'lucide-react'

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

const STATUS_LABELS: Record<string, { label: string; classes: string }> = {
  draft_observe_only: { label: 'Observe-only',  classes: 'bg-blue-50 text-blue-700 border-blue-200' },
  draft_in_review:   { label: 'In review',      classes: 'bg-amber-50 text-amber-700 border-amber-200' },
  draft_archived:    { label: 'Archived',        classes: 'bg-gray-100 text-gray-500 border-gray-200' },
}

function statusBadge(status: string) {
  const def = STATUS_LABELS[status] || { label: status, classes: 'bg-gray-100 text-gray-600 border-gray-200' }
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold border ${def.classes}`}>
      {def.label}
    </span>
  )
}

function DraftControlsPageInner() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const initialFindingId = (searchParams.get('finding_id') || '').trim()
  const initialDeviceId  = (searchParams.get('device_id')  || '').trim()

  const [gate, setGate]               = useState(false)
  const [drafts, setDrafts]           = useState<Draft[]>([])
  const [activeTab, setActiveTab]     = useState<'designer' | 'queue'>('designer')
  const [query, setQuery]             = useState('')
  const [deviceSim, setDeviceSim]     = useState('')
  const [detailModal, setDetailModal] = useState<{ title: string; payload: unknown } | null>(null)
  const [editing, setEditing]         = useState<{ id: string; notes: string; status: string } | null>(null)
  const [savingNotes, setSavingNotes] = useState(false)
  const [createOpen, setCreateOpen]   = useState(false)
  const [findingId, setFindingId]     = useState('lab-finding-001')
  const [action, setAction]           = useState('Observe-only DNS sinkhole review on matching queries')

  useEffect(() => {
    if (!readLabAuthenticated()) { router.replace('/'); return }
    setGate(true)
  }, [router])

  async function refresh() {
    const r = await fetch('/api/actions/platform/draft-controls', { cache: 'no-store' })
    if (!r.ok) return
    const j = await r.json()
    setDrafts(j.drafts || [])
  }

  useEffect(() => { if (gate) void refresh() }, [gate])

  async function simulate(id: string) {
    if (!deviceSim.trim()) return
    await fetch(`/api/actions/platform/draft-controls/${encodeURIComponent(id)}/simulate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ device_id: deviceSim.trim() }),
    })
    await refresh()
  }

  async function createDraft() {
    await fetch('/api/actions/platform/draft-controls', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source_finding_id: findingId, proposed_action: action }),
    })
    await refresh()
  }

  function onLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  if (!gate) return <div className="flex min-h-screen items-center justify-center text-sm">Loading…</div>

  const health: { label: string; tone: HealthTone; text: string } = {
    label: 'Observe-only', tone: 'amber', text: 'No enforcement active',
  }

  const queueCount      = drafts.length
  const simulatedCount  = drafts.filter((d) => typeof d.simulation_match_count === 'number').length
  const needsSim        = drafts.filter((d) => typeof d.simulation_match_count !== 'number').length
  const filteredDrafts  = drafts.filter(
    (d) => !query.trim() || JSON.stringify(d).toLowerCase().includes(query.trim().toLowerCase()),
  )

  return (
    <ConsoleShell
      activeNavId="controls"
      breadcrumbs={[{ label: 'Controls' }]}
      health={health}
      onLogout={onLogout}
    >
      <div className="mx-auto max-w-6xl px-5 py-6">

        {/* Page header */}
        <div className="flex items-center justify-between mb-5">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Controls</h1>
            <p className="text-sm text-gray-500 mt-0.5">
              Design observe-only controls from findings, simulate scope, and queue for audit promotion.
            </p>
          </div>
          <button
            type="button"
            onClick={() => void refresh()}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-[#1e3a5f] rounded-md hover:bg-[#162d4a] transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
        </div>

        {/* Observe-only safety banner */}
        <div className="mb-5 flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5 text-amber-600" />
          <div>
            <span className="font-semibold">Observe-only.</span> Drafts and simulations do not enforce policy.
            They project historical evidence matches for operator review. Use{' '}
            <a href="/control/audit-bundles" className="font-semibold underline hover:text-amber-700">
              Audit Bundles
            </a>{' '}
            to promote a reviewed draft to staged audit mode.
          </div>
        </div>

        {/* Stat cards */}
        <div className="mb-6 grid gap-4 md:grid-cols-3">
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Draft queue</p>
            <p className="mt-2 text-2xl font-semibold text-slate-900">{queueCount}</p>
            <p className="mt-2 text-sm text-slate-600">Total observe-only drafts</p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Simulated</p>
            <p className="mt-2 text-2xl font-semibold text-emerald-700">{simulatedCount}</p>
            <p className="mt-2 text-sm text-slate-600">Drafts with simulation results</p>
          </div>
          <div className={`rounded-xl border p-5 shadow-sm ${needsSim > 0 ? 'border-amber-200 bg-amber-50' : 'border-slate-200 bg-white'}`}>
            <p className={`text-xs font-semibold uppercase tracking-[0.2em] ${needsSim > 0 ? 'text-amber-700' : 'text-slate-500'}`}>
              Needs simulation
            </p>
            <p className={`mt-2 text-2xl font-semibold ${needsSim > 0 ? 'text-amber-800' : 'text-slate-900'}`}>{needsSim}</p>
            <p className={`mt-2 text-sm ${needsSim > 0 ? 'text-amber-700' : 'text-slate-600'}`}>
              {needsSim > 0 ? 'Run simulation before promotion' : 'All drafts simulated'}
            </p>
          </div>
        </div>

        {/* Underline tabs */}
        <div className="flex items-end border-b border-gray-200 mb-5">
          {(['designer', 'queue'] as const).map((tab) => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={`px-5 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap ${
                activeTab === tab
                  ? 'border-[#1e3a5f] text-[#1e3a5f]'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              {tab === 'designer' ? 'Finding-to-control designer' : `Draft queue (${queueCount})`}
            </button>
          ))}
        </div>

        {/* Designer tab */}
        {activeTab === 'designer' && (
          <FindingToControlPanel
            embedded
            initialFindingId={initialFindingId}
            initialDeviceId={initialDeviceId}
            onDraftCreated={() => void refresh()}
          />
        )}

        {/* Queue tab */}
        {activeTab === 'queue' && (
          <>
            {/* Filter bar */}
            <div className="bg-white rounded-lg shadow p-4 mb-5">
              <div className="flex flex-wrap items-center gap-3">
                <div className="relative flex-1 min-w-[200px] max-w-xs">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input
                    type="text"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    placeholder="Filter by finding, action, status…"
                    className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent"
                  />
                </div>
                <div className="relative flex-1 min-w-[200px] max-w-xs">
                  <input
                    type="text"
                    value={deviceSim}
                    onChange={(e) => setDeviceSim(e.target.value)}
                    placeholder="Device ID for simulation…"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent"
                  />
                </div>
                <button
                  type="button"
                  onClick={() => setCreateOpen(true)}
                  className="ml-auto inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-[#1e3a5f] rounded-md hover:bg-[#162d4a] transition-colors"
                >
                  New draft
                </button>
              </div>
            </div>

            {/* Draft table */}
            <div className="bg-white rounded-lg shadow overflow-hidden">
              {filteredDrafts.length === 0 ? (
                <div className="p-10 text-center text-gray-500 text-sm">
                  {drafts.length === 0
                    ? 'No drafts yet. Use the designer to create a control from a finding.'
                    : 'No drafts match your filter.'}
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Draft</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Proposed action</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Simulation</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {filteredDrafts.map((d) => (
                        <tr key={d.id} className="hover:bg-gray-50 transition-colors">
                          <td className="px-6 py-4">
                            <div className="font-mono text-xs text-gray-500">{formatHash(d.id)}</div>
                            <div className="text-xs text-gray-600 mt-0.5">
                              Finding: {d.source_finding_title || formatHash(d.source_finding_id)}
                            </div>
                            {d.source_device_id && (
                              <div className="text-[11px] text-gray-400 mt-0.5">Device: {d.source_device_id}</div>
                            )}
                          </td>
                          <td className="px-6 py-4 max-w-xs">
                            <p className="text-sm text-gray-900 line-clamp-2">{d.proposed_action}</p>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <div className="flex flex-col gap-1">
                              {statusBadge(d.status)}
                              {d.confidence && (
                                <span className="text-[11px] text-gray-400">conf: {d.confidence}</span>
                              )}
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            {typeof d.simulation_match_count === 'number' ? (
                              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
                                {d.simulation_match_count} matches
                              </span>
                            ) : (
                              <span className="text-xs text-gray-400">Not simulated</span>
                            )}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <div className="flex flex-wrap gap-2">
                              <button
                                type="button"
                                disabled={!deviceSim.trim()}
                                onClick={() => void simulate(d.id)}
                                className="text-xs font-semibold text-[#1e3a5f] hover:underline disabled:text-gray-300 disabled:cursor-not-allowed"
                              >
                                Simulate
                              </button>
                              <span className="text-gray-300">·</span>
                              <button
                                type="button"
                                onClick={() => setEditing({ id: d.id, notes: d.operator_notes || '', status: d.status })}
                                className="text-xs font-semibold text-[#1e3a5f] hover:underline"
                              >
                                Notes
                              </button>
                              <span className="text-gray-300">·</span>
                              <button
                                type="button"
                                onClick={() => setDetailModal({ title: `Draft ${d.id}`, payload: summarizeDraft(d) })}
                                className="text-xs font-semibold text-[#1e3a5f] hover:underline"
                              >
                                Detail
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            {/* Promote CTA */}
            {simulatedCount > 0 && (
              <div className="mt-4 flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900">
                <span>
                  {simulatedCount} draft{simulatedCount > 1 ? 's' : ''} ready for audit promotion.
                </span>
                <a
                  href="/control/audit-bundles"
                  className="inline-flex items-center gap-1 font-semibold text-emerald-700 hover:text-emerald-900"
                >
                  Go to Audit Bundles
                  <ArrowRight className="h-3.5 w-3.5" />
                </a>
              </div>
            )}
          </>
        )}
      </div>

      {/* Detail modal */}
      <DetailModal
        open={Boolean(detailModal)}
        title={detailModal?.title || 'Draft detail'}
        detail={detailModal?.payload || {}}
        onClose={() => setDetailModal(null)}
      />

      {/* Edit notes modal */}
      {editing && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 p-4">
          <div className="w-full max-w-xl rounded-xl border border-gray-200 bg-white p-6 shadow-xl">
            <h2 className="text-base font-semibold text-gray-900 mb-1">Operator notes & status</h2>
            <p className="text-xs text-gray-500 mb-4">Document scope rationale, pending review items, and what would change before promotion.</p>
            <div className="grid gap-4">
              <label className="grid gap-1.5 text-xs font-medium text-gray-700">
                Operator notes
                <textarea
                  className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent h-32 resize-none"
                  value={editing.notes}
                  onChange={(e) => setEditing({ ...editing, notes: e.target.value })}
                />
              </label>
              <label className="grid gap-1.5 text-xs font-medium text-gray-700">
                Status
                <select
                  className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent"
                  value={editing.status}
                  onChange={(e) => setEditing({ ...editing, status: e.target.value })}
                >
                  <option value="draft_observe_only">Observe-only</option>
                  <option value="draft_in_review">In review</option>
                  <option value="draft_archived">Archived</option>
                </select>
              </label>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button
                type="button"
                className="px-4 py-2 text-sm font-medium text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                onClick={() => setEditing(null)}
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={savingNotes}
                className="px-4 py-2 text-sm font-medium text-white bg-[#1e3a5f] rounded-lg hover:bg-[#162d4a] disabled:opacity-50 transition-colors"
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
      )}

      {/* Create draft modal */}
      {createOpen && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 p-4">
          <div className="w-full max-w-xl rounded-xl border border-gray-200 bg-white p-6 shadow-xl">
            <h2 className="text-base font-semibold text-gray-900 mb-4">Create manual draft</h2>
            <div className="grid gap-4">
              <label className="grid gap-1.5 text-xs font-medium text-gray-700">
                Source finding ID
                <input
                  className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent"
                  value={findingId}
                  onChange={(e) => setFindingId(e.target.value)}
                />
              </label>
              <label className="grid gap-1.5 text-xs font-medium text-gray-700">
                Proposed observe-only action
                <input
                  className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-[#1e3a5f] focus:border-transparent"
                  value={action}
                  onChange={(e) => setAction(e.target.value)}
                />
              </label>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button
                type="button"
                className="px-4 py-2 text-sm font-medium text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                onClick={() => setCreateOpen(false)}
              >
                Cancel
              </button>
              <button
                type="button"
                className="px-4 py-2 text-sm font-medium text-white bg-[#1e3a5f] rounded-lg hover:bg-[#162d4a] transition-colors"
                onClick={async () => { await createDraft(); setCreateOpen(false) }}
              >
                Save draft
              </button>
            </div>
          </div>
        </div>
      )}
    </ConsoleShell>
  )
}

function summarizeDraft(d: Draft) {
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
          Loading controls…
        </div>
      }
    >
      <DraftControlsPageInner />
    </Suspense>
  )
}
