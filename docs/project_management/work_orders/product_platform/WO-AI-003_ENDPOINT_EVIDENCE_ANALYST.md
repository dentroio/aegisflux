# WO-AI-003: Endpoint Evidence Analyst

**Status:** Completed (May 2026 — lab slice implemented)
**Phase:** Product Platform  
**Primary owner:** Backend / UI  

## Goal

Add the first AegisFlux AI platform agent action: explain AI activity on a selected endpoint using bounded, auditable context.

## Scope

- Device detail AI action.
- Bounded context builder.
- Backend AI-agent route.
- Audited response display.

## Deliverables

- Registered AI agent: `endpoint_evidence_analyst`.
- UI action on device detail: "Explain AI activity".
- Context object containing:
  - device id
  - source OS/sensor
  - selected time window
  - relevant findings
  - AI destination DNS records
  - process lineage evidence
  - flow evidence
  - browser extension evidence
  - SASE/SSE component evidence
  - detection pack versions when available
- Response sections:
  - Assessment
  - Evidence
  - Confidence
  - Recommended next action
- AI run/audit record linked to device id.

## Acceptance Criteria

- Action works for Windows lab device.
- Action works for Linux lab device even when browser evidence is absent.
- If AI is unavailable, UI shows deterministic fallback summary.
- AI response includes evidence references, not generic claims.
- `npm run build` passes.

## Dependencies

- WO-PLAT-002
- WO-AI-002

## Non-Goals

- No automatic enforcement.
- No detection-pack approval.

