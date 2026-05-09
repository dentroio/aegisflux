'use client'

import { Suspense, useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import { EvidenceGraphPanel } from '@/components/EvidenceGraphPanel'

function EvidenceRouteInner() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [gate, setGate] = useState(false)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/?next=/analyze/evidence')
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

  const findingId = (searchParams.get('finding_id') || '').trim()
  const deviceId = (searchParams.get('device_id') || '').trim()
  const processGUID = (searchParams.get('process_guid') || '').trim()

  const health = { label: 'Observe-only', tone: 'amber' as HealthTone, text: 'Read-only investigation' }

  return (
    <ConsoleShell
      activeNavId="evidence"
      breadcrumbs={[{ label: 'Analyze', href: '/analyze/evidence' }, { label: 'Evidence path' }]}
      health={health}
      onLogout={onLogout}
    >
      <main className="mx-auto max-w-[1100px] px-5 py-6">
        <EvidenceGraphPanel
          initialFindingId={findingId}
          initialDeviceId={deviceId}
          initialProcessGUID={processGUID}
          autoLoad={Boolean(findingId || deviceId || processGUID)}
        />
      </main>
    </ConsoleShell>
  )
}

export default function EvidenceRoute() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center bg-slate-50 text-sm text-slate-600">
          Loading evidence path…
        </div>
      }
    >
      <EvidenceRouteInner />
    </Suspense>
  )
}
