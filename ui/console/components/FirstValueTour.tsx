'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import {
  ArrowRight,
  Bot,
  CheckCircle2,
  Clock,
  Compass,
  Feather,
  GraduationCap,
  Network,
  Newspaper,
  Play,
  ShieldCheck,
  Sparkles,
  Stars,
  XCircle,
} from 'lucide-react'

const STORAGE_KEY = 'aegisflux.firstValueDemo.v1'

type ProgressState = {
  startedAt?: number
  completedAt?: number
  visited: Record<string, number>
  skipped?: boolean
}

function loadProgress(): ProgressState {
  if (typeof window === 'undefined') return { visited: {} }
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return { visited: {} }
    const parsed = JSON.parse(raw) as ProgressState
    return { ...parsed, visited: parsed.visited || {} }
  } catch {
    return { visited: {} }
  }
}

function saveProgress(state: ProgressState) {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    // ignore
  }
}

export function FirstValueTourBanner({ href = '/demo' }: { href?: string }) {
  const [shouldShow, setShouldShow] = useState(false)

  useEffect(() => {
    setShouldShow(!tourCompleted())
  }, [])

  if (!shouldShow) return null

  return (
    <section className="mb-4 rounded-xl border border-blue-200 bg-blue-50 p-4 shadow-sm">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="flex items-start gap-3">
          <span className="mt-0.5 inline-flex h-9 w-9 items-center justify-center rounded-md bg-blue-600 text-white">
            <GraduationCap className="h-5 w-5" />
          </span>
          <div>
            <h3 className="text-sm font-semibold text-blue-900">Take the five-minute tour</h3>
            <p className="text-xs text-blue-900/80">
              Discover, investigate, design, adapt — see the four AegisFlux pillars with live links into the product.
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Link
            href={href}
            className="inline-flex h-9 items-center gap-2 rounded-md border border-blue-600 bg-blue-600 px-3 text-sm font-semibold text-white hover:bg-blue-700"
          >
            <Play className="h-4 w-4" />
            Start tour
          </Link>
          <button
            type="button"
            onClick={() => {
              markTourSkipped()
              setShouldShow(false)
            }}
            className="inline-flex h-9 items-center gap-2 rounded-md border border-blue-200 bg-white px-3 text-sm font-semibold text-blue-700 hover:bg-blue-100"
          >
            Skip
          </button>
        </div>
      </div>
    </section>
  )
}

export function tourCompleted(): boolean {
  if (typeof window === 'undefined') return false
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return false
    const parsed = JSON.parse(raw) as ProgressState
    return Boolean(parsed.completedAt) || Boolean(parsed.skipped)
  } catch {
    return false
  }
}

export function markTourCompleted() {
  const state = loadProgress()
  state.completedAt = Date.now()
  saveProgress(state)
}

export function markTourSkipped() {
  const state = loadProgress()
  state.skipped = true
  saveProgress(state)
}

export function resetTour() {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.removeItem(STORAGE_KEY)
  } catch {
    // ignore
  }
}

export type FirstValueStep = {
  id: string
  title: string
  oneLiner: string
  paragraph: string
  productPillar: 'discover' | 'investigate' | 'design' | 'adapt'
  estimateSeconds: number
  ctaLabel: string
  ctaHref: string
  icon: typeof Sparkles
}

const STEPS: FirstValueStep[] = [
  {
    id: 'welcome',
    title: 'Welcome to AegisFlux',
    oneLiner: 'A control-design assistant for AI on the endpoint.',
    paragraph:
      'AegisFlux helps operators see AI-capable tools, trust why a finding matters, design observe-only controls, and adapt to new AI tools — in under five minutes per shift.',
    productPillar: 'discover',
    estimateSeconds: 30,
    ctaLabel: 'Start tour',
    ctaHref: '#step-1',
    icon: GraduationCap,
  },
  {
    id: 'discover',
    title: 'Discover — Agent Bill of Materials',
    oneLiner: 'See AI-capable tools, capabilities, and reachability across endpoints.',
    paragraph:
      'The Agent Bill of Materials inventories every AI desktop app, browser extension, coding agent, MCP endpoint, local model runtime, and model gateway across the fleet — with confidence and evidence per item.',
    productPillar: 'discover',
    estimateSeconds: 60,
    ctaLabel: 'Open Agent Bill of Materials',
    ctaHref: '/discover/abom',
    icon: Sparkles,
  },
  {
    id: 'investigate',
    title: 'Investigate — Evidence path',
    oneLiner: 'Turn raw telemetry into a trusted finding → control path.',
    paragraph:
      'The evidence path stitches finding, parent process, process, network flow, DNS lookup, endpoint, detection pack, and supporting draft control into one explainable story — with explicit missing-evidence callouts.',
    productPillar: 'investigate',
    estimateSeconds: 90,
    ctaLabel: 'Open evidence path',
    ctaHref: '/analyze/evidence',
    icon: Network,
  },
  {
    id: 'design',
    title: 'Design — Finding-to-control',
    oneLiner: 'Draft observe-only controls with blast radius and rollback.',
    paragraph:
      'Turn evidence into thoughtful, observe-only draft controls — with proposed scope, expected blast radius, rollback steps, operator notes, and a deterministic lab simulation for projected matches.',
    productPillar: 'design',
    estimateSeconds: 90,
    ctaLabel: 'Open finding-to-control designer',
    ctaHref: '/control/controls',
    icon: ShieldCheck,
  },
  {
    id: 'adapt',
    title: 'Adapt — AI research feed',
    oneLiner: 'Govern new AI ecosystem intelligence into detection opportunities.',
    paragraph:
      'New AI tools appear weekly. The research feed gives every item a clear lifecycle (new → scoped → ready_for_pack → promoted/declined), with indicators, evidence required, and guard rails — with promotion that is observe-only by default.',
    productPillar: 'adapt',
    estimateSeconds: 60,
    ctaLabel: 'Open AI research feed',
    ctaHref: '/analyze/research',
    icon: Newspaper,
  },
  {
    id: 'wrap',
    title: 'Done in five minutes',
    oneLiner: 'Discover. Investigate. Design. Adapt.',
    paragraph:
      'You have seen the four product pillars. From the dashboard you can return to any of them, or pull up the always-on tour from the help menu later.',
    productPillar: 'discover',
    estimateSeconds: 20,
    ctaLabel: 'Return to dashboard',
    ctaHref: '/',
    icon: CheckCircle2,
  },
]

const PILLAR_TONE: Record<FirstValueStep['productPillar'], string> = {
  discover: 'border-cyan-200 bg-cyan-50 text-cyan-900',
  investigate: 'border-blue-200 bg-blue-50 text-blue-900',
  design: 'border-emerald-200 bg-emerald-50 text-emerald-900',
  adapt: 'border-amber-200 bg-amber-50 text-amber-900',
}

const PILLAR_ICON: Record<FirstValueStep['productPillar'], typeof Compass> = {
  discover: Compass,
  investigate: Stars,
  design: Feather,
  adapt: Bot,
}

export function FirstValueTour() {
  const [progress, setProgress] = useState<ProgressState>({ visited: {} })
  const [activeStep, setActiveStep] = useState<string>('welcome')

  useEffect(() => {
    setProgress(loadProgress())
  }, [])

  useEffect(() => {
    if (!progress.startedAt) {
      const next: ProgressState = { ...progress, startedAt: Date.now() }
      setProgress(next)
      saveProgress(next)
    }
  }, [progress])

  const totalSeconds = useMemo(() => STEPS.reduce((acc, step) => acc + step.estimateSeconds, 0), [])
  const visitedCount = useMemo(() => Object.keys(progress.visited).length, [progress.visited])
  const visitedSeconds = useMemo(
    () => STEPS.filter((step) => progress.visited[step.id]).reduce((acc, step) => acc + step.estimateSeconds, 0),
    [progress.visited],
  )
  const remainingSeconds = Math.max(0, totalSeconds - visitedSeconds)

  const visit = useCallback(
    (id: string) => {
      const next: ProgressState = {
        ...progress,
        visited: { ...progress.visited, [id]: Date.now() },
      }
      setProgress(next)
      saveProgress(next)
    },
    [progress],
  )

  const completeTour = useCallback(() => {
    const next: ProgressState = { ...progress, completedAt: Date.now(), visited: { ...progress.visited } }
    for (const step of STEPS) next.visited[step.id] = next.visited[step.id] || Date.now()
    setProgress(next)
    saveProgress(next)
  }, [progress])

  const skipTour = useCallback(() => {
    const next: ProgressState = { ...progress, skipped: true }
    setProgress(next)
    saveProgress(next)
  }, [progress])

  const reset = useCallback(() => {
    resetTour()
    setProgress({ visited: {} })
    setActiveStep('welcome')
  }, [])

  return (
    <div className="space-y-6">
      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">First-value tour</h1>
            <p className="mt-1 max-w-2xl text-sm text-slate-600">
              Five focused steps. Roughly {Math.round(totalSeconds / 60)} minutes total. Each step opens a real part of the product
              so the value is in the workflow, not in slides.
            </p>
          </div>
          <div className="flex items-center gap-3 text-xs text-slate-600">
            <span className="inline-flex items-center gap-1 rounded-full border border-slate-200 px-2 py-1">
              <Clock className="h-3.5 w-3.5" />
              {Math.round(remainingSeconds / 60)} min left
            </span>
            <span className="inline-flex items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-1 text-emerald-800">
              <CheckCircle2 className="h-3.5 w-3.5" />
              {visitedCount} / {STEPS.length} visited
            </span>
            {progress.completedAt ? (
              <span className="inline-flex items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-1 text-emerald-800">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Completed
              </span>
            ) : null}
          </div>
        </div>

        <div className="mt-5 grid gap-3 md:grid-cols-4">
          {(['discover', 'investigate', 'design', 'adapt'] as const).map((pillar) => {
            const Icon = PILLAR_ICON[pillar]
            return (
              <div key={pillar} className={`rounded-xl border p-3 ${PILLAR_TONE[pillar]}`}>
                <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide">
                  <Icon className="h-4 w-4" />
                  {pillar}
                </div>
                <p className="mt-1 text-sm font-semibold">
                  {pillar === 'discover' ? 'Agent Bill of Materials' : null}
                  {pillar === 'investigate' ? 'Evidence path' : null}
                  {pillar === 'design' ? 'Finding-to-control' : null}
                  {pillar === 'adapt' ? 'AI research feed' : null}
                </p>
              </div>
            )
          })}
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            onClick={skipTour}
            className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
          >
            <XCircle className="h-4 w-4" />
            Skip for now
          </button>
          <button
            type="button"
            onClick={completeTour}
            className="inline-flex h-9 items-center gap-2 rounded-md border border-blue-600 bg-blue-600 px-3 text-sm font-semibold text-white hover:bg-blue-700"
          >
            <CheckCircle2 className="h-4 w-4" />
            Mark all visited
          </button>
          <button
            type="button"
            onClick={reset}
            className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
          >
            <Play className="h-4 w-4" />
            Reset progress
          </button>
        </div>
      </header>

      <ol className="space-y-4">
        {STEPS.map((step, idx) => {
          const visitedAt = progress.visited[step.id]
          const expanded = activeStep === step.id || !!visitedAt
          const Icon = step.icon
          return (
            <li
              key={step.id}
              id={`step-${idx}`}
              className={`rounded-2xl border bg-white shadow-sm transition ${
                visitedAt ? 'border-emerald-200' : 'border-slate-200'
              }`}
            >
              <button
                type="button"
                onClick={() => setActiveStep(step.id)}
                className="flex w-full items-start gap-3 px-5 py-4 text-left"
                aria-expanded={expanded}
              >
                <div
                  className={`mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${
                    visitedAt ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-700'
                  }`}
                >
                  <Icon className="h-5 w-5" />
                </div>
                <div className="flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                      Step {idx} · {step.productPillar}
                    </span>
                    <span className="inline-flex items-center gap-1 text-xs text-slate-500">
                      <Clock className="h-3 w-3" />
                      {Math.round(step.estimateSeconds)}s
                    </span>
                    {visitedAt ? (
                      <span className="inline-flex items-center gap-1 text-xs text-emerald-700">
                        <CheckCircle2 className="h-3 w-3" />
                        Visited
                      </span>
                    ) : null}
                  </div>
                  <h2 className="mt-1 text-lg font-semibold text-slate-900">{step.title}</h2>
                  <p className="text-sm text-slate-700">{step.oneLiner}</p>
                </div>
                <ArrowRight className={`mt-1 h-5 w-5 shrink-0 transition ${expanded ? 'rotate-90 text-blue-700' : 'text-slate-400'}`} />
              </button>
              {expanded ? (
                <div className="border-t border-slate-100 px-5 py-4">
                  <p className="text-sm text-slate-700">{step.paragraph}</p>
                  <div className="mt-3 flex flex-wrap items-center gap-3">
                    {step.ctaHref.startsWith('#') ? (
                      <button
                        type="button"
                        className="inline-flex h-9 items-center gap-2 rounded-md border border-blue-600 bg-blue-600 px-3 text-sm font-semibold text-white hover:bg-blue-700"
                        onClick={() => {
                          visit(step.id)
                          const nextStep = STEPS[idx + 1]
                          if (nextStep) setActiveStep(nextStep.id)
                        }}
                      >
                        {step.ctaLabel}
                      </button>
                    ) : (
                      <a
                        href={step.ctaHref}
                        onClick={() => visit(step.id)}
                        className="inline-flex h-9 items-center gap-2 rounded-md border border-blue-600 bg-blue-600 px-3 text-sm font-semibold text-white hover:bg-blue-700"
                      >
                        {step.ctaLabel}
                        <ArrowRight className="h-4 w-4" />
                      </a>
                    )}
                    {!visitedAt ? (
                      <button
                        type="button"
                        onClick={() => visit(step.id)}
                        className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
                      >
                        Mark visited
                      </button>
                    ) : null}
                  </div>
                </div>
              ) : null}
            </li>
          )
        })}
      </ol>
    </div>
  )
}
