'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import { BoundedTable, DetailModal, EmptyState, FilterBar, KpiTile, SummaryStrip } from '@/components/workbench/primitives'
import { formatHash } from '@/shared/formatting'

type Draft = {
  id: string
  status: string
  source_finding_id: string
  proposed_action: string
  simulation_match_count?: number
}

export default function DraftControlsPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)
  const [drafts, setDrafts] = useState<Draft[]>([])
  const [findingId, setFindingId] = useState('lab-finding-001')
  const [action, setAction] = useState('Observe-only DNS sinkhole review on matching queries')
  const [deviceSim, setDeviceSim] = useState('')
  const [query, setQuery] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [detailModal, setDetailModal] = useState<{ title: string; payload: unknown } | null>(null)
  const [showRaw, setShowRaw] = useState(false)

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
      <main className="mx-auto max-w-4xl px-4 py-6">
        <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          <strong>Observe-only.</strong> Drafts and simulations do not enforce policy. They project historical matches for review.
        </div>
        <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
          <Link href="/" className="text-sm font-semibold text-blue-700">Dashboard</Link>
          <button type="button" className="btn btn-primary h-9 px-3 text-sm" onClick={() => setCreateOpen(true)}>
            New draft
          </button>
        </div>

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
                  <div className="text-xs text-gray-600">Finding: {formatHash(d.source_finding_id)}</div>
                </div>,
                <span key={`${d.id}-action`} className="text-sm text-gray-900">{d.proposed_action}</span>,
                <span key={`${d.id}-status`} className="text-xs text-gray-600">{d.status}</span>,
                <span key={`${d.id}-sim`} className="text-xs text-gray-600">
                  {typeof d.simulation_match_count === 'number' ? `${d.simulation_match_count} matches` : 'Not simulated'}
                </span>,
                <div key={`${d.id}-actions`} className="flex gap-2">
                  <button type="button" className="btn btn-secondary h-8 px-2 text-xs" onClick={() => simulate(d.id)} disabled={!deviceSim.trim()}>
                    Simulate
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
    finding: d.source_finding_id,
    action: d.proposed_action,
    simulation_match_count: d.simulation_match_count ?? 'not simulated',
  }
}
