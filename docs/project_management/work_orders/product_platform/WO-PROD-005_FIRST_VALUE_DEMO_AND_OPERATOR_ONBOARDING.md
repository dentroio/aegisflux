# WO-PROD-005: First-Value Demo and Operator Onboarding

**Status:** Draft  
**Phase:** Product Differentiation  
**Primary owner:** Product / UI / Docs  

## Goal

Build a focused first-value path that shows why AegisFlux matters in under five minutes: discover AI capability, inspect evidence, draft an observe-only control, and understand next steps.

## Why This Makes Aegis Stand Out

People use products that help them win quickly. AegisFlux needs a guided path that avoids raw telemetry overwhelm and tells the product story through a real operator workflow.

The platform should feel like a control-design assistant, not a pile of endpoint data.

## Scope

- Demo route or guided workflow inside the console.
- Seeded lab scenario docs.
- Operator-facing copy and empty states.
- Links into ABOM, evidence graph, and draft controls when available.

## Deliverables

- Define the first-value journey:
  1. Fleet has reporting agents.
  2. Operator sees AI capability inventory.
  3. Operator opens one endpoint.
  4. Operator sees the evidence path.
  5. Operator creates or reviews an observe-only draft control.
  6. Operator sees blast-radius/simulation context.
- Add a "Start here" or guided demo entry point in the console.
- Add polished empty states for missing lab data.
- Add docs for preparing demo data and lab agents.
- Add screenshots or checklist placeholders where appropriate.

## Acceptance Criteria

- A new operator can understand what AegisFlux does without reading architecture docs.
- The demo path uses meaningful labels, not raw telemetry categories.
- Empty states explain how to get value when data is missing.
- The path avoids long scrolling and right-side detail sprawl.
- `npm run build` passes in `ui/console` if UI changes are made.
- Documentation explains how to run the demo.

## Dependencies

- WO-PROD-001 recommended
- WO-PROD-002 recommended
- WO-PROD-003 recommended
- Current lab connectivity and agent reporting

## Non-Goals

- No fake claims about enforcement.
- No marketing landing page instead of product workflow.
- No production installer work.

## Suggested Verification

- UI build and route checks.
- Manual walkthrough using lab data.
- Docs review for clear operator language.

