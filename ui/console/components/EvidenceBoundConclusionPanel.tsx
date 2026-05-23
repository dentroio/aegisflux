'use client'

/** WO-AGENTS-002: shared evidence-bound conclusion shape from Actions API harness. */

export type EvidenceRef = {
  kind: string
  ref: string
  detail?: string
}

export type MissingEvidenceItem = {
  category: string
  detail?: string
}

export type EvidenceBoundConclusion = {
  conclusion: string
  evidence: EvidenceRef[]
  assumptions: string[]
  missing_evidence: MissingEvidenceItem[]
  confidence_bucket: string
  confidence_rationale: string
  safety_boundaries: string[]
  recommendations?: string[]
}

function confidenceTone(bucket: string) {
  switch (bucket) {
    case 'high':
      return 'bg-emerald-100 text-emerald-800 border-emerald-200'
    case 'medium':
      return 'bg-amber-100 text-amber-800 border-amber-200'
    case 'low':
      return 'bg-orange-100 text-orange-800 border-orange-200'
    default:
      return 'bg-slate-100 text-slate-700 border-slate-200'
  }
}

function formatKind(kind: string) {
  return kind.replace(/_/g, ' ')
}

type Props = {
  conclusion: EvidenceBoundConclusion
  validationErrors?: string[]
  compact?: boolean
}

export function EvidenceBoundConclusionPanel({ conclusion, validationErrors, compact }: Props) {
  const c = {
    ...conclusion,
    evidence: conclusion.evidence ?? [],
    assumptions: conclusion.assumptions ?? [],
    missing_evidence: conclusion.missing_evidence ?? [],
    safety_boundaries: conclusion.safety_boundaries ?? [],
    recommendations: conclusion.recommendations ?? [],
  }
  return (
    <div className="space-y-4 text-sm text-slate-800">
      {validationErrors && validationErrors.length > 0 ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-800">
          <div className="font-semibold">Validation failed</div>
          <ul className="mt-1 list-inside list-disc">
            {validationErrors.map((e) => (
              <li key={e}>{e}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">Conclusion</span>
          <span
            className={`inline-flex rounded-full border px-2 py-0.5 text-xs font-semibold capitalize ${confidenceTone(c.confidence_bucket)}`}
          >
            {c.confidence_bucket} confidence
          </span>
        </div>
        <p className={`mt-2 text-slate-900 ${compact ? 'text-sm' : 'text-base'}`}>{c.conclusion}</p>
        <p className="mt-2 text-xs text-slate-600">{c.confidence_rationale}</p>
      </div>

      {c.evidence.length > 0 ? (
        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-emerald-700">Evidence cited</h3>
          <ul className="mt-2 space-y-2">
            {c.evidence.map((e, i) => (
              <li key={`${e.kind}-${e.ref}-${i}`} className="rounded-lg border border-emerald-100 bg-emerald-50/50 px-3 py-2 text-xs">
                <div className="font-mono font-semibold text-emerald-900">
                  {formatKind(e.kind)} · {e.ref}
                </div>
                {e.detail ? <div className="mt-1 text-slate-600">{e.detail}</div> : null}
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {c.missing_evidence.length > 0 ? (
        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-amber-700">Missing evidence</h3>
          <ul className="mt-2 space-y-2">
            {c.missing_evidence.map((m, i) => (
              <li key={`${m.category}-${i}`} className="rounded-lg border border-amber-100 bg-amber-50 px-3 py-2 text-xs text-amber-950">
                <span className="font-semibold">{formatKind(m.category)}</span>
                {m.detail ? <span className="text-amber-800"> — {m.detail}</span> : null}
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {c.assumptions.length > 0 ? (
        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Assumptions</h3>
          <ul className="mt-2 list-inside list-disc space-y-1 text-xs text-slate-600">
            {c.assumptions.map((a, i) => (
              <li key={i}>{a}</li>
            ))}
          </ul>
        </section>
      ) : null}

      {c.safety_boundaries.length > 0 ? (
        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-blue-700">Safety boundaries</h3>
          <ul className="mt-2 space-y-1 rounded-lg border border-blue-100 bg-blue-50/60 px-3 py-2 text-xs text-blue-950">
            {c.safety_boundaries.map((b, i) => (
              <li key={i}>{b}</li>
            ))}
          </ul>
        </section>
      ) : null}

      {c.recommendations && c.recommendations.length > 0 ? (
        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wide text-violet-700">Recommendations</h3>
          <ul className="mt-2 list-inside list-disc space-y-1 text-xs text-violet-900">
            {c.recommendations.map((r, i) => (
              <li key={i}>{r}</li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  )
}
