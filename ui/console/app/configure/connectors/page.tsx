'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'

export default function ConnectorsHubPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/')
      return
    }
    setGate(true)
  }, [router])

  function onLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  const health = { label: 'Configure', tone: 'slate' as HealthTone, text: 'Connectors' }

  if (!gate) {
    return null
  }

  return (
    <ConsoleShell activeNavId="connectors" breadcrumbs={[{ label: 'Connectors' }]} health={health} onLogout={onLogout}>
      <main className="mx-auto max-w-2xl px-4 py-8">
        <h1 className="text-xl font-semibold text-slate-900">Connectors</h1>
        <p className="mt-2 text-sm text-slate-600">Integrate telemetry transports and governed AI routes.</p>
        <ul className="mt-6 space-y-3 text-sm">
          <li>
            <Link className="font-semibold text-blue-700" href="/configure/connectors/ai">
              AI Providers
            </Link>
            <span className="text-slate-600"> — routing, lab health checks, secrets never returned to the browser.</span>
          </li>
        </ul>
      </main>
    </ConsoleShell>
  )
}
