'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'

type ShellHeroProps = {
  title: string
  subtitle: string
  activeNavId: string
}

export function RoutedShellHero({ title, subtitle, activeNavId }: ShellHeroProps) {
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
    window.location.href = '/'
  }

  const health = { label: 'Console', tone: 'slate' as HealthTone, text: 'Lab shell navigation' }

  if (!gate) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-50 text-sm text-slate-600">
        Loading…
      </div>
    )
  }

  return (
    <ConsoleShell activeNavId={activeNavId} breadcrumbs={[{ label: title }]} health={health} onLogout={onLogout}>
      <main className="mx-auto max-w-[900px] px-5 py-8">
        <h1 className="text-xl font-semibold text-slate-900">{title}</h1>
        <p className="mt-2 text-sm leading-6 text-slate-600">{subtitle}</p>
        <Link href="/" className="mt-6 inline-block text-sm font-semibold text-blue-700 hover:text-blue-900">
          Back to dashboard
        </Link>
      </main>
    </ConsoleShell>
  )
}
