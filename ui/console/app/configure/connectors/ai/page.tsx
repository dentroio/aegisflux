'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import {
  ArrowLeft,
  Bot,
  Edit,
  MessageSquare,
  Plus,
  RefreshCw,
  Send,
  Shield,
  Sparkles,
  Star,
  Wifi,
} from 'lucide-react'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import {
  EvidenceBoundConclusionPanel,
  type EvidenceBoundConclusion,
} from '@/components/EvidenceBoundConclusionPanel'
import { fetchHarnessAvailable } from '@/shared/agentAnalyst'

type Provider = {
  id: string
  kind: string
  name: string
  enabled: boolean
  secret_configured: boolean
  last_health_ok: boolean
  last_health_message?: string
}

type HarnessToolCallBrief = {
  call_id?: string
  tool_id: string
  duration_ms: number
  error?: string
}

type HarnessRunRow = {
  run_id: string
  job_id: string
  agent_id: string
  device_id: string
  finding_id?: string
  status: string
  provider_kind: string
  model: string
  started_at_ms: number
  duration_ms: number
  tool_calls?: HarnessToolCallBrief[]
}

type HarnessToolCallDetail = {
  call_id: string
  tool_id: string
  duration_ms: number
  error?: string
  input_json?: unknown
  output_json?: unknown
}

type HarnessRunDetail = Omit<HarnessRunRow, 'tool_calls'> & {
  tool_calls: HarnessToolCallDetail[]
  prompt_redacted_preview?: string
  error?: string
  assessment?: string
  evidence_summary?: string
  confidence?: string
  recommended_next_action?: string
  evidence_bound_conclusion?: EvidenceBoundConclusion
  evidence_bound_validation_errors?: string[]
}

function providerLabel(kind: string) {
  switch (kind) {
    case 'openai':
      return 'OpenAI'
    case 'local':
      return 'Local (Ollama)'
    case 'anthropic':
      return 'Anthropic'
    case 'google':
      return 'Google'
    default:
      return kind
  }
}

function iconButtonClass(extra = '') {
  return `inline-flex h-11 w-11 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-500 shadow-sm transition-all hover:-translate-y-0.5 hover:bg-slate-50 hover:text-blue-600 ${extra}`
}

export default function AIProvidersPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)
  const [providers, setProviders] = useState<Provider[]>([])
  const [defaultId, setDefaultId] = useState('')
  const [privacy, setPrivacy] = useState<{ allow_external_ai?: boolean } | null>(null)
  const [harnessRuns, setHarnessRuns] = useState<HarnessRunRow[]>([])
  const [runDetail, setRunDetail] = useState<HarnessRunDetail | null>(null)
  const [runDetailLoading, setRunDetailLoading] = useState(false)
  const [harnessAvailable, setHarnessAvailable] = useState<boolean | null>(null)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/')
      return
    }
    setGate(true)
  }, [router])

  async function load() {
    const [p, pr, runsRes] = await Promise.all([
      fetch('/api/actions/platform/ai/providers', { cache: 'no-store' }).then((r) => (r.ok ? r.json() : null)),
      fetch('/api/actions/platform/ai/privacy', { cache: 'no-store' }).then((r) => (r.ok ? r.json() : null)),
      fetch('/api/actions/platform/ai/agent-harness/runs', { cache: 'no-store' }).then((r) => (r.ok ? r.json() : null)),
    ])
    if (p) {
      setProviders(p.providers || [])
      setDefaultId(p.default_provider_id || '')
    }
    if (pr) setPrivacy(pr)
    if (runsRes?.runs) setHarnessRuns(runsRes.runs as HarnessRunRow[])
    else setHarnessRuns([])
    setHarnessAvailable(await fetchHarnessAvailable())
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

  async function openRunDetail(runId: string) {
    setRunDetail(null)
    setRunDetailLoading(true)
    try {
      const res = await fetch(`/api/actions/platform/ai/agent-harness/runs/${encodeURIComponent(runId)}`, {
        cache: 'no-store',
      })
      if (res.ok) {
        setRunDetail((await res.json()) as HarnessRunDetail)
      }
    } finally {
      setRunDetailLoading(false)
    }
  }

  function onLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  const health = { label: 'AI', tone: 'slate' as HealthTone, text: 'Provider lab' }
  const defaultProvider = providers.find((provider) => provider.id === defaultId)

  if (!gate) return null

  return (
    <ConsoleShell activeNavId="connectors" breadcrumbs={[{ label: 'Connectors', href: '/configure/connectors' }, { label: 'AI Providers' }]} health={health} onLogout={onLogout}>
      <main className="space-y-6 px-8 py-6">
        <Link href="/configure/connectors" className="inline-flex items-center gap-2 text-sm font-medium text-slate-500 hover:text-slate-800">
          <ArrowLeft className="h-4 w-4" />
          Back to Connectors
        </Link>

        <section className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight text-slate-950">
                <Sparkles className="h-5 w-5 text-purple-500" />
                AI / LLM Providers
              </h1>
              <p className="mt-2 text-base text-slate-500">
                Configure one or more AI providers. One is designated as the default for all agents.
              </p>
            </div>
            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-xl bg-gradient-to-r from-sky-600 to-blue-600 px-5 py-3 text-sm font-semibold text-white shadow-[0_10px_30px_-12px_rgba(37,99,235,0.65)] transition-all hover:-translate-y-0.5"
              onClick={async () => {
                const kind = window.prompt('Provider kind (openai, local, anthropic, google)', 'openai')
                if (!kind) return
                const sec = window.prompt(`Optional API secret for ${providerLabel(kind)} (never shown again)`, '')
                if (sec === null) return
                await fetch('/api/actions/platform/ai/providers', {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({ kind, secret: sec || undefined }),
                })
                await load()
              }}
            >
              <Plus className="h-4 w-4" />
              Add Provider
            </button>
          </div>

          {defaultProvider ? (
            <div className="mt-5 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800">
              <div className="flex flex-wrap items-center gap-2">
                <Star className="h-4 w-4 text-blue-600" />
                <span className="font-semibold">Default: {defaultProvider.name}</span>
                <span>· {providerLabel(defaultProvider.kind)}</span>
                <span>· {defaultProvider.last_health_ok ? 'Connected' : 'Health unknown'}</span>
              </div>
            </div>
          ) : null}

          <div className="mt-5 space-y-3">
            {providers.length === 0 ? (
              <div className="rounded-lg border-2 border-dashed border-gray-200 px-6 py-10 text-center">
                <Sparkles className="mx-auto h-10 w-10 text-slate-300" />
                <p className="mt-3 text-sm font-semibold text-slate-700">No AI providers configured</p>
                <p className="mt-1 text-sm text-slate-500">Add a provider to enable AI-assisted device classification.</p>
              </div>
            ) : (
              providers.map((p) => {
                const isDefault = defaultId === p.id
                return (
                  <div
                    key={p.id}
                    className={`rounded-lg border p-5 transition-colors ${
                      isDefault ? 'border-blue-300 bg-blue-50' : 'border-gray-200 bg-white'
                    }`}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-4">
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <h2 className="text-lg font-bold text-slate-900">{p.name}</h2>
                          {isDefault ? (
                            <span className="inline-flex items-center gap-1 rounded-full bg-blue-600 px-2.5 py-1 text-xs font-semibold text-white">
                              <Star className="h-3.5 w-3.5" />
                              Default
                            </span>
                          ) : null}
                          <span className="rounded bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600">
                            {providerLabel(p.kind)}
                          </span>
                          <span className="inline-flex items-center gap-1.5 text-sm text-slate-500">
                            <span className={`h-2.5 w-2.5 rounded-full ${p.last_health_ok ? 'bg-emerald-500' : 'bg-slate-300'}`} />
                            {p.last_health_ok ? 'Connected' : 'Unknown'}
                          </span>
                        </div>
                        <p className="mt-2 text-sm text-slate-500">
                          Secret stored: {p.secret_configured ? 'yes (server-only)' : 'no'}
                          {p.last_health_message ? ` · ${p.last_health_message}` : ''}
                        </p>
                      </div>
                      <div className="flex items-center gap-2">
                        <button type="button" className={iconButtonClass()} onClick={() => testProvider(p.id)} title="Test connection">
                          <Wifi className="h-4 w-4" />
                        </button>
                        {!isDefault ? (
                          <button type="button" className={iconButtonClass()} onClick={() => setDefault(p.id)} title="Set as default">
                            <Star className="h-4 w-4" />
                          </button>
                        ) : null}
                        <button
                          type="button"
                          className={iconButtonClass()}
                          title="Store secret"
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
                          <Edit className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                )
              })
            )}
          </div>
        </section>

        <section className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="flex items-center gap-2 text-lg font-bold text-slate-900">
            <MessageSquare className="h-4 w-4 text-purple-500" />
            Test Chat
          </h2>
          <p className="mt-2 text-sm text-slate-500">
            Send a test query using <span className="font-semibold">{defaultProvider?.name || 'the default provider'}</span>.
          </p>
          <div className="mt-5 flex gap-2">
            <input
              type="text"
              disabled
              placeholder="Ask a question..."
              className="h-11 flex-1 rounded-lg border border-gray-200 bg-white px-4 text-sm text-slate-500 outline-none"
            />
            <button
              type="button"
              disabled
              className="inline-flex items-center gap-2 rounded-xl bg-gradient-to-r from-violet-500 to-fuchsia-500 px-5 py-3 text-sm font-semibold text-white opacity-70"
            >
              <Send className="h-4 w-4" />
              Send
            </button>
          </div>
        </section>

        <section className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h2 className="flex items-center gap-2 text-lg font-bold text-slate-900">
                <Bot className="h-4 w-4 text-purple-500" />
                Registered Agents
              </h2>
              <p className="mt-2 text-sm text-slate-500">
                AI agents built into AegisFlux that use the configured provider. Read-only — agents are defined by the system.
              </p>
            </div>
            <Link href="/agents" className="text-sm font-semibold text-blue-600 hover:text-blue-800">
              View agents →
            </Link>
          </div>
        </section>

        <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
          <Link
            href="/configure/settings"
            className="flex w-full items-center justify-between px-6 py-5 text-left"
          >
            <div className="flex items-center gap-3">
              <Shield className="h-4 w-4 text-blue-600" />
              <div>
                <h2 className="text-lg font-bold text-slate-900">External LLM Privacy Controls</h2>
                {privacy ? (
                  <p className="mt-1 text-sm text-slate-500">
                    External AI requests are {privacy.allow_external_ai ? 'enabled' : 'limited to local paths'}.
                  </p>
                ) : null}
              </div>
            </div>
            <span className="text-xl text-slate-400">›</span>
          </Link>
        </section>

        <section className="mt-10 rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          {harnessAvailable === false ? (
            <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950">
              Agent harness routes are not available on the Actions API at port 8083. Rebuild and restart{' '}
              <code className="rounded bg-amber-100 px-1 font-mono text-xs">actions-api</code> from current source, or run{' '}
              <code className="rounded bg-amber-100 px-1 font-mono text-xs">docker compose build actions-api &amp;&amp; docker compose up -d actions-api</code>.
            </div>
          ) : null}
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Agent harness runs</h2>
              <p className="mt-1 text-sm text-slate-600">
                Governed jobs with typed read-only tools and evidence-bound conclusions (WO-AGENTS-001/002). Use “Explain AI activity” on a
                device to create runs.
              </p>
            </div>
            <button type="button" className={iconButtonClass('h-9 w-9')} onClick={() => void load()} title="Refresh">
              <RefreshCw className="h-4 w-4" />
            </button>
          </div>
          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-left text-sm text-slate-800">
              <thead className="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="py-2 pr-3">Started</th>
                  <th className="py-2 pr-3">Agent</th>
                  <th className="py-2 pr-3">Device</th>
                  <th className="py-2 pr-3">Status</th>
                  <th className="py-2 pr-3">Tools</th>
                  <th className="py-2 pr-3">Duration</th>
                  <th className="py-2"> </th>
                </tr>
              </thead>
              <tbody>
                {harnessRuns.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="py-6 text-center text-slate-500">
                      No harness runs yet.
                    </td>
                  </tr>
                ) : (
                  harnessRuns.slice(0, 40).map((row) => (
                    <tr key={row.run_id} className="border-b border-slate-100">
                      <td className="py-2 pr-3 whitespace-nowrap text-xs text-slate-600">
                        {new Date(row.started_at_ms).toLocaleString()}
                      </td>
                      <td className="py-2 pr-3 font-mono text-xs">{row.agent_id}</td>
                      <td className="py-2 pr-3 font-mono text-xs">{row.device_id}</td>
                      <td className="py-2 pr-3">{row.status}</td>
                      <td className="py-2 pr-3 text-xs text-slate-600">{row.tool_calls?.length ?? 0}</td>
                      <td className="py-2 pr-3 text-xs">{row.duration_ms} ms</td>
                      <td className="py-2">
                        <button
                          type="button"
                          className="text-xs font-semibold text-blue-700 hover:underline"
                          onClick={() => void openRunDetail(row.run_id)}
                        >
                          Detail
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
          {runDetailLoading ? <p className="mt-3 text-sm text-slate-500">Loading run…</p> : null}
          {runDetail ? (
            <div className="mt-6 rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm">
              <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                <span className="font-semibold text-slate-900">Run {runDetail.run_id}</span>
                <button type="button" className="text-xs text-slate-600 hover:text-slate-900" onClick={() => setRunDetail(null)}>
                  Close
                </button>
              </div>
              <dl className="grid gap-2 text-xs sm:grid-cols-2">
                <div>
                  <dt className="text-slate-500">Model</dt>
                  <dd className="font-mono">{runDetail.model}</dd>
                </div>
                <div>
                  <dt className="text-slate-500">Provider</dt>
                  <dd className="font-mono">{runDetail.provider_kind}</dd>
                </div>
                <div>
                  <dt className="text-slate-500">Status</dt>
                  <dd>{runDetail.status}</dd>
                </div>
              </dl>
              {runDetail.evidence_bound_conclusion ? (
                <div className="mt-4 rounded-lg border border-slate-200 bg-white p-4">
                  <EvidenceBoundConclusionPanel
                    conclusion={runDetail.evidence_bound_conclusion}
                    validationErrors={runDetail.evidence_bound_validation_errors}
                  />
                </div>
              ) : runDetail.assessment ? (
                <div className="mt-4 rounded-lg border border-slate-200 bg-white p-3 text-xs text-slate-700">
                  <div className="font-medium text-slate-500">Legacy assessment</div>
                  <p className="mt-1">{runDetail.assessment}</p>
                </div>
              ) : null}
              {runDetail.prompt_redacted_preview ? (
                <div className="mt-3">
                  <div className="text-xs font-medium text-slate-500">Redacted prompt preview</div>
                  <pre className="mt-1 max-h-40 overflow-auto rounded border border-slate-200 bg-white p-2 text-xs text-slate-700">
                    {runDetail.prompt_redacted_preview}
                  </pre>
                </div>
              ) : null}
              <div className="mt-4">
                <div className="text-xs font-medium text-slate-500">Tool calls</div>
                <ul className="mt-2 space-y-2">
                  {runDetail.tool_calls?.map((tc) => (
                    <li key={tc.call_id} className="rounded border border-slate-200 bg-white p-2 text-xs">
                      <div className="font-mono font-semibold text-slate-900">{tc.tool_id}</div>
                      <div className="text-slate-600">
                        {tc.duration_ms} ms{tc.error ? ` · error: ${tc.error}` : ''}
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          ) : null}
        </section>
      </main>
    </ConsoleShell>
  )
}
