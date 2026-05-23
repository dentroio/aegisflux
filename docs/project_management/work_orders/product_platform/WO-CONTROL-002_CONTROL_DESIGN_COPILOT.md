# WO-CONTROL-002: Control Design Copilot

**Status:** Planned  
**Phase:** AI-Native Leap / Control Design  
**Primary owner:** AI Platform / Control / UI  

## Goal

Add a governed Control Design Copilot that converts evidence-bound findings into observe-only control proposals with blast radius, rollback, and approval packet details.

## Problem

The product has draft controls and simulation primitives, but operators need help turning nuanced endpoint evidence into a safe, reviewable control design. The copilot must make recommendations without enabling blocking behavior.

## Scope

- Finding/evidence path to draft-control proposal.
- Control simulation and rollback notes.
- Approval packet generation.
- Audit-mode recommendation only.

## Deliverables

- Add Control Designer agent definition and allowed tools.
- Produce proposal fields:
  - title,
  - scope,
  - evidence refs,
  - expected match telemetry,
  - blast radius,
  - breakage risks,
  - rollback notes,
  - expiration/review date,
  - approval questions.
- Add UI action from finding/evidence path and draft-control detail.
- Store proposal in decision ledger and draft-control history.

## Acceptance Criteria

- Copilot output validates against evidence-bound reasoning contract.
- Proposed controls remain observe-only unless separately staged as audit-mode.
- UI makes clear what is AI-drafted versus operator-approved.
- Operator can accept, edit, reject, or request more evidence.

## Dependencies

- WO-AGENTS-001.
- WO-AGENTS-002.
- WO-GROWTH-003.
- WO-GROWTH-007.

## Non-Goals

- No blocking or deny behavior.
- No direct policy publish.
- No hidden approval.

## Suggested Verification

- Backend tests for proposal validation.
- Manual finding-to-control copilot flow.
- Confirm operational/decision events record accept/edit/reject.
