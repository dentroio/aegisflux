'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { AlertTriangle, CheckCircle2, Clock, RefreshCw } from 'lucide-react'
import {
  BoundedTable,
  EmptyState,
  KpiTile,
  SummaryStrip,
  WorkbenchHeader,
} from '@/components/workbench/primitives'

type EndpointStatus = {
  device_id: string
  status: 'pending' | 'accepted' | 'rejected' | 'incompatible' | 'stale'
  reason?: string
  agent_version?: string
  reported_at_ms?: number
  last_match_at_ms?: number
}

type AuditMatch = {
  id: string
  device_id?: string
  process?: string
  at_ms?: number
  indicator?: string
  detail?: string
}

type AuditEvent = {
  id: string
  at_ms?: number
  action: string
  from_status?: string
  to_status?: string
  note?: string
}

type AuditBundle = {
  id: string
  version: string
  mode: string
  title: string
  description?: string
  scope?: string[]
  expected_match_telemetry?: string[]
  approval_refs?: string[]
  rollback_notes?: string
  source_candidate_id?: string
  source_draft_id?: string
  status: string
  staged_at_ms?: number
  expires_at_ms?: number
  created_at_ms?: number
  updated_at_ms?: number
  endpoint_statuses?: EndpointStatus[]
  matches?: AuditMatch[]
  history?: AuditEvent[]
}

type CollectionResponse = {
  bundles?: AuditBundle[]
  total?: number
  status_counts?: Record<string, number>
}

const STATUS_TONE: Record<string, string> = {
  draft: 'bg-slate-100 text-slate-700 border-slate-200',
  staged: 'bg-emerald-50 text-emerald-800 border-emerald-200',
  expired: 'bg-amber-50 text-amber-900 border-amber-200',
  revoked: 'bg-slate-100 text-slate-500 border-slate-200',
}

const ENDPOINT_TONE: Record<EndpointStatus['status'], string> = {
  pending: 'bg-slate-50 text-slate-700 border-slate-200',
  accepted: 'bg-emerald-50 text-emerald-800 border-emerald-200',
  rejected: 'bg-rose-50 text-rose-800 border-rose-200',
  incompatible: 'bg-amber-50 text-amber-900 border-amber-200',
  stale: 'bg-amber-50 text-amber-900 border-amber-200',
}

export function AuditBundlesPanel() {
  const [bundles, setBundles] = useState<AuditBundle[]>([])
  const [counts, setCounts] = useState<Record<string, number>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [info, setInfo] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [selected, setSelected] = useState<AuditBundle | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [draft, setDraft] = useState({
    title: '',
    description: '',
    scope: '',
    expected_match_telemetry: '',
    rollback_notes: '',
  })

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/actions/platform/audit-bundles', { cache: 'no-store' })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      const body = (await res.json()) as CollectionResponse
      setBundles(body?.bundles || [])
      setCounts(body?.status_counts || {})
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load audit bundles')
      setBundles([])
      setCounts({})
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function postJson(path: string, body: unknown) {
    setBusyId(path)
    setError(null)
    setInfo(null)
    try {
      const res = await fetch(path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body || {}),
      })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      await load()
      setInfo('Audit bundle updated.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update audit bundle')
    } finally {
      setBusyId(null)
    }
  }

  const summary = useMemo(() => {
    const totals = { staged: 0, accepted: 0, withMatches: 0, endpoints: 0 }
    for (const b of bundles) {
      if (b.status === 'staged') totals.staged++
      for (const e of b.endpoint_statuses || []) {
        totals.endpoints++
        if (e.status === 'accepted') totals.accepted++
        if (e.last_match_at_ms) totals.withMatches++
      }
    }
    return totals
  }, [bundles])

  return (
    <div>
      <WorkbenchHeader
        title="Audit-mode bundles"
        subtitle="Foundation for safe enforcement. Bundles are observe-only. Endpoints accept the bundle, evaluate it, and report match telemetry."
        actions={
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void load()}
              className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh
            </button>
            <button
              type="button"
              onClick={() => setCreateOpen(true)}
              className="inline-flex h-9 items-center gap-2 rounded-md border border-blue-600 bg-blue-600 px-3 text-sm font-semibold text-white hover:bg-blue-700"
            >
              New bundle
            </button>
          </div>
        }
      />

      <SummaryStrip>
        <KpiTile label="Total bundles" value={bundles.length} />
        <KpiTile label="Staged" value={summary.staged} />
        <KpiTile label="Endpoints accepted" value={summary.accepted} />
        <KpiTile label="Endpoints with matches" value={summary.withMatches} />
      </SummaryStrip>

      {error ? (
        <div className="mb-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">{error}</div>
      ) : null}
      {info ? (
        <div className="mb-3 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900">{info}</div>
      ) : null}

      <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        {loading && bundles.length === 0 ? (
          <EmptyState title="Loading audit bundles" message="Fetching staged bundles and endpoint status." />
        ) : bundles.length === 0 ? (
          <EmptyState
            title="No audit bundles yet"
            message="Audit bundles capture scope, expected match telemetry, expiration, and rollback notes before any enforcement could be considered."
            hint="In the lab, create a bundle from a draft control or detection candidate to exercise the staging contract."
            actions={
              <button
                type="button"
                onClick={() => setCreateOpen(true)}
                className="inline-flex h-8 items-center gap-1 rounded-md border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-700 hover:bg-slate-50"
              >
                Create audit bundle
              </button>
            }
          />
        ) : (
          <BoundedTable
            headers={['Bundle', 'Status', 'Endpoints', 'Matches', 'Updated', 'Actions']}
            rows={bundles.map((bundle) => {
              const endpointStats = summarizeEndpoints(bundle.endpoint_statuses || [])
              const matches = (bundle.matches || []).length
              return [
                <button
                  key={`${bundle.id}-title`}
                  type="button"
                  onClick={() => setSelected(bundle)}
                  className="text-left"
                >
                  <div className="text-sm font-semibold text-slate-900 hover:text-blue-700">{bundle.title}</div>
                  <div className="text-[11px] font-mono text-slate-500">{bundle.id}</div>
                  <div className="mt-1 text-[11px] text-slate-500">
                    Mode: <span className="font-mono">{bundle.mode}</span> · Version: {bundle.version}
                  </div>
                </button>,
                <span
                  key={`${bundle.id}-status`}
                  className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-semibold ${
                    STATUS_TONE[bundle.status] || 'bg-slate-100 text-slate-700 border-slate-200'
                  }`}
                >
                  {bundle.status}
                </span>,
                <div key={`${bundle.id}-eps`} className="text-xs text-slate-700">
                  {endpointStats.total} total · {endpointStats.accepted} accepted
                  {endpointStats.rejected ? ` · ${endpointStats.rejected} rejected` : ''}
                  {endpointStats.incompatible ? ` · ${endpointStats.incompatible} incompatible` : ''}
                  {endpointStats.stale ? ` · ${endpointStats.stale} stale` : ''}
                </div>,
                <span key={`${bundle.id}-matches`} className="text-xs text-slate-700">
                  {matches}
                </span>,
                <span key={`${bundle.id}-updated`} className="text-xs text-slate-500">
                  {bundle.updated_at_ms ? formatRelative(bundle.updated_at_ms) : '—'}
                </span>,
                <div key={`${bundle.id}-actions`} className="flex flex-wrap gap-2">
                  {bundle.status === 'draft' ? (
                    <button
                      type="button"
                      disabled={busyId !== null}
                      onClick={() => void postJson(`/api/actions/platform/audit-bundles/${encodeURIComponent(bundle.id)}/stage`, {})}
                      className="inline-flex items-center gap-1 rounded border border-emerald-300 bg-emerald-50 px-2 py-1 text-[11px] font-semibold text-emerald-800 hover:bg-emerald-100 disabled:opacity-50"
                    >
                      Stage
                    </button>
                  ) : null}
                  {bundle.status === 'staged' ? (
                    <button
                      type="button"
                      disabled={busyId !== null}
                      onClick={() => {
                        const deviceId = prompt('Device id reporting status?', '') || ''
                        if (!deviceId) return
                        const status = prompt('Status (accepted/rejected/incompatible/stale)?', 'accepted') || ''
                        if (!status) return
                        void postJson(`/api/actions/platform/audit-bundles/${encodeURIComponent(bundle.id)}/status`, {
                          device_id: deviceId,
                          status,
                        })
                      }}
                      className="inline-flex items-center gap-1 rounded border border-slate-300 bg-white px-2 py-1 text-[11px] font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50"
                    >
                      Report status
                    </button>
                  ) : null}
                  {bundle.status === 'staged' ? (
                    <button
                      type="button"
                      disabled={busyId !== null}
                      onClick={() => {
                        const indicator = prompt('Indicator (e.g. listen_port==11434)?', '') || ''
                        const deviceId = prompt('Device id?', '') || ''
                        void postJson(`/api/actions/platform/audit-bundles/${encodeURIComponent(bundle.id)}/match`, {
                          device_id: deviceId,
                          indicator,
                        })
                      }}
                      className="inline-flex items-center gap-1 rounded border border-slate-300 bg-white px-2 py-1 text-[11px] font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50"
                    >
                      Record match
                    </button>
                  ) : null}
                  {bundle.status !== 'revoked' ? (
                    <button
                      type="button"
                      disabled={busyId !== null}
                      onClick={() => {
                        const note = prompt('Revoke note?', '') || ''
                        void postJson(`/api/actions/platform/audit-bundles/${encodeURIComponent(bundle.id)}/revoke`, { note })
                      }}
                      className="inline-flex items-center gap-1 rounded border border-slate-300 bg-white px-2 py-1 text-[11px] font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                    >
                      Revoke
                    </button>
                  ) : null}
                </div>,
              ]
            })}
          />
        )}
      </section>

      {selected ? <BundleDetailModal bundle={selected} onClose={() => setSelected(null)} /> : null}

      {createOpen ? (
        <CreateBundleModal
          draft={draft}
          onChange={setDraft}
          onClose={() => setCreateOpen(false)}
          onSubmit={async () => {
            const body = {
              title: draft.title.trim(),
              description: draft.description.trim(),
              scope: draft.scope.split(/\n+/).map((s) => s.trim()).filter(Boolean),
              expected_match_telemetry: draft.expected_match_telemetry.split(/\n+/).map((s) => s.trim()).filter(Boolean),
              rollback_notes: draft.rollback_notes.trim(),
              mode: 'audit',
            }
            try {
              const res = await fetch('/api/actions/platform/audit-bundles', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
              })
              if (!res.ok) {
                const text = await res.text().catch(() => '')
                throw new Error(text || `HTTP ${res.status}`)
              }
              await load()
              setCreateOpen(false)
              setDraft({ title: '', description: '', scope: '', expected_match_telemetry: '', rollback_notes: '' })
              setInfo('Audit bundle created in draft.')
            } catch (err) {
              setError(err instanceof Error ? err.message : 'Failed to create audit bundle')
            }
          }}
        />
      ) : null}
    </div>
  )
}

function summarizeEndpoints(list: EndpointStatus[]) {
  const out = { total: list.length, pending: 0, accepted: 0, rejected: 0, incompatible: 0, stale: 0, withMatches: 0 }
  for (const st of list) {
    switch (st.status) {
      case 'pending':
        out.pending++
        break
      case 'accepted':
        out.accepted++
        break
      case 'rejected':
        out.rejected++
        break
      case 'incompatible':
        out.incompatible++
        break
      case 'stale':
        out.stale++
        break
    }
    if (st.last_match_at_ms) out.withMatches++
  }
  return out
}

function formatRelative(ts: number) {
  const seconds = Math.max(0, Math.round((Date.now() - ts) / 1000))
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 48) return `${hours}h ago`
  return `${Math.round(hours / 24)}d ago`
}

function BundleDetailModal({ bundle, onClose }: { bundle: AuditBundle; onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 p-4">
      <div className="w-full max-w-3xl rounded-xl border border-slate-200 bg-white p-5 shadow-xl">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h3 className="text-lg font-semibold text-slate-900">{bundle.title}</h3>
            <p className="mt-1 text-xs text-slate-500 font-mono">{bundle.id}</p>
            <p className="mt-1 text-xs text-slate-600">Mode: {bundle.mode} · Status: {bundle.status} · Version: {bundle.version}</p>
          </div>
          <button type="button" onClick={onClose} className="text-sm text-slate-500 hover:text-slate-800">
            Close
          </button>
        </div>
        {bundle.description ? <p className="mt-3 text-sm text-slate-700">{bundle.description}</p> : null}
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <Section title="Scope">
            <List items={bundle.scope || []} fallback="No scope set." />
          </Section>
          <Section title="Expected match telemetry">
            <List items={bundle.expected_match_telemetry || []} fallback="No expected telemetry recorded." />
          </Section>
          <Section title="Approval refs">
            <List items={bundle.approval_refs || []} fallback="No approvals recorded." />
          </Section>
          <Section title="Rollback notes">
            <p className="text-sm text-slate-700">{bundle.rollback_notes || 'No rollback notes recorded.'}</p>
          </Section>
        </div>
        <Section title="Endpoint statuses">
          {(bundle.endpoint_statuses || []).length === 0 ? (
            <p className="text-xs text-slate-500">No endpoints have reported status yet.</p>
          ) : (
            <div className="grid gap-1 text-xs text-slate-700">
              {(bundle.endpoint_statuses || []).map((st) => (
                <div key={st.device_id} className="flex flex-wrap items-center gap-2 rounded border border-slate-200 bg-white p-2">
                  <span className="font-mono text-slate-700">{st.device_id}</span>
                  <span className={`inline-flex items-center rounded-full border px-2 py-0.5 ${ENDPOINT_TONE[st.status] || 'border-slate-200 bg-slate-50'}`}>
                    {st.status}
                  </span>
                  {st.agent_version ? <span className="text-slate-500">v{st.agent_version}</span> : null}
                  {st.reason ? <span className="text-slate-500">{st.reason}</span> : null}
                  {st.last_match_at_ms ? <span className="text-emerald-700">match {formatRelative(st.last_match_at_ms)}</span> : null}
                </div>
              ))}
            </div>
          )}
        </Section>
        <Section title="Matches">
          {(bundle.matches || []).length === 0 ? (
            <p className="text-xs text-slate-500">No match telemetry yet.</p>
          ) : (
            <ul className="grid gap-1 text-xs text-slate-700">
              {(bundle.matches || []).slice(-10).reverse().map((m) => (
                <li key={m.id} className="rounded border border-slate-200 bg-white p-2">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold">{m.indicator || 'match'}</span>
                    <span className="text-[11px] text-slate-500">{m.at_ms ? formatRelative(m.at_ms) : ''}</span>
                  </div>
                  {m.device_id ? <p className="text-[11px] text-slate-500">device: {m.device_id}</p> : null}
                  {m.process ? <p className="text-[11px] text-slate-500">process: {m.process}</p> : null}
                  {m.detail ? <p className="mt-1 text-slate-700">{m.detail}</p> : null}
                </li>
              ))}
            </ul>
          )}
        </Section>
        <Section title="History">
          {(bundle.history || []).length === 0 ? (
            <p className="text-xs text-slate-500">No history yet.</p>
          ) : (
            <ol className="grid gap-1 text-xs text-slate-700">
              {(bundle.history || []).slice().reverse().map((entry) => (
                <li key={entry.id} className="rounded border border-slate-200 bg-white p-2">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold">{entry.action}</span>
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
        <div className="mt-4 rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900">
          <div className="flex items-center gap-2 font-semibold">
            <AlertTriangle className="h-4 w-4" /> Audit-only
          </div>
          <p className="mt-1">
            Audit-mode bundles never block, deny, or quarantine. Endpoints evaluate the bundle and report observe-only
            telemetry. Revoke a bundle to remove it from rollout.
          </p>
        </div>
      </div>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mt-4">
      <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">{title}</h4>
      <div className="mt-1">{children}</div>
    </div>
  )
}

function List({ items, fallback }: { items: string[]; fallback: string }) {
  if (!items || items.length === 0) {
    return <p className="text-xs text-slate-500">{fallback}</p>
  }
  return (
    <ul className="grid gap-1 pl-5 text-sm text-slate-700 list-disc">
      {items.map((item, idx) => (
        <li key={`${item}-${idx}`}>{item}</li>
      ))}
    </ul>
  )
}

function CreateBundleModal({
  draft,
  onChange,
  onClose,
  onSubmit,
}: {
  draft: { title: string; description: string; scope: string; expected_match_telemetry: string; rollback_notes: string }
  onChange: (next: { title: string; description: string; scope: string; expected_match_telemetry: string; rollback_notes: string }) => void
  onClose: () => void
  onSubmit: () => void | Promise<void>
}) {
  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 p-4">
      <div className="w-full max-w-2xl rounded-xl border border-slate-200 bg-white p-5 shadow-xl">
        <h3 className="text-lg font-semibold text-slate-900">Create audit bundle</h3>
        <p className="mt-1 text-xs text-slate-500">
          Audit-mode bundles are observe-only by design. Endpoints will receive the bundle, evaluate scope, and
          report match telemetry without blocking.
        </p>
        <div className="mt-4 grid gap-3">
          <label className="grid gap-1 text-xs font-semibold text-slate-700">
            Title
            <input
              className="input h-9"
              value={draft.title}
              onChange={(e) => onChange({ ...draft, title: e.target.value })}
            />
          </label>
          <label className="grid gap-1 text-xs font-semibold text-slate-700">
            Description
            <textarea
              className="input h-20"
              value={draft.description}
              onChange={(e) => onChange({ ...draft, description: e.target.value })}
            />
          </label>
          <label className="grid gap-1 text-xs font-semibold text-slate-700">
            Scope (one selector per line)
            <textarea
              className="input h-20 font-mono text-xs"
              value={draft.scope}
              onChange={(e) => onChange({ ...draft, scope: e.target.value })}
            />
          </label>
          <label className="grid gap-1 text-xs font-semibold text-slate-700">
            Expected match telemetry (one per line)
            <textarea
              className="input h-20 font-mono text-xs"
              value={draft.expected_match_telemetry}
              onChange={(e) => onChange({ ...draft, expected_match_telemetry: e.target.value })}
            />
          </label>
          <label className="grid gap-1 text-xs font-semibold text-slate-700">
            Rollback notes
            <textarea
              className="input h-20"
              value={draft.rollback_notes}
              onChange={(e) => onChange({ ...draft, rollback_notes: e.target.value })}
            />
          </label>
        </div>
        <div className="mt-4 flex justify-end gap-2">
          <button type="button" onClick={onClose} className="btn btn-secondary h-9 px-3 text-sm">
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void onSubmit()}
            className="btn btn-primary h-9 px-3 text-sm"
            disabled={!draft.title.trim()}
          >
            <CheckCircle2 className="mr-1 h-4 w-4" />
            Create
          </button>
        </div>
        <p className="mt-3 inline-flex items-center gap-1 text-[11px] text-slate-500">
          <Clock className="h-3 w-3" /> Bundle starts in <span className="font-semibold">draft</span>; explicitly stage to publish to endpoints.
        </p>
      </div>
    </div>
  )
}
