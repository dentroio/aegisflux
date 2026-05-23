# WO-AGENTS-003: Endpoint Analyst Deep Agent

**Status:** Planned  
**Phase:** AI-Native Leap  
**Primary owner:** AI Platform / UI / Backend  

## Goal

Upgrade endpoint analysis from a single AI action into a deep investigation agent that can answer evidence-bound questions about a device, finding, or AI-capable activity.

## Problem

Operators need a teammate that can explain what changed, why it matters, what evidence exists, what is missing, and what to inspect next. Generic chat is too loose; the analyst must work from bounded endpoint context and cite evidence.

## Scope

- Endpoint detail and finding context.
- Evidence query tools from the agent harness.
- Evidence-bound output contract.
- UI surface for analysis runs and follow-up prompts.

## Deliverables

- Add Endpoint Analyst agent definition and allowed tools.
- Support initial question types:
  - explain AI activity on this endpoint,
  - summarize evidence for this finding,
  - compare endpoint to recent baseline,
  - identify missing evidence,
  - recommend next investigation step.
- Add run initiation from endpoint detail and finding/evidence path surfaces.
- Render cited evidence, confidence, missing evidence, and next step.
- Store analyst runs in the decision/run ledger.

## Acceptance Criteria

- Endpoint Analyst uses only bounded tools and selected device/finding context.
- Answers cite evidence and explicitly call out unknowns.
- Operator can open prior analyst runs for the same device/finding.
- No endpoint mutation, enforcement, or broad data dump is possible from this agent.

## Dependencies

- WO-AGENTS-001.
- WO-AGENTS-002.
- WO-GROWTH-002.
- WO-GROWTH-004.

## Non-Goals

- No generic chatbot.
- No multi-tenant enterprise search.
- No automatic control approval.

## Suggested Verification

- Backend tests for allowed tool scope.
- UI build and route smoke for endpoint detail.
- Manual analyst run on Linux and Windows lab devices.
