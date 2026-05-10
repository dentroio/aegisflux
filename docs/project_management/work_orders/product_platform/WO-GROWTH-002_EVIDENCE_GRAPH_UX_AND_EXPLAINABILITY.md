# WO-GROWTH-002: Evidence Graph UX and Explainability

**Status:** Draft  
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

- Add a plain-language evidence summary block:
  - "What happened"
  - "Why it matters"
  - "What we know"
  - "What is missing"
  - "Recommended next step"
- Replace raw node labels where possible with operator language.
- Add confidence reasons, not just confidence values.
- Add compact timeline/path ordering.
- Add "Open related ABOM item" where evidence maps to ABOM.
- Add "Design observe-only control" CTA where evidence can seed a draft.
- Add bounded raw evidence drawer only for operators who need details.

## Acceptance Criteria

- Evidence path can be understood without reading JSON.
- Missing evidence has actionable explanations.
- Graph/path UI avoids long scroll by default.
- A finding can deep-link into evidence graph and then into control design.
- `npm run build` passes in `ui/console`.
- Existing evidence path backend tests still pass.

## Dependencies

- WO-PROD-002
- WO-PROD-003
- WO-GROWTH-001 recommended

## Non-Goals

- No graph database.
- No complex canvas visualization.
- No enforcement behavior.

## Suggested Verification

- `npm run build` in `ui/console`.
- Existing evidence path backend tests.
- Manual walkthrough with a complete and partial evidence path.

