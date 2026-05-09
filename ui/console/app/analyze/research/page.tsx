'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import { ResearchFeedPanel } from '@/components/ResearchFeedPanel'

export default function ResearchFeedPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/?next=/analyze/research')
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

  const health = { label: 'Observe-only', tone: 'amber' as HealthTone, text: 'Research feed — governed lifecycle' }

  return (
    <ConsoleShell
      activeNavId="research"
      breadcrumbs={[{ label: 'Analyze', href: '/analyze/research' }, { label: 'AI Research Feed' }]}
      health={health}
      onLogout={onLogout}
    >
      <main className="mx-auto max-w-[1500px] px-5 py-6">
        <ResearchFeedPanel />
      </main>
    </ConsoleShell>
  )
}
