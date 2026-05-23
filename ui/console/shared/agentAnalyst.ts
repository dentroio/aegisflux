import type { EvidenceBoundConclusion } from '@/components/EvidenceBoundConclusionPanel'

export type AnalystApiResponse = {
  run_id?: string
  assessment?: string
  evidence?: string
  confidence?: string
  recommended_next_action?: string
  evidence_bound_conclusion?: EvidenceBoundConclusion
  evidence_bound_validation_errors?: string[]
  error?: string
  status?: string
}

export type AnalystParseResult = {
  conclusion: EvidenceBoundConclusion | null
  error: string | null
  validationErrors?: string[]
  legacyApi?: boolean
}

function normalizeConclusion(c: EvidenceBoundConclusion): EvidenceBoundConclusion {
  return {
    ...c,
    evidence: c.evidence ?? [],
    assumptions: c.assumptions ?? [],
    missing_evidence: c.missing_evidence ?? [],
    safety_boundaries: c.safety_boundaries ?? [],
    recommendations: c.recommendations ?? [],
  }
}

function confidenceBucketFromLegacy(confidence?: string): string {
  const s = (confidence || '').toLowerCase()
  if (s.includes('high')) return 'high'
  if (s.includes('low')) return 'low'
  if (s.includes('medium')) return 'medium'
  return 'unknown'
}

/** Maps pre-harness analyst JSON into the evidence-bound shape for display. */
export function legacyAnalystToConclusion(body: AnalystApiResponse): EvidenceBoundConclusion {
  return normalizeConclusion({
    conclusion: body.assessment || 'No assessment returned.',
    evidence: [],
    assumptions: [
      'Response came from an older Actions API build without the agent harness / evidence-bound contract.',
      'Restart or rebuild actions-api to enable cited evidence, harness runs, and validation.',
    ],
    missing_evidence: [
      {
        category: 'process_telemetry',
        detail: 'Structured evidence refs require actions-api with WO-AGENTS-001/002 routes.',
      },
    ],
    confidence_bucket: confidenceBucketFromLegacy(body.confidence),
    confidence_rationale: body.confidence || 'Legacy confidence string from API.',
    safety_boundaries: ['Observe-only analysis; no enforcement implied by this endpoint.'],
    recommendations: body.recommended_next_action ? [body.recommended_next_action] : [],
  })
}

export function parseAnalystResponse(res: Response, body: AnalystApiResponse | null): AnalystParseResult {
  if (!body) {
    return {
      conclusion: null,
      error: res.ok ? 'Empty response from analyst API' : `Analyst request failed (HTTP ${res.status})`,
    }
  }

  if (body.evidence_bound_conclusion) {
    return {
      conclusion: normalizeConclusion(body.evidence_bound_conclusion),
      error: res.ok ? null : body.error || `Validation failed (HTTP ${res.status})`,
      validationErrors: body.evidence_bound_validation_errors,
    }
  }

  if (res.ok && body.assessment) {
    return {
      conclusion: legacyAnalystToConclusion(body),
      error: null,
      legacyApi: true,
    }
  }

  return {
    conclusion: null,
    error: body.error || `Analyst request failed (HTTP ${res.status})`,
  }
}

export async function fetchHarnessAvailable(): Promise<boolean> {
  try {
    const res = await fetch('/api/actions/platform/ai/agent-harness/agents', { cache: 'no-store' })
    return res.ok
  } catch {
    return false
  }
}
