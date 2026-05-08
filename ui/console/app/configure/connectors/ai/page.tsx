'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'

type Provider = {
  id: string
  kind: string
  name: string
  enabled: boolean
  secret_configured: boolean
  last_health_ok: boolean
  last_health_message?: string
}

export default function AIProvidersPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)
  const [providers, setProviders] = useState<Provider[]>([])
  const [defaultId, setDefaultId] = useState('')
  const [privacy, setPrivacy] = useState<{ allow_external_ai?: boolean } | null>(null)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/')
      return
    }
    setGate(true)
  }, [router])

  async function load() {
    const [p, pr] = await Promise.all([
      fetch('/api/actions/platform/ai/providers', { cache: 'no-store' }).then((r) => (r.ok ? r.json() : null)),
      fetch('/api/actions/platform/ai/privacy', { cache: 'no-store' }).then((r) => (r.ok ? r.json() : null)),
    ])
    if (p) {
      setProviders(p.providers || [])
      setDefaultId(p.default_provider_id || '')
    }
    if (pr) setPrivacy(pr)
  }

  useEffect(() => {
    if (!gate) return undefined
    load()
    return undefined
  }, [gate])

  async function setDefault(pid: string) {
    await fetch('/api/actions/platform/ai/providers/configure', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ default_provider_id: pid }),
    })
    await load()
  }

  async function testProvider(pid: string) {
    await fetch(`/api/actions/platform/ai/providers/${encodeURIComponent(pid)}/test`, { method: 'POST' })
    await load()
  }

  async function togglePrivacy(flag: boolean) {
    await fetch('/api/actions/platform/ai/privacy', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        ...(privacy || {}),
        allow_external_ai: flag,
      }),
    })
    await load()
  }

  function onLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  const health = { label: 'AI', tone: 'slate' as HealthTone, text: 'Provider lab' }

  if (!gate) return null

  return (
    <ConsoleShell activeNavId="connectors" breadcrumbs={[{ label: 'Connectors', href: '/configure/connectors' }, { label: 'AI Providers' }]} health={health} onLogout={onLogout}>
      <main className="mx-auto max-w-3xl px-4 py-8">
        <div className="mb-6 flex justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold text-slate-900">AI Providers</h1>
            <p className="text-sm text-slate-600">Governed connectivity for local/offline analysis and enterprise gateways.</p>
          </div>
          <Link href="/configure/settings" className="text-sm font-semibold text-blue-700">
            Privacy settings
          </Link>
        </div>
        {privacy && (
          <div className="mb-6 rounded-lg border border-gray-200 bg-white p-4 text-sm shadow-sm">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <span className="font-medium text-gray-900">External AI requests</span>
              <label className="flex items-center gap-2 text-gray-700">
                <input type="checkbox" checked={Boolean(privacy.allow_external_ai)} onChange={(e) => togglePrivacy(e.target.checked)} />
                Allow (still redacts outbound context)
              </label>
            </div>
          </div>
        )}
        <ul className="space-y-3">
          {providers.map((p) => (
            <li key={p.id} className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="font-semibold text-gray-900">{p.name}</div>
                  <div className="text-xs text-gray-500">
                    {p.kind} · secret stored: {p.secret_configured ? 'yes (server-only)' : 'no'} · health:{' '}
                    {p.last_health_ok ? 'ok' : 'unknown'}
                  </div>
                  {p.last_health_message && <div className="mt-1 text-xs text-gray-600">{p.last_health_message}</div>}
                </div>
                <div className="flex flex-wrap gap-2">
                  <button type="button" className="btn btn-secondary h-8 px-2 text-xs" onClick={() => setDefault(p.id)}>
                    {defaultId === p.id ? 'Default ✓' : 'Make default'}
                  </button>
                  <button type="button" className="btn btn-secondary h-8 px-2 text-xs" onClick={() => testProvider(p.id)}>
                    Test
                  </button>
                  <button
                    type="button"
                    className="btn btn-secondary h-8 px-2 text-xs"
                    onClick={async () => {
                      const sec = window.prompt(`Optional API secret for ${p.name} (never shown again)`, '')
                      if (sec === null) return
                      await fetch('/api/actions/platform/ai/providers', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ kind: p.kind, secret: sec || undefined }),
                      })
                      await load()
                    }}
                  >
                    Store secret…
                  </button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      </main>
    </ConsoleShell>
  )
}
