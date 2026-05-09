# WO-PROD-004: AI Research Feed and Detection Opportunities

**Status:** Draft  
**Phase:** Product Differentiation  
**Primary owner:** AI Platform / Detection / UI  

## Goal

Create a research feed that turns new AI-agent ecosystem intelligence into detection opportunities, candidate packs, and operator-visible rationale.

## Why This Makes Aegis Stand Out

AI tooling changes faster than static signatures. AegisFlux should make adaptation visible: new AI tools and protocols become reviewed detection opportunities, not invisible vendor updates.

This shows the "Adapt" part of Observe. Adapt. Enforce.

## Scope

- Research opportunity model.
- Manual or seeded research entries in the first slice.
- Link opportunities to detection candidates.
- UI queue for review and promotion.

## Deliverables

- Define research opportunity fields:
  - title
  - source
  - tool/protocol/vendor
  - observed indicators
  - capability class
  - risk hypothesis
  - detection idea
  - confidence
  - status
  - related candidate pack id
- Add API/storage for opportunities.
- Add UI page under Analyze or Configure:
  - opportunity queue
  - detail view
  - promote to detection candidate
  - mark reviewed / rejected
- Seed examples:
  - MCP local server
  - coding agent CLI
  - browser automation extension
  - local model runtime
- Emit operational events for promotion/rejection.

## Acceptance Criteria

- Operator can review detection opportunities separately from signed packs.
- Opportunity detail explains why the signal might matter.
- Opportunity can become a detection candidate or be rejected.
- The flow remains governed and observe-only.
- Relevant backend tests pass.
- `npm run build` passes in `ui/console` if UI changes are made.

## Dependencies

- WO-DET-002
- WO-AI-002
- WO-PLAT-006

## Non-Goals

- No autonomous internet research in the first slice unless separately approved.
- No automatic signing or deployment of detection packs.
- No endpoint-agent changes required.

## Suggested Verification

- Backend tests for opportunity lifecycle.
- UI route checks for opportunity queue.
- Existing detection candidate tests.

