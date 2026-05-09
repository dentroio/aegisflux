'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import { AbomPanel } from '@/components/AbomPanel'

export default function AbomFleetPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/?next=/discover/abom')
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

  const health = { label: 'Observe-only', tone: 'amber' as HealthTone, text: 'Inventory only — no enforcement' }

  return (
    <ConsoleShell
      activeNavId="abom"
      breadcrumbs={[{ label: 'Discover', href: '/discover/abom' }, { label: 'Agent Bill of Materials' }]}
      health={health}
      onLogout={onLogout}
    >
      <main className="mx-auto max-w-[1500px] px-5 py-6">
        <AbomPanel scope="fleet" />
      </main>
    </ConsoleShell>
  )
}
