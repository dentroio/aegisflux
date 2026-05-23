'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Bot, CheckCircle2 } from 'lucide-react'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'

type ConnectorCard = {
  title: string
  description: string
  unlocks: string
  note: string
  href: string
  status?: 'connected' | 'planned'
}

const externalConnectors: ConnectorCard[] = [
  {
    title: 'AI / LLM Providers',
    description: 'Configure governed AI routes for local models, enterprise model gateways, and external LLM providers.',
    unlocks: 'Provider health checks, default routing, redaction controls',
    note: 'Required for AI-assisted investigation',
    href: '/configure/connectors/ai',
    status: 'connected',
  },
]

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
      <main className="px-8 py-6">
        <div className="mb-8">
          <h1 className="text-2xl font-bold tracking-tight text-slate-950">Connectors</h1>
          <p className="mt-3 text-base text-slate-500">Connect AegisFlux to external systems used by this deployment.</p>
          <div className="mt-3 flex flex-wrap items-center gap-3">
            <span className="inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-sm font-medium text-emerald-700">
              Live connector
            </span>
            <span className="text-sm text-slate-500">Last updated just now</span>
          </div>
        </div>

        <section>
          <h2 className="mb-4 flex items-center gap-2 text-xl font-bold text-slate-900">
            <Bot className="h-5 w-5 text-slate-500" />
            AI
          </h2>
          <div className="grid gap-5 lg:grid-cols-2 xl:grid-cols-3">
            {externalConnectors.map((card) => (
              <Link
                key={card.title}
                href={card.href}
                className="group block min-h-[206px] rounded-lg border border-gray-200 bg-white p-6 shadow-sm transition-all hover:-translate-y-0.5 hover:border-blue-200 hover:shadow-md"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-center gap-2">
                    <Bot className="h-4 w-4 text-blue-600" />
                    <h3 className="text-lg font-bold text-slate-900">{card.title}</h3>
                  </div>
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-100 px-2.5 py-1 text-sm font-medium text-emerald-700">
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    Connected
                  </span>
                </div>
                <p className="mt-4 text-sm leading-6 text-slate-500">{card.description}</p>
                <p className="mt-4 text-sm leading-5 text-slate-600">
                  <span className="font-semibold text-slate-700">Unlocks:</span> {card.unlocks}
                </p>
                <p className="mt-3 text-sm font-semibold text-slate-600">{card.note}</p>
                <p className="mt-5 text-sm font-semibold text-clarion-blue group-hover:text-blue-700">Open →</p>
              </Link>
            ))}
          </div>
        </section>
      </main>
    </ConsoleShell>
  )
}
