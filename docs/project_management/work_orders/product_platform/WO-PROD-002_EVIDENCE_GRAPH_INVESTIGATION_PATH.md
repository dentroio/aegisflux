# WO-PROD-002: Evidence Graph Investigation Path

**Status:** Draft  
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

- Define evidence graph node types:
  - endpoint
  - user/session
  - process
  - parent process
  - command
  - network flow
  - DNS lookup
  - browser extension
  - local runtime
  - finding
  - detection pack
  - draft control
- Define edge types:
  - launched
  - resolved
  - connected
  - matched
  - observed_on
  - supports_control
- Add graph/path API for a finding or endpoint activity.
- Add compact UI path:
  - high-level path first
  - expandable raw evidence
  - confidence and missing-evidence indicators
- Add docs explaining relationship quality and limitations.

## Acceptance Criteria

- From a finding, an operator can see the most relevant process/network/DNS path without reading raw JSON.
- Missing evidence is explicit rather than hidden.
- Raw records remain available behind bounded detail.
- The graph path is stable enough to support future draft-control generation.
- Relevant backend tests pass.
- `npm run build` passes in `ui/console` if UI changes are made.

## Dependencies

- WO-API-001
- WO-PROD-001 recommended
- Existing visibility storage and query APIs

## Non-Goals

- No full graph database requirement.
- No complex graph visualization in the first slice.
- No enforcement or control staging.

## Suggested Verification

- Tests for path building with complete and partial evidence.
- UI route checks for finding/agent drill-in surfaces.

