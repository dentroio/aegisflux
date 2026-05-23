# WO-GOV-001: Agent Governance, Memory, and Decision Ledger

**Status:** Planned  
**Phase:** AI-Native Leap / Govern  
**Primary owner:** AI Platform / Backend / Security  

## Goal

Create durable governance and memory primitives for AI agent work: what was known, what was recommended, who approved/rejected it, and why.

## Problem

Deep agents become risky if their reasoning, memory, and decisions are opaque. AegisFlux needs product memory and a decision ledger that are scoped, auditable, and useful without becoming unbounded chat history.

## Scope

- Agent memory model.
- Decision ledger.
- Approval/rejection records.
- Prompt/tool/run audit integration.
- Privacy-aware retention.

## Deliverables

- Define memory types: device, fleet, detection, control, run.
- Define decision ledger entries for recommendations, approvals, rejections, edits, superseded decisions, and expiry.
- Add APIs to append/query ledger entries.
- Add retention/privacy rules for stored prompts, redacted previews, tool outputs, and evidence snapshots.
- Add UI surface for decision history on agent runs, candidates, draft controls, audit bundles, and device detail.

## Acceptance Criteria

- Agent-generated recommendations are linked to run, evidence, and decision outcome.
- Operator decisions record actor, timestamp, rationale, and affected object.
- Memory is scoped and queryable; it is not a free-form global chat log.
- Privacy settings influence stored previews and retained context.

## Dependencies

- WO-AGENTS-001.
- WO-AGENTS-002.
- WO-AI-002.
- WO-PLAT-006.

## Non-Goals

- No unbounded vector memory.
- No hidden autonomous approvals.
- No storage of raw secrets.

## Suggested Verification

- Backend tests for ledger append/query and privacy redaction behavior.
- Manual accept/reject/edit workflow from one agent recommendation.
