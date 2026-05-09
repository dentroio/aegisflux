'use client'

import { useCallback, useMemo, useState } from 'react'
import { AlertTriangle, ArrowDown, RefreshCw, Save, Search, ShieldCheck, TerminalSquare } from 'lucide-react'
import {
  EmptyState,
  KpiTile,
  SummaryStrip,
  WorkbenchHeader,
} from '@/components/workbench/primitives'

type EvidencePathDraft = {
  control_id?: string
  title?: string
  action?: string
  target?: string
  scope?: string
  reason?: string
  blast_radius?: string[]
  rollback?: string[]
  evidence?: string[]
}

type EvidencePathResponse = {
  ok?: boolean
  subject?: { type?: string; id?: string; device_id?: string }
  summary?: string
  confidence_overall?: string
  missing_evidence?: string[]
  draft_controls?: EvidencePathDraft[]
  raw_findings?: any[]
}

type ProposalForm = {
  proposedAction: string
  scopeSelectors: string
  blastRadius: string
  blastRadiusNotes: string
  rollbackPlan: string
  rollbackSteps: string
  expectedBreakageRisk: string
  operatorNotes: string
  confidence: string
}

const DEFAULT_FORM: ProposalForm = {
  proposedAction: '',
  scopeSelectors: '',
  blastRadius: '',
  blastRadiusNotes: '',
  rollbackPlan: '',
  rollbackSteps: '',
  expectedBreakageRisk: 'low (observe-only; no enforcement)',
  operatorNotes: '',
  confidence: 'medium',
}

export type FindingToControlPanelProps = {
  initialFindingId?: string
  initialDeviceId?: string
  onDraftCreated?: () => void
  embedded?: boolean
}

export function FindingToControlPanel({
  initialFindingId = '',
  initialDeviceId = '',
  onDraftCreated,
  embedded = false,
}: FindingToControlPanelProps) {
  const [findingId, setFindingId] = useState(initialFindingId)
  const [deviceId, setDeviceId] = useState(initialDeviceId)
  const [evidence, setEvidence] = useState<EvidencePathResponse | null>(null)
  const [form, setForm] = useState<ProposalForm>(DEFAULT_FORM)
  const [loading, setLoading] = useState(false)
  const [savingDraft, setSavingDraft] = useState(false)
  const [simBusy, setSimBusy] = useState(false)
  const [simulationDeviceId, setSimulationDeviceId] = useState(initialDeviceId)
  const [savedDraft, setSavedDraft] = useState<{ id: string; matches?: number; status?: string } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [info, setInfo] = useState<string | null>(null)

  const canLoad = findingId.trim() !== '' || deviceId.trim() !== ''

  const loadEvidence = useCallback(async () => {
    if (!canLoad) {
      setError('Provide a finding id or device id to load evidence.')
      return
    }
    setLoading(true)
    setError(null)
    setInfo(null)
    try {
      const params = new URLSearchParams()
      if (findingId.trim()) params.set('finding_id', findingId.trim())
      if (deviceId.trim()) params.set('device_id', deviceId.trim())
      const res = await fetch(`/api/visibility/evidence-path?${params.toString()}`, { cache: 'no-store' })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      const body = (await res.json()) as EvidencePathResponse
      setEvidence(body)
      seedFormFromEvidence(body)
      if (body.subject?.device_id && !simulationDeviceId.trim()) {
        setSimulationDeviceId(body.subject.device_id)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load evidence path')
      setEvidence(null)
    } finally {
      setLoading(false)
    }
  }, [findingId, deviceId, simulationDeviceId, canLoad])

  function seedFormFromEvidence(body: EvidencePathResponse) {
    const draft = (body.draft_controls || [])[0]
    const findings = body.raw_findings || []
    const lead = findings[0] as Record<string, any> | undefined
    const findingTitle: string = lead?.title || lead?.classification || lead?.event_type || 'finding'
    const next: ProposalForm = {
      ...DEFAULT_FORM,
      confidence: body.confidence_overall || 'medium',
    }
    if (draft) {
      next.proposedAction = draft.action || ''
      next.scopeSelectors = draft.scope || ''
      next.blastRadius = `Observe-only projection. ${draft.reason || ''}`.trim()
      next.blastRadiusNotes = (draft.blast_radius || []).join('\n')
      next.rollbackPlan = (draft.rollback || []).slice(0, 1).join('\n') || DEFAULT_FORM.rollbackPlan
      next.rollbackSteps = (draft.rollback || []).join('\n')
    }
    if (!next.proposedAction) {
      next.proposedAction = `Observe-only review for ${findingTitle}`
    }
    if (!next.scopeSelectors && body.subject?.device_id) {
      next.scopeSelectors = `device:${body.subject.device_id}`
    }
    setForm(next)
  }

  function updateForm(field: keyof ProposalForm, value: string) {
    setForm((prev) => ({ ...prev, [field]: value }))
  }

  async function saveDraft() {
    setSavingDraft(true)
    setError(null)
    setInfo(null)
    try {
      const findings = evidence?.raw_findings as Array<Record<string, any>> | undefined
      const sourceFindingId =
        findingId.trim() ||
        (findings && (findings[0]?.finding_id || findings[0]?.event_id)) ||
        ''
      if (!sourceFindingId) {
        throw new Error('A finding id is required to save a draft control.')
      }
      const sourceTitle = findings?.[0]?.title || findings?.[0]?.classification || ''
      const targetDevice = (evidence?.subject?.device_id || deviceId).trim()
      const payload = {
        source_finding_id: sourceFindingId,
        source_finding_title: sourceTitle,
        source_device_id: targetDevice,
        proposed_action: form.proposedAction,
        scope_selectors: splitMulti(form.scopeSelectors),
        evidence_refs: evidenceRefsFromBody(evidence, sourceFindingId),
        expected_effect: 'observe_only',
        confidence: form.confidence,
        expected_breakage_risk: form.expectedBreakageRisk,
        blast_radius: form.blastRadius,
        blast_radius_notes: splitMulti(form.blastRadiusNotes),
        rollback_plan: form.rollbackPlan,
        rollback_steps: splitMulti(form.rollbackSteps),
        operator_notes: form.operatorNotes,
      }
      const res = await fetch('/api/actions/platform/draft-controls', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      const body = await res.json()
      const id = body?.id || body?.draft?.id
      const status = body?.draft?.status || 'draft_observe_only'
      setSavedDraft({ id, status })
      setInfo(`Draft control ${id} saved as observe-only.`)
      onDraftCreated?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save draft control')
    } finally {
      setSavingDraft(false)
    }
  }

  async function simulateDraft() {
    if (!savedDraft?.id) {
      setError('Save the draft before simulating.')
      return
    }
    setSimBusy(true)
    setError(null)
    setInfo(null)
    try {
      const target = simulationDeviceId.trim() || deviceId.trim() || evidence?.subject?.device_id || ''
      if (!target) {
        throw new Error('Provide a device id for the simulation projection.')
      }
      const res = await fetch(`/api/actions/platform/draft-controls/${encodeURIComponent(savedDraft.id)}/simulate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ device_id: target }),
      })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(text || `HTTP ${res.status}`)
      }
      const body = await res.json()
      const matches = Number(body?.matched_events ?? 0)
      setSavedDraft({ ...savedDraft, matches })
      setInfo(`Simulation projected ${matches} historical match(es) for ${target}. No enforcement.`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to simulate draft control')
    } finally {
      setSimBusy(false)
    }
  }

  const evidenceSummaryRows = useMemo(() => {
    if (!evidence) return []
    return [
      { label: 'Subject', value: subjectLabel(evidence) },
      { label: 'Confidence', value: evidence.confidence_overall || 'low' },
      { label: 'Path summary', value: evidence.summary || 'no summary' },
      { label: 'Missing evidence', value: (evidence.missing_evidence || []).join(', ') || 'none' },
    ]
  }, [evidence])

  return (
    <div className={embedded ? '' : 'min-h-screen bg-slate-50'}>
      <WorkbenchHeader
        title="Finding-to-control designer"
        subtitle="Turn a finding into an explainable observe-only draft control with blast radius and rollback notes."
        actions={
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void loadEvidence()}
              disabled={loading || !canLoad}
              className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-60"
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              Load evidence
            </button>
          </div>
        }
      />

      <section className="mb-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="grid gap-3 md:grid-cols-3">
          <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Finding id
            <input
              value={findingId}
              onChange={(event) => setFindingId(event.target.value)}
              className="mt-1 h-9 w-full rounded-md border border-slate-200 px-3 text-sm normal-case tracking-normal text-slate-900 outline-none focus:border-blue-500"
              placeholder="aegisflux finding id"
            />
          </label>
          <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Source device id
            <input
              value={deviceId}
              onChange={(event) => setDeviceId(event.target.value)}
              className="mt-1 h-9 w-full rounded-md border border-slate-200 px-3 text-sm normal-case tracking-normal text-slate-900 outline-none focus:border-blue-500"
              placeholder="optional"
            />
          </label>
          <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Simulation device id
            <input
              value={simulationDeviceId}
              onChange={(event) => setSimulationDeviceId(event.target.value)}
              className="mt-1 h-9 w-full rounded-md border border-slate-200 px-3 text-sm normal-case tracking-normal text-slate-900 outline-none focus:border-blue-500"
              placeholder="device id for projection"
            />
          </label>
        </div>
        <button
          type="button"
          onClick={() => void loadEvidence()}
          disabled={loading || !canLoad}
          className="mt-3 inline-flex h-9 items-center gap-2 rounded-md border border-blue-600 bg-blue-600 px-3 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-60"
        >
          <Search className="h-4 w-4" />
          {loading ? 'Loading…' : 'Load evidence path'}
        </button>
      </section>

      {error ? (
        <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">{error}</div>
      ) : null}
      {info ? (
        <div className="mb-4 rounded-md border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900">{info}</div>
      ) : null}

      {!evidence && !loading ? (
        <EmptyState
          title="No evidence loaded"
          message="Provide a finding id (and optionally a device id), then click Load evidence path to seed a proposal."
        />
      ) : null}

      {evidence ? (
        <>
          <SummaryStrip>
            <KpiTile label="Subject" value={subjectKpi(evidence)} />
            <KpiTile label="Path nodes" value={(evidence as any).nodes?.length || 0} />
            <KpiTile label="Confidence" value={evidence.confidence_overall || 'low'} />
            <KpiTile label="Drafts seeded" value={(evidence.draft_controls || []).length} />
          </SummaryStrip>

          <section className="mb-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-slate-500">Evidence summary</h3>
            <dl className="grid gap-2 md:grid-cols-2">
              {evidenceSummaryRows.map((row) => (
                <div key={row.label}>
                  <dt className="text-xs font-semibold text-slate-500">{row.label}</dt>
                  <dd className="text-sm text-slate-800" title={row.value}>{row.value || 'n/a'}</dd>
                </div>
              ))}
            </dl>
            {(evidence.missing_evidence || []).length > 0 ? (
              <div className="mt-3 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
                <div className="flex items-center gap-2 font-semibold">
                  <AlertTriangle className="h-4 w-4" />
                  Confidence is bounded by missing evidence — review before promoting.
                </div>
              </div>
            ) : null}
          </section>

          <section className="mb-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <h3 className="mb-3 inline-flex items-center gap-2 text-sm font-semibold text-slate-900">
              <TerminalSquare className="h-4 w-4 text-blue-600" />
              Proposal
            </h3>
            <div className="grid gap-3 md:grid-cols-2">
              <Field label="Proposed action">
                <textarea
                  value={form.proposedAction}
                  onChange={(event) => updateForm('proposedAction', event.target.value)}
                  rows={2}
                  className="h-20 w-full rounded-md border border-slate-200 px-3 py-2 text-sm outline-none focus:border-blue-500"
                />
              </Field>
              <Field label="Scope selectors (comma or newline)">
                <textarea
                  value={form.scopeSelectors}
                  onChange={(event) => updateForm('scopeSelectors', event.target.value)}
                  rows={2}
                  className="h-20 w-full rounded-md border border-slate-200 px-3 py-2 text-sm outline-none focus:border-blue-500"
                />
              </Field>
              <Field label="Blast radius summary">
                <textarea
                  value={form.blastRadius}
                  onChange={(event) => updateForm('blastRadius', event.target.value)}
                  rows={2}
                  className="h-20 w-full rounded-md border border-slate-200 px-3 py-2 text-sm outline-none focus:border-blue-500"
                />
              </Field>
              <Field label="Blast radius notes (one per line)">
                <textarea
                  value={form.blastRadiusNotes}
                  onChange={(event) => updateForm('blastRadiusNotes', event.target.value)}
                  rows={3}
                  className="h-24 w-full rounded-md border border-slate-200 px-3 py-2 text-sm outline-none focus:border-blue-500"
                />
              </Field>
              <Field label="Rollback plan">
                <textarea
                  value={form.rollbackPlan}
                  onChange={(event) => updateForm('rollbackPlan', event.target.value)}
                  rows={2}
                  className="h-20 w-full rounded-md border border-slate-200 px-3 py-2 text-sm outline-none focus:border-blue-500"
                />
              </Field>
              <Field label="Rollback steps (one per line)">
                <textarea
                  value={form.rollbackSteps}
                  onChange={(event) => updateForm('rollbackSteps', event.target.value)}
                  rows={3}
                  className="h-24 w-full rounded-md border border-slate-200 px-3 py-2 text-sm outline-none focus:border-blue-500"
                />
              </Field>
              <Field label="Expected breakage risk">
                <input
                  value={form.expectedBreakageRisk}
                  onChange={(event) => updateForm('expectedBreakageRisk', event.target.value)}
                  className="h-9 w-full rounded-md border border-slate-200 px-3 text-sm outline-none focus:border-blue-500"
                />
              </Field>
              <Field label="Operator notes">
                <textarea
                  value={form.operatorNotes}
                  onChange={(event) => updateForm('operatorNotes', event.target.value)}
                  rows={3}
                  className="h-24 w-full rounded-md border border-slate-200 px-3 py-2 text-sm outline-none focus:border-blue-500"
                  placeholder="Why this scope, who owns this finding, what is needed before promotion."
                />
              </Field>
              <Field label="Confidence">
                <select
                  value={form.confidence}
                  onChange={(event) => updateForm('confidence', event.target.value)}
                  className="h-9 w-full rounded-md border border-slate-200 px-3 text-sm outline-none focus:border-blue-500"
                >
                  <option value="high">high</option>
                  <option value="medium">medium</option>
                  <option value="low">low</option>
                </select>
              </Field>
            </div>
          </section>

          <section className="mb-4 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void saveDraft()}
              disabled={savingDraft}
              className="inline-flex h-10 items-center gap-2 rounded-md border border-blue-600 bg-blue-600 px-4 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-60"
            >
              <Save className="h-4 w-4" />
              {savingDraft ? 'Saving…' : 'Save observe-only draft'}
            </button>
            <ArrowDown className="h-4 w-4 text-slate-400" />
            <button
              type="button"
              onClick={() => void simulateDraft()}
              disabled={simBusy || !savedDraft}
              className="inline-flex h-10 items-center gap-2 rounded-md border border-emerald-600 bg-emerald-600 px-4 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-60"
            >
              <ShieldCheck className="h-4 w-4" />
              {simBusy ? 'Simulating…' : 'Simulate blast radius'}
            </button>
            {savedDraft ? (
              <span className="text-xs text-slate-600">
                Draft <span className="font-mono">{savedDraft.id.slice(0, 8)}</span>{' '}
                {typeof savedDraft.matches === 'number'
                  ? `· projected ${savedDraft.matches} historical matches`
                  : '· not yet simulated'}
              </span>
            ) : null}
          </section>

          <section className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-xs text-amber-900">
            <div className="flex items-center gap-2 font-semibold">
              <AlertTriangle className="h-4 w-4" />
              Observe-only
            </div>
            <p className="mt-1">
              Saving a draft does not enforce policy. Simulation projects historical matches against existing telemetry so the
              operator can review scope and rollback before any future enforcement project.
            </p>
          </section>
        </>
      ) : null}
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
      {label}
      <div className="mt-1 normal-case tracking-normal">{children}</div>
    </label>
  )
}

function splitMulti(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

function evidenceRefsFromBody(body: EvidencePathResponse | null, fallback: string): string[] {
  if (!body) return [fallback]
  const refs: string[] = []
  for (const draft of body.draft_controls || []) {
    for (const ref of draft.evidence || []) refs.push(ref)
  }
  refs.push(`finding:${fallback}`)
  return Array.from(new Set(refs)).slice(0, 16)
}

function subjectLabel(body: EvidencePathResponse): string {
  const subj = body.subject
  if (!subj) return 'unknown subject'
  if (subj.type === 'finding') return `Finding ${subj.id || 'unknown'} on ${subj.device_id || 'unknown'}`
  if (subj.type === 'process') return `Process ${subj.id || 'unknown'} on ${subj.device_id || 'unknown'}`
  return `Endpoint ${subj.device_id || subj.id || 'unknown'}`
}

function subjectKpi(body: EvidencePathResponse): string {
  const subj = body.subject
  if (!subj) return '—'
  if (subj.type === 'finding') return 'Finding'
  if (subj.type === 'process') return 'Process'
  return 'Endpoint'
}
