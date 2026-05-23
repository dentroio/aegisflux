# WO-ADAPT-002: Detection Candidate Simulation Harness

**Status:** Planned  
**Phase:** AI-Native Leap / Adapt  
**Primary owner:** Detection / Backend / QA  

## Goal

Build a simulation harness that replays candidate detection packs against fixtures and historical lab evidence before approval.

## Problem

Adaptive detection is only trustworthy if candidate packs can be tested before rollout. AegisFlux needs repeatable simulation results that show coverage, false-positive risk, missing evidence, resource estimates, and rollout readiness.

## Scope

- Detection-pack candidates.
- Fixture and historical evidence replay.
- Validation agent/tool integration.
- Simulation result storage and UI review.

## Deliverables

- Define simulation result schema: candidate id/version, fixture set, matched events, unmatched expected events, false-positive notes, missing evidence, estimated endpoint cost, recommendation.
- Add simulation tool callable by Pack Validation or Simulation Agent.
- Add fixture set for AI tool bridge/MCP-style behavior and baseline normal developer activity.
- Store simulation results on candidates.
- Add UI/API view for simulation history and approval blockers.

## Acceptance Criteria

- Candidate pack cannot be marked ready for approval without a simulation result.
- Simulation distinguishes no matches, expected matches, broad/noisy matches, and missing evidence.
- Results are auditable and linked to candidate version and fixture set.
- Existing detection-pack rollout remains unchanged.

## Dependencies

- WO-ADAPT-001.
- WO-DET-001 through WO-DET-003.
- WO-OPS-007 recommended for replay/load baselines.

## Non-Goals

- No production enforcement.
- No arbitrary code in detection packs.
- No guarantee that lab fixtures represent every customer environment.

## Suggested Verification

- Unit tests for simulation classification.
- Replay fixture pack against fixture data.
- Manual review of simulation result from candidate detail.
