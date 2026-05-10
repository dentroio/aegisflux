# WO-GROWTH-005: Adaptive Detection Workflow Maturity

**Status:** Done  
**Phase:** Product Growth / Adapt  
**Primary owner:** Detection / AI Platform / UI  

## Goal

Mature the research feed into an end-to-end adaptive detection workflow: opportunity, candidate, simulation, governance, signed pack, rollout visibility, and retirement.

## Why This Matters

The "Adapt" pillar is only compelling if the workflow feels governed and repeatable. Aegis should show how new AI-agent behavior becomes safe observe-only detection without rebuilding endpoint agents.

## Scope

- Research opportunity lifecycle.
- Candidate quality gates.
- Simulation result display.
- Pack release/retirement governance.
- Rollout visibility cross-links.

## Deliverables

- Add quality gates:
  - required evidence fields
  - expected false-positive notes
  - simulation match count
  - expiration date
  - reviewer/approval notes
  - rollback/deprecation notes
- Link research opportunity -> candidate -> signed pack -> rollout status.
- Add detection workflow status board.
- Add candidate simulation summary in the UI.
- Add pack retirement/deprecation path for stale detections.
- Emit operational events for promote, approve, sign, deploy, retire.

## Acceptance Criteria

- Operator can follow a detection from research opportunity to rollout status.
- Candidate cannot appear release-ready without evidence requirements and simulation summary.
- Signed pack status links back to research rationale.
- Retired/deprecated packs are visible and auditable.
- Relevant backend tests pass.
- `npm run build` passes in `ui/console`.

## Dependencies

- WO-PROD-004
- WO-DET-002
- WO-DET-003
- WO-PLAT-004
- WO-PLAT-006

## Non-Goals

- No automatic external internet research unless separately approved.
- No automatic endpoint enforcement.
- No replacing signed-pack trust model.

## Suggested Verification

- Backend tests for lifecycle transitions.
- `npm run build` in `ui/console`.
- Manual walkthrough from research feed to detection pack rollout.

## Implementation Notes (Done)

### Backend (`backend/actions-api/internal/api`)

- Extended `PlatformData` to include a `Candidates []DetectionCandidate` slice and added `LinkedCandidateID` to `ResearchItem`.
- New types in `platform_state.go`:
  - `DetectionCandidate` (id, source research id, status, pack_id/version, rollout status, operator/reviewer notes, expires_at, rollback plan, retirement reason, rule, simulations, history, gate, timestamps).
  - `DetectionCandidateGate` (required evidence, expected false positives, has_simulation/reviewer_notes/expiration/rollback flags, missing_fields).
  - `DetectionCandidateSimulation` (deterministic match counts, devices, top indicators, window, confidence, notes).
  - `DetectionCandidateEvent` (history entry with action, from/to status, note, timestamp).
- New `detection_candidates.go` implementing `handleDetectionCandidatesCollection` (GET/POST), `handleDetectionCandidateItem` (GET/PATCH plus `/simulate` and `/retire` sub-paths). Lifecycle constants and `canTransition` enforce: `candidate_new -> simulated -> reviewed -> signed -> deployed`, with `retired` reachable from any non-retired status. `simulateDetectionCandidate` builds deterministic simulation results so demo runs are reproducible. `recomputeCandidateGate` flags missing evidence/simulation/reviewer notes/expiration/rollback before signing; signing returns `409` with the missing fields when the gate is open. Each transition appends a `DetectionCandidateEvent` to the candidate history and emits the matching operational event via the existing event ledger (`detection_candidate.created/updated/simulated/signed/deployed/retired`).
- Updated `research_feed.go` so promoting a research item also creates a linked `DetectionCandidate`, recomputes its quality gate, sets `LinkedCandidateID` on the research record, and emits an enriched `research.promoted` event with the candidate id.
- Registered routes in `platform_routes.go`:
  - `GET/POST /platform/detection-candidates`
  - `GET/PATCH/POST /platform/detection-candidates/{id}` (with `/simulate` and `/retire` sub-paths).
- Tests in `detection_candidates_test.go` cover `canTransition`, `recomputeCandidateGate` (flags missing fields and clears them when satisfied), deterministic `buildCandidateSimulation`, and the `actionForCandidateChange` mapping.

### Frontend (`ui/console`)

- `components/ResearchFeedPanel.tsx` now loads `/api/actions/platform/detection-candidates` alongside the research feed, exposes the `DetectionCandidate` type, and renders a new `DetectionWorkflowBoard` section with one column per stage (`candidate_new`, `simulated`, `reviewed`, `signed`, `deployed`, `retired`).
- Each card surfaces pack id, rollout status, latest simulation summary, and any open quality-gate gaps. Action buttons (`Simulate`, `Mark reviewed`, `Sign`, `Mark deployed`, `Retire`) call the new actions-api endpoints; the `Sign` button is disabled with a tooltip listing the missing gate fields whenever the candidate is not release-ready.
- A `CandidateDetailContent` modal shows the full rule, quality gate, latest simulation, history timeline, and an explicit observe-only disclaimer reinforcing that the workflow does not change endpoint enforcement state.
- `npm run build` (Next.js 14) passes; no new lints introduced. Backend `go vet ./...` and the new candidate unit tests pass; the broader `go test ./...` run still hits the local `_SecTrustCopyCertificateChain` linker issue noted in earlier WOs and is deferred to CI.

### Acceptance Criteria

- Operator can follow a detection from research opportunity to rollout status: research item now records `linked_candidate_id`, and the workflow board surfaces every downstream stage.
- Candidate cannot appear release-ready without evidence requirements and simulation summary: signing is gated on `quality_gate.missing_fields` being empty, both in API and UI.
- Signed pack status links back to research rationale: `DetectionCandidate.source_research_id` is populated on creation and shown in the candidate detail modal.
- Retired/deprecated packs are visible and auditable: `retired` column on the board, `retirement_reason` captured in history, and a dedicated `detection_candidate.retired` operational event.
- Relevant backend tests pass (`go vet ./...`, targeted candidate tests). `npm run build` passes in `ui/console`.

