'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import { FirstValueTour } from '@/components/FirstValueTour'

export default function FirstValueDemoPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/?next=/demo')
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

  const health = { label: 'Lab', tone: 'amber' as HealthTone, text: 'Five-minute first-value tour' }

  return (
    <ConsoleShell
      activeNavId="dashboard"
      breadcrumbs={[{ label: 'First-value tour' }]}
      health={health}
      onLogout={onLogout}
    >
      <main className="mx-auto max-w-4xl px-5 py-8">
        <FirstValueTour />
      </main>
    </ConsoleShell>
  )
}
