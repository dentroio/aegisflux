'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import {
  ArrowRight,
  Bot,
  Chrome,
  Cpu,
  Network,
  ShieldCheck,
  Sparkles,
  Wrench,
} from 'lucide-react'

type Scenario = {
  id: string
  title: string
  oneLiner: string
  pillar: 'discover' | 'investigate' | 'design' | 'adapt'
  steps: string[]
  expected: string[]
  routes: { label: string; href: string }[]
  icon: typeof Sparkles
}

const SCENARIOS: Scenario[] = [
  {
    id: 'browser-ai-extension',
    title: 'Browser AI extension',
    oneLiner: 'A managed Chromium endpoint installs a popular AI assistant extension that exfiltrates clipboard content to a third-party model.',
    pillar: 'discover',
    icon: Chrome,
    steps: [
      'Confirm the agent has reported recent extension inventory.',
      'Open Discover → Agent Bill of Materials and filter by category browser_ai_extension.',
      'Pick the highlighted extension and review its findings, last seen, and devices list.',
      'Click "Design control" on a finding to draft an observe-only block of the extension origin.',
    ],
    expected: [
      'ABOM lists at least one browser_ai_extension item with multiple devices.',
      'A finding shows clipboard read paired with outbound DNS to an AI-model domain.',
      'A draft control records the finding id, scope, and observe-only flag.',
    ],
    routes: [
      { label: 'Agent Bill of Materials', href: '/discover/abom' },
      { label: 'Evidence path', href: '/analyze/evidence' },
      { label: 'Finding-to-control', href: '/control/controls' },
    ],
  },
  {
    id: 'coding-agent',
    title: 'Coding agent (CLI)',
    oneLiner: 'A developer machine runs a coding agent that reads source code and uploads a project tree to an external completion API.',
    pillar: 'investigate',
    icon: Bot,
    steps: [
      'Open Discover → Activity to see recent process executions on the developer host.',
      'Pivot into Analyze → Evidence path for the coding-agent process.',
      'Review the parent process, file reads, DNS lookups, and the model gateway destination.',
      'Promote the suggested observe-only detection from the research feed if not already linked.',
    ],
    expected: [
      'Evidence path shows a clear chain from shell → coding agent → outbound API.',
      'Operator label and confidence reason explain why the chain is suspicious.',
      'Research feed item is linked to a detection candidate in the workflow board.',
    ],
    routes: [
      { label: 'Activity', href: '/discover/activity' },
      { label: 'Evidence path', href: '/analyze/evidence' },
      { label: 'AI research feed', href: '/analyze/research' },
    ],
  },
  {
    id: 'local-model-runtime',
    title: 'Local model runtime',
    oneLiner: 'A local model runtime (e.g., Ollama-style) listens on localhost and is reachable from any process on the host.',
    pillar: 'design',
    icon: Cpu,
    steps: [
      'Open Discover → ABOM and filter by local_model_runtime.',
      'Inspect the listening sockets, devices, and confidence score.',
      'Open Finding-to-control and draft an observe-only egress control around the runtime port.',
      'Run the lab simulation to see projected matches before any rollout.',
    ],
    expected: [
      'ABOM item lists the runtime with at least one listening port and bind address.',
      'Draft control captures scope (port, process, host) and a rollback note.',
      'Latest simulation shows match count, top processes, and destinations.',
    ],
    routes: [
      { label: 'Agent Bill of Materials', href: '/discover/abom' },
      { label: 'Finding-to-control', href: '/control/controls' },
    ],
  },
  {
    id: 'mcp-endpoint',
    title: 'MCP endpoint exposure',
    oneLiner: 'A workstation exposes an MCP endpoint that allows an external agent to invoke arbitrary tools on the host.',
    pillar: 'investigate',
    icon: Wrench,
    steps: [
      'Open ABOM and filter by mcp_endpoint.',
      'Open the device detail and review readiness and connectivity.',
      'Open Evidence path to view recent invocations of the MCP endpoint.',
      'Promote the related research item and review the candidate quality gate.',
    ],
    expected: [
      'ABOM lists the MCP endpoint with confidence and devices.',
      'Agent readiness explains heartbeat, event ingestion, and connectivity health.',
      'Detection candidate appears on the workflow board in the simulated stage.',
    ],
    routes: [
      { label: 'Agent Bill of Materials', href: '/discover/abom' },
      { label: 'Agents', href: '/agents' },
      { label: 'AI research feed', href: '/analyze/research' },
    ],
  },
  {
    id: 'suspicious-automation',
    title: 'Suspicious automation finding',
    oneLiner: 'An RPA-style automation chains a browser AI extension and a CLI agent to read clipboard, upload, and write files.',
    pillar: 'adapt',
    icon: Network,
    steps: [
      'Open Analyze → Findings to locate the high-risk automation finding.',
      'Pivot into the evidence path and review the missing-evidence callouts.',
      'Open the AI research feed for related intelligence; promote if scoped.',
      'Watch the candidate move from new → simulated → reviewed and check the gate.',
    ],
    expected: [
      'Finding ties to multiple ABOM items and a single evidence narrative.',
      'Research feed item shows a linked candidate id.',
      'Workflow board shows the candidate progressing with quality-gate status.',
    ],
    routes: [
      { label: 'Findings', href: '/analyze/findings' },
      { label: 'Evidence path', href: '/analyze/evidence' },
      { label: 'AI research feed', href: '/analyze/research' },
    ],
  },
]

const PILLAR_TONE: Record<Scenario['pillar'], string> = {
  discover: 'border-cyan-200 bg-cyan-50 text-cyan-900',
  investigate: 'border-blue-200 bg-blue-50 text-blue-900',
  design: 'border-emerald-200 bg-emerald-50 text-emerald-900',
  adapt: 'border-amber-200 bg-amber-50 text-amber-900',
}

export default function SampleScenariosPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/?next=/demo/scenarios')
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

  const health = { label: 'Lab', tone: 'amber' as HealthTone, text: 'Sample scenarios for first-value demo' }

  return (
    <ConsoleShell
      activeNavId="dashboard"
      breadcrumbs={[{ label: 'First-value tour', href: '/demo' }, { label: 'Sample scenarios' }]}
      health={health}
      onLogout={onLogout}
    >
      <main className="mx-auto max-w-5xl px-5 py-8">
        <header className="mb-6 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h1 className="text-2xl font-semibold text-slate-900">Sample scenarios</h1>
          <p className="mt-2 max-w-3xl text-sm text-slate-600">
            Five short scenarios show how AegisFlux turns AI-on-the-endpoint signal into governed, observe-only control.
            Each scenario lists the steps, expected observations, and the product routes that demonstrate them.
            Detections and controls in the lab are observe-only; nothing is enforced on agents.
          </p>
          <div className="mt-3 flex flex-wrap gap-2 text-xs">
            <Link
              href="/demo"
              className="inline-flex items-center gap-1 rounded-md border border-blue-200 bg-blue-50 px-2 py-1 font-semibold text-blue-800 hover:bg-blue-100"
            >
              <ShieldCheck className="h-3.5 w-3.5" />
              First-value tour
            </Link>
            <a
              href="https://github.com/sgerhart/aegisflux/blob/main/docs/demos/SAMPLE_SCENARIOS.md"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-2 py-1 font-semibold text-slate-700 hover:bg-slate-50"
            >
              Operator narrative (docs)
            </a>
            <a
              href="https://github.com/sgerhart/aegisflux/blob/main/docs/demos/CHECKLIST.md"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-2 py-1 font-semibold text-slate-700 hover:bg-slate-50"
            >
              Demo checklist (docs)
            </a>
          </div>
        </header>

        <ol className="space-y-4">
          {SCENARIOS.map((scenario, idx) => {
            const Icon = scenario.icon
            return (
              <li
                key={scenario.id}
                className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
              >
                <div className="flex flex-wrap items-start gap-3">
                  <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${PILLAR_TONE[scenario.pillar]}`}>
                    <Icon className="h-5 w-5" />
                  </div>
                  <div className="flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                        Scenario {idx + 1} · {scenario.pillar}
                      </span>
                    </div>
                    <h2 className="mt-1 text-lg font-semibold text-slate-900">{scenario.title}</h2>
                    <p className="text-sm text-slate-700">{scenario.oneLiner}</p>
                  </div>
                </div>
                <div className="mt-4 grid gap-4 md:grid-cols-2">
                  <Section title="Steps">
                    <ol className="grid gap-1 pl-5 text-sm text-slate-700 list-decimal">
                      {scenario.steps.map((step) => (
                        <li key={step}>{step}</li>
                      ))}
                    </ol>
                  </Section>
                  <Section title="Expected observations">
                    <ul className="grid gap-1 pl-5 text-sm text-slate-700 list-disc">
                      {scenario.expected.map((item) => (
                        <li key={item}>{item}</li>
                      ))}
                    </ul>
                  </Section>
                </div>
                <div className="mt-4 flex flex-wrap gap-2">
                  {scenario.routes.map((route) => (
                    <Link
                      key={route.href}
                      href={route.href}
                      className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-3 py-1 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                    >
                      {route.label}
                      <ArrowRight className="h-3.5 w-3.5" />
                    </Link>
                  ))}
                </div>
              </li>
            )
          })}
        </ol>

        <p className="mt-6 text-xs text-slate-500">
          All scenarios run against lab/demo data and remain observe-only. They do not change endpoint enforcement state.
        </p>
      </main>
    </ConsoleShell>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">{title}</h3>
      <div className="mt-1">{children}</div>
    </div>
  )
}
