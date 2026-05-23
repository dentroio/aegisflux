# WO-DEMO-002: Autonomous Demo Scenario Generator

**Status:** Planned  
**Phase:** AI-Native Leap / Demo  
**Primary owner:** Product / QA / AI Platform  

## Goal

Build a scenario generator that creates and validates realistic first-value demos for AI-capable endpoint activity, detection adaptation, and safe control design.

## Problem

The product needs compelling, repeatable demos. Manually crafting scenarios is slow and brittle. AegisFlux should be able to generate lab-safe scenarios, expected evidence, expected detections, and validation checks.

## Scope

- Lab-safe sample scenario definitions.
- Generated event fixtures and optional endpoint scripts.
- Expected console/API evidence.
- Validation against e2e smoke and UI routes.

## Deliverables

- Define scenario schema: story, endpoint type, generated signals, expected findings, expected ABOM items, expected drift, expected control draft, validation checks.
- Add initial scenarios:
  - local MCP/tool bridge observed,
  - new AI CLI reaches model gateway,
  - browser AI extension plus unusual upload destination,
  - stale agent with valid previous detection-pack status.
- Add generator or templating workflow for fixtures.
- Link scenario output to e2e validation and demo docs.

## Acceptance Criteria

- A generated scenario can seed or replay evidence without unsafe endpoint behavior.
- Expected API/UI observations are documented.
- Scenario validation can fail clearly when evidence, detection, or UI surfaces regress.
- Demo text stays grounded in actual generated evidence.

## Dependencies

- WO-OPS-001.
- WO-ADAPT-002 recommended.
- WO-FLEET-001 recommended.

## Non-Goals

- No malware simulation.
- No external attack tooling.
- No uncontrolled endpoint mutation.

## Suggested Verification

- Generate one scenario fixture.
- Replay it through ingest or fixture harness.
- Confirm expected dashboard/agent/inventory/detection/control surfaces update.
