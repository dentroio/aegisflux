'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { useRouter } from 'next/navigation'
import { BoundedTable, DetailModal, EmptyState, FilterBar, KpiTile, SummaryStrip } from '@/components/workbench/primitives'
import { formatHash, formatRelativeAge } from '@/shared/formatting'

type OpEvent = {
  id: string
  created_at_ms: number
  event_type: string
  status?: string
  subject?: string
  device_id?: string
  agent_uid?: string
  description: string
}

export default function OperationalEventFeedPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)
  const [events, setEvents] = useState<OpEvent[]>([])
  const [filter, setFilter] = useState('')
  const [selectedStatus, setSelectedStatus] = useState<'all' | 'ok' | 'error' | 'other'>('all')
  const [detailModal, setDetailModal] = useState<{ title: string; payload: unknown } | null>(null)
  const [showRaw, setShowRaw] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/')
      return
    }
    setGate(true)
  }, [router])

  useEffect(() => {
    if (!gate) return undefined
    fetch('/api/actions/platform/operational-events?limit=500', { cache: 'no-store' })
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error(String(r.status)))))
      .then((j) => setEvents((j.events as OpEvent[]) || []))
      .catch((e) => setErr(e instanceof Error ? e.message : String(e)))
    return undefined
  }, [gate])

  function onLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  const health = { label: 'Operate', tone: 'slate' as HealthTone, text: 'Operational events' }
  const needle = filter.trim().toLowerCase()
  const visible = events.filter((ev) => {
    if (needle && !JSON.stringify(ev).toLowerCase().includes(needle)) return false
    if (selectedStatus === 'all') return true
    const st = (ev.status || '').toLowerCase()
    if (selectedStatus === 'ok') return st === 'ok' || st === 'success'
    if (selectedStatus === 'error') return st === 'error' || st === 'failed'
    return st !== 'ok' && st !== 'success' && st !== 'error' && st !== 'failed'
  })
  const okCount = events.filter((ev) => ['ok', 'success'].includes((ev.status || '').toLowerCase())).length
  const errCount = events.filter((ev) => ['error', 'failed'].includes((ev.status || '').toLowerCase())).length

  if (!gate) {
    return <div className="flex min-h-screen items-center justify-center text-sm text-gray-600">Loading…</div>
  }

  return (
    <ConsoleShell activeNavId="event-feed" breadcrumbs={[{ label: 'Operational Event Feed' }]} health={health} onLogout={onLogout}>
      <main className="mx-auto max-w-5xl px-4 py-6">
        <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="text-lg font-semibold text-slate-900">Operational Event Feed</h1>
            <p className="text-sm text-slate-600">Platform actions, rollout signals, and hooks for AI and integration exports.</p>
          </div>
          <Link href="/" className="text-sm font-semibold text-blue-700">
            Dashboard
          </Link>
        </div>
        <SummaryStrip>
          <KpiTile label="Total events" value={events.length} />
          <KpiTile label="Success" value={okCount} />
          <KpiTile label="Errors" value={errCount} />
          <KpiTile label="Visible" value={visible.length} />
        </SummaryStrip>
        <FilterBar>
          <input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="input h-9 min-w-[220px] max-w-md"
            placeholder="Filter type, subject, device..."
          />
          {([
            ['all', 'All'],
            ['ok', 'Success'],
            ['error', 'Errors'],
            ['other', 'Other'],
          ] as const).map(([id, label]) => (
            <button
              key={id}
              type="button"
              onClick={() => setSelectedStatus(id)}
              className={`rounded-full border px-3 py-1 text-xs font-semibold ${
                selectedStatus === id ? 'border-primary-600 bg-primary-600 text-white' : 'border-gray-200 bg-gray-50 text-gray-700'
              }`}
            >
              {label}
            </button>
          ))}
        </FilterBar>
        {err && <div className="mb-4 rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">{err}</div>}
        {visible.length === 0 ? (
          <EmptyState title="No events" message={events.length === 0 ? 'No operational events recorded yet.' : 'No events match this filter.'} />
        ) : (
          <BoundedTable
            headers={['Time', 'Type', 'Status', 'Description', 'Subject/Device', 'Detail']}
            rows={visible.map((ev) => ([
              <span key={`${ev.id}-time`} className="text-xs text-gray-600" title={new Date(ev.created_at_ms).toLocaleString()}>
                {formatRelativeAge(ev.created_at_ms)}
              </span>,
              <span key={`${ev.id}-type`} className="text-xs font-semibold text-slate-800">{ev.event_type}</span>,
              <span key={`${ev.id}-status`} className="text-xs text-gray-600">{ev.status || 'n/a'}</span>,
              <span key={`${ev.id}-desc`} className="text-sm text-gray-900">{ev.description}</span>,
              <span key={`${ev.id}-subject`} className="text-xs text-gray-600">
                {ev.subject ? `subject:${formatHash(ev.subject)}` : ''} {ev.device_id ? `device:${formatHash(ev.device_id)}` : ''}
              </span>,
              <button
                key={`${ev.id}-detail`}
                type="button"
                className="text-xs font-semibold text-primary-700"
                onClick={() => {
                  setShowRaw(false)
                  setDetailModal({ title: `${ev.event_type} (${ev.id})`, payload: ev })
                }}
              >
                View detail
              </button>,
            ]))}
          />
        )}
      </main>
      <DetailModal
        open={Boolean(detailModal)}
        title={detailModal?.title || 'Event detail'}
        detail={showRaw ? detailModal?.payload || {} : summarizeEvent(detailModal?.payload)}
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
    </ConsoleShell>
  )
}

function summarizeEvent(payload: unknown) {
  if (!payload || typeof payload !== 'object') return payload
  const ev = payload as OpEvent
  return {
    id: ev.id,
    created_at: new Date(ev.created_at_ms).toLocaleString(),
    event_type: ev.event_type,
    status: ev.status || 'n/a',
    description: ev.description,
    subject: ev.subject || 'n/a',
    device_id: ev.device_id || 'n/a',
    agent_uid: ev.agent_uid || 'n/a',
  }
}
