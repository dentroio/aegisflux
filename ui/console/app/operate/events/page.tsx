'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/lib/labAuth'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { useRouter } from 'next/navigation'

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
  const visible = needle
    ? events.filter((ev) => JSON.stringify(ev).toLowerCase().includes(needle))
    : events

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
        <input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="mb-4 w-full max-w-md rounded border border-gray-300 px-3 py-2 text-sm"
          placeholder="Filter type, subject, device…"
        />
        {err && <div className="mb-4 rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">{err}</div>}
        {visible.length === 0 ? (
          <div className="rounded-xl border border-dashed border-gray-300 bg-white p-10 text-center text-sm text-gray-600">
            {events.length === 0 ? 'No operational events recorded yet.' : 'No events match this filter.'}
          </div>
        ) : (
          <ul className="space-y-2">
            {visible.map((ev) => (
              <li key={ev.id} className="rounded-lg border border-gray-200 bg-white px-4 py-3 text-sm shadow-sm">
                <div className="flex flex-wrap items-center gap-2 text-xs text-gray-500">
                  <span className="font-mono">{new Date(ev.created_at_ms).toLocaleString()}</span>
                  <span className="rounded bg-slate-100 px-2 py-0.5 font-semibold text-slate-800">{ev.event_type}</span>
                  {ev.status ? <span className="rounded bg-blue-50 px-2 py-0.5 text-blue-900">{ev.status}</span> : null}
                </div>
                <p className="mt-2 text-gray-900">{ev.description}</p>
                {(ev.subject || ev.device_id) && (
                  <p className="mt-1 text-xs text-gray-500">
                    {ev.subject && <span className="mr-3">Subject: {ev.subject}</span>}
                    {ev.device_id && <span>Device: {ev.device_id}</span>}
                  </p>
                )}
              </li>
            ))}
          </ul>
        )}
      </main>
    </ConsoleShell>
  )
}
