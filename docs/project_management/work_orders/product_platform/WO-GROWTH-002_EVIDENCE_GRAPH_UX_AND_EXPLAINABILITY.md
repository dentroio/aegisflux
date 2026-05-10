# WO-GROWTH-002: Evidence Graph UX and Explainability

**Status:** Done  
**Phase:** Product Growth / Investigation  
**Primary owner:** UI / Backend  

## Goal

Make the evidence graph feel like an explanation, not a data structure. Operators should immediately understand what happened, why Aegis thinks it matters, and what evidence is missing.

## Why This Matters

If operators still ask "what does process mean?" then the product is leaking implementation details. AegisFlux should translate endpoint telemetry into a trusted narrative:

1. This activity happened.
2. This is the program/user/destination involved.
3. This is why it matters.
4. This is what evidence is strong or missing.
5. This is the recommended next step.

## Scope

- Evidence graph UI refinement.
- Better summary language.
- Missing-evidence explanations.
- Linkage into ABOM and control design.

## Deliverables

- Backend `enrichEvidenceForOperator` in `evidence_path_query.go`:
  - Produces an `evidenceNarrative` block (`what_happened`, `why_it_matters`, `what_we_know`, `what_is_missing`, `recommended_next_step`).
  - Adds per-node `operator_label` (e.g. "Program that ran", "Where it talked to") and `confidence_reason` strings that explain why the row is trusted or marked missing.
  - Cross-links process and DNS nodes to ABOM via `related_abom_id` / `related_abom_label` using the existing ABOM detection regexes.
  - Returns an `overall_confidence_reason` string that names the evidence anchors and gaps.
- Response shape extended with `narrative`, `confidence_reason`, and the new per-node fields. Existing fields kept for backwards compatibility.
- Tests in `evidence_path_query_test.go`:
  - `TestEnrichEvidenceForOperator_NarrativeAndLabels` covers the full-path case with operator labels, ABOM cross-links on the process and DNS nodes, and a populated narrative.
  - `TestEnrichEvidenceForOperator_PartialEvidenceMissingCopy` covers the partial-evidence case with explicit missing reasons and narrative gaps.
- Console `EvidenceGraphPanel`:
  - New `NarrativeBlock` rendered at the top of the path with five callouts (What happened / Why it matters / What we know / What is missing / Recommended next step) plus an overall confidence reason chip.
  - Prominent "Design observe-only control" CTA that deep-links into the finding-to-control designer with `finding_id` and `device_id` query params, plus the seeded draft id when present.
  - Per-node card shows the operator label first and falls back to the underlying node type, plus the new confidence reason text.
  - "Open related ABOM item" link surfaced when a process or DNS row maps to an ABOM category.
  - Raw evidence drawer remains bounded and collapsible (unchanged).

## Acceptance Criteria

- [x] Evidence path can be understood without reading JSON (narrative block + operator labels).
- [x] Missing evidence has actionable explanations (per-node confidence reason + narrative `what_is_missing` bullets).
- [x] Graph/path UI avoids long scroll by default (narrative block plus existing compact cards).
- [x] A finding can deep-link into evidence graph and then into control design ("Design observe-only control" CTA preserves `finding_id` / `device_id`).
- [x] `npm run build` passes in `ui/console`.
- [x] Existing evidence path backend tests still pass (`go vet` green; signature of `buildEvidencePath` unchanged).

## Dependencies

- WO-PROD-002
- WO-PROD-003
- WO-GROWTH-001 recommended

## Non-Goals

- No graph database.
- No complex canvas visualization.
- No enforcement behavior.

## Suggested Verification

- `cd backend/ingest && go vet ./internal/server/...`.
- `cd ui/console && npm run build`.
- Manual walkthrough with a complete and partial evidence path.
