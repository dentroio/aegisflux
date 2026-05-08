'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/lib/labAuth'

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

  if (!gate) {
    return <div className="flex min-h-screen items-center justify-center text-sm">Loading…</div>
  }

  return (
    <ConsoleShell activeNavId="controls" breadcrumbs={[{ label: 'Draft controls' }]} health={health} onLogout={onLogout}>
      <main className="mx-auto max-w-4xl px-4 py-6">
        <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          <strong>Observe-only.</strong> Drafts and simulations do not enforce policy. They project historical matches for review.
        </div>
        <div className="mb-6 flex flex-wrap gap-3">
          <Link href="/" className="text-sm font-semibold text-blue-700">
            Dashboard
          </Link>
        </div>
        <section className="mb-8 rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <h2 className="text-sm font-semibold text-gray-900">Create draft from finding id</h2>
          <div className="mt-3 grid gap-3 md:grid-cols-2">
            <label className="grid gap-1 text-xs font-medium text-gray-700">
              Source finding id
              <input className="rounded border px-2 py-1 text-sm" value={findingId} onChange={(e) => setFindingId(e.target.value)} />
            </label>
            <label className="grid gap-1 text-xs font-medium text-gray-700 md:col-span-2">
              Proposed observe-only action
              <input className="rounded border px-2 py-1 text-sm" value={action} onChange={(e) => setAction(e.target.value)} />
            </label>
          </div>
          <button type="button" className="btn btn-primary mt-3 h-9 px-3 text-sm" onClick={createDraft}>
            Save draft
          </button>
        </section>
        <section className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <h2 className="text-sm font-semibold text-gray-900">Drafts</h2>
          <div className="mt-3 flex flex-wrap items-end gap-2">
            <label className="grid gap-1 text-xs font-medium text-gray-700">
              Device id for simulation
              <input
                className="rounded border px-2 py-1 text-sm"
                value={deviceSim}
                onChange={(e) => setDeviceSim(e.target.value)}
                placeholder="win-lab / linux-lab device id"
              />
            </label>
          </div>
          <ul className="mt-4 space-y-3">
            {drafts.length === 0 ? (
              <li className="text-sm text-gray-500">No drafts yet.</li>
            ) : (
              drafts.map((d) => (
                <li key={d.id} className="rounded-lg border border-gray-100 bg-gray-50 px-3 py-2 text-sm">
                  <div className="font-mono text-xs text-gray-500">{d.id}</div>
                  <div className="mt-1 font-medium text-gray-900">{d.proposed_action}</div>
                  <div className="text-xs text-gray-600">Finding: {d.source_finding_id}</div>
                  {typeof d.simulation_match_count === 'number' && (
                    <div className="text-xs text-gray-600">Simulated matches: {d.simulation_match_count}</div>
                  )}
                  <button
                    type="button"
                    className="btn btn-secondary mt-2 h-8 px-2 text-xs"
                    onClick={() => simulate(d.id)}
                    disabled={!deviceSim.trim()}
                  >
                    Simulate history
                  </button>
                </li>
              ))
            )}
          </ul>
        </section>
      </main>
    </ConsoleShell>
  )
}
