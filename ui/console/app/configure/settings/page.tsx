'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/lib/labAuth'

type Privacy = Record<string, boolean>

export default function SettingsPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)
  const [privacy, setPrivacy] = useState<Privacy | null>(null)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/')
      return
    }
    setGate(true)
  }, [router])

  useEffect(() => {
    if (!gate) return undefined
    fetch('/api/actions/platform/ai/privacy', { cache: 'no-store' })
      .then((r) => (r.ok ? r.json() : null))
      .then(setPrivacy)
    return undefined
  }, [gate])

  async function save(next: Privacy) {
    await fetch('/api/actions/platform/ai/privacy', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(next),
    })
    setPrivacy(next)
  }

  function onLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  const health = { label: 'Settings', tone: 'slate' as HealthTone, text: 'Lab privacy' }

  if (!gate || !privacy) return gate ? <div className="p-6 text-sm">Loading…</div> : null

  const keys: (keyof Privacy)[] = [
    'allow_external_ai',
    'redact_ips',
    'redact_macs',
    'redact_usernames',
    'redact_emails',
    'redact_hostnames',
    'redact_command_lines',
    'redact_file_paths',
    'redact_raw_secrets',
  ]

  return (
    <ConsoleShell activeNavId="settings" breadcrumbs={[{ label: 'Settings' }]} health={health} onLogout={onLogout}>
      <main className="mx-auto max-w-xl px-4 py-8">
        <h1 className="text-xl font-semibold text-slate-900">Privacy & audit</h1>
        <p className="mt-2 text-sm text-slate-600">
          Tune redaction defaults for outbound AI context. Matching records are audited on the Actions API (`/platform/ai/*`).
        </p>
        <Link href="/configure/connectors/ai" className="mt-4 inline-block text-sm font-semibold text-blue-700">
          ← AI Providers
        </Link>
        <div className="mt-8 space-y-3 rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          {keys.map((k) => (
            <label key={k as string} className="flex items-center justify-between gap-4 text-sm text-gray-800">
              <span className="font-mono text-xs">{k as string}</span>
              <input
                type="checkbox"
                checked={Boolean(privacy[k])}
                onChange={(e) => save({ ...privacy, [k]: e.target.checked })}
              />
            </label>
          ))}
        </div>
      </main>
    </ConsoleShell>
  )
}
