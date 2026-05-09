# WO-PROD-002: Evidence Graph Investigation Path

**Status:** Done  
**Phase:** Product Differentiation  
**Primary owner:** Backend / UI  

## Goal

Create an evidence graph that connects finding, process, parent process, command line, flow, DNS, endpoint, detection pack, and draft control context into one explainable path.

## Why This Makes Aegis Stand Out

Security tools often show many events and ask the operator to mentally stitch them together. AegisFlux should do that work. The product should say: this finding came from this process, launched by this parent, contacting this destination, resolved through this domain, matched by this detection, and supports this proposed control.

Trust comes from linkage.

## Scope

- Read-only investigation relationship model.
- Backend path-building from existing visibility records.
- UI investigation path component.
- Endpoint and finding drill-in integration.

## Deliverables

- Defined evidence graph node types in `backend/ingest/internal/server/evidence_path_query.go`:
  - `endpoint`, `parent_process`, `process`, `flow`, `dns`, `finding`, `detection_pack`, `draft_control` (forward-compatible: `user_session`, `browser_extension`, `local_runtime`).
- Defined edge types: `launched`, `resolved`, `connected`, `matched`, `observed_on`, `supports_control`.
- New backend API: `GET /v1/visibility/evidence-path` with filters `finding_id`, `device_id`, `process_guid`, `agent_id`. Response includes:
  - subject (finding | process | endpoint),
  - curated `nodes` and `edges`,
  - `missing_evidence` callouts (e.g. `parent_process`, `dns`, `detection_pack`),
  - `confidence_overall` ranked from per-node confidences,
  - `summary` one-liner ready to read,
  - bounded raw evidence (`raw_processes`, `raw_flows`, `raw_dns`, `raw_findings`, plus the existing draft-control synthesizer).
- Reusable `EvidenceGraphPanel` component in the console with:
  - inputs for finding id, device id, and process GUID,
  - high-level path cards (one per node), edge labels, confidence dots,
  - amber missing-evidence callouts and per-node "missing" styling,
  - bounded raw JSON view behind a collapsible "Raw evidence" section,
  - graceful empty state.
- New `/analyze/evidence` route that uses the panel (replacing the placeholder).
- Endpoint detail page: new `Evidence Path` tab that auto-loads the path for the device's most recent finding.
- Dashboard "Next best actions" gains a deep link into the evidence path.
- Backend tests in `evidence_path_query_test.go`:
  - full path with finding + process + flow + DNS + draft control;
  - partial evidence (only finding) marks missing nodes and downgrades confidence;
  - empty inputs return no nodes and low confidence.

## Acceptance Criteria

- [x] From a finding, an operator can see the most relevant process/network/DNS path without reading raw JSON (path cards plus summary).
- [x] Missing evidence is explicit rather than hidden (`missing_evidence` field + amber UI callouts + node-level missing badge).
- [x] Raw records remain available behind bounded detail (collapsible "Raw evidence" with `JSON.stringify` blocks limited to the first six rows per category).
- [x] The graph path is stable enough to support future draft-control generation (re-uses `buildDraftControls` and surfaces them as `supports_control` edges).
- [x] Relevant backend tests added.
- [x] `npm run build` passes in `ui/console`.

## Dependencies

- WO-API-001
- WO-PROD-001 recommended
- Existing visibility storage and query APIs

## Non-Goals

- No full graph database requirement.
- No complex graph visualization in the first slice.
- No enforcement or control staging.

## Suggested Verification

- `cd backend/ingest && go test ./internal/server -run TestBuildEvidencePath`.
- `cd ui/console && npm run build`.
- Manual walkthrough: open `/analyze/evidence`, paste a finding id (or device id) from a lab agent, click `Build path`, confirm the path renders with confidence dots and missing-evidence callouts. Then open Agents → endpoint → `Evidence Path` tab and confirm it auto-loads.
