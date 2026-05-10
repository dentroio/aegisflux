'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import { AuditBundlesPanel } from '@/components/AuditBundlesPanel'

export default function AuditBundlesPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/?next=/control/audit-bundles')
      return
    }
    setGate(true)
  }, [router])

  function onLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  if (!gate) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-50 text-sm text-slate-600">
        Loading…
      </div>
    )
  }

  const health = { label: 'Audit-only', tone: 'amber' as HealthTone, text: 'No enforcement' }

  return (
    <ConsoleShell
      activeNavId="controls"
      breadcrumbs={[{ label: 'Controls', href: '/control/controls' }, { label: 'Audit bundles' }]}
      health={health}
      onLogout={onLogout}
    >
      <main className="mx-auto max-w-6xl px-4 py-6">
        <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          <strong>Audit-only.</strong> Audit-mode bundles never block, deny, or quarantine. Endpoints accept the
          bundle, evaluate it, and report observe-only match telemetry. Read more in the
          {' '}
          <a
            href="https://github.com/sgerhart/aegisflux/blob/main/docs/safety/AUDIT_MODE.md"
            target="_blank"
            rel="noreferrer"
            className="underline"
          >
            audit-mode safety doc
          </a>
          .
        </div>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <Link href="/control/controls" className="text-sm font-semibold text-blue-700">Back to controls</Link>
        </div>
        <AuditBundlesPanel />
      </main>
    </ConsoleShell>
  )
}
