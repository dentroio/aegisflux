# WO-AGENTS-002: Evidence-Bound Reasoning Contract

**Status:** Implemented (lab)  
**Phase:** AI-Native Leap  
**Primary owner:** AI Platform / Backend / Product  

## Goal

Require every AI agent conclusion to cite evidence, confidence, missing evidence, assumptions, and safety boundaries.

## Problem

Security operators cannot trust AI-generated recommendations unless they can see exactly which endpoint evidence supports the claim and what remains unknown. AegisFlux needs an output contract that makes uncertainty explicit.

## Scope

- Agent output schemas.
- Evidence reference model.
- Confidence and missing-evidence taxonomy.
- Validation for agent outputs that propose detections, controls, or approval packets.

## Deliverables

- Define shared `EvidenceBoundConclusion` schema.
- Add evidence reference types for process, DNS, flow, finding, ABOM item, detection pack, audit bundle, operational event, and collector health.
- Add confidence buckets and required rationale.
- Add missing-evidence categories.
- Add schema validation to agent run completion.
- Add UI rendering pattern for evidence-cited conclusions.

## Acceptance Criteria

- Agent runs cannot complete with a product-impacting conclusion unless the evidence-bound contract validates.
- Missing evidence can be represented without treating the run as failed.
- UI distinguishes evidence, assumptions, missing evidence, and recommendations.
- Tests cover valid, invalid, and low-confidence conclusions.

## Dependencies

- WO-AGENTS-001.
- WO-PROD-002.
- WO-GROWTH-002.

## Non-Goals

- No claim that AI reasoning is automatically correct.
- No hidden chain-of-thought storage requirement.
- No free-form conclusion format for governed agents.

## Suggested Verification

- Schema/unit tests for conclusion validation.
- Manual Endpoint Analyst stub output with cited and missing evidence.

## Implementation notes (May 12, 2026)

- **Schema + validation:** `backend/actions-api/internal/agentsharness/evidence_bound.go` — `EvidenceBoundConclusion`, evidence ref kinds (process, dns, flow, finding, abom_item, detection_pack, audit_bundle, operational_event, collector_health, plus lab integration/candidate kinds), missing-evidence categories, confidence buckets (`low`/`medium`/`high`/`unknown`), `ValidateEvidenceBoundConclusion`.
- **Synthesis:** `evidence_bound_build.go` builds conclusions from audited harness tool outputs; `harness.go` validates before `completed` (product-impacting default **on**); invalid → run/job **failed** with `evidence_bound_validation_errors`.
- **API:** `RunRecord.evidence_bound_conclusion`; `POST /platform/ai/endpoint-evidence-analyst` returns conclusion on success; **422** when validation fails.
- **UI:** `ui/console/components/EvidenceBoundConclusionPanel.tsx`; harness run detail on AI Providers; device **Explain AI activity** panel.

## Verification results (lab)

| Check | Result |
|-------|--------|
| `go test ./internal/agentsharness/...` | Pass (validation + build round-trip) |
| `go test ./internal/api/...` | Pass (harness + endpoint analyst returns `evidence_bound_conclusion`) |
| `npx tsc --noEmit` in `ui/console` | Pass |
| Manual | Run endpoint analyst or harness job; confirm cited evidence, missing evidence, assumptions, safety boundaries in UI |
