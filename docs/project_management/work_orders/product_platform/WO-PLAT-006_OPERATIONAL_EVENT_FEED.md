# WO-PLAT-006: Operational Event Feed

**Status:** Completed (May 2026 — lab slice implemented)
**Phase:** Product Platform  
**Primary owner:** Backend / UI  

## Goal

Create a first operational event feed so AegisFlux operators can see platform actions, status transitions, approvals, rollout events, AI activity, and integration exports in one auditable place.

## Scope

- Backend event model and append/read API.
- UI page under Operate > Operational Event Feed.
- Initial producers from existing platform events where practical.

## Deliverables

- Operational event record:
  - id
  - timestamp
  - type
  - actor
  - subject_type
  - subject_id
  - status
  - summary
  - details
  - related_device_id
  - related_agent_uid
  - related_pack_id
  - related_candidate_id
  - related_control_id
- API to append and list events.
- UI feed with filters for type, status, subject, device/agent, and time.
- Event producers for:
  - agent heartbeat/stale transitions where available.
  - detection candidate validation/approval/rejection/signing.
  - detection-pack rollout apply/reject/rollback status.
  - AI provider tests and future AI run/audit records.
  - draft control create/simulate/approve/reject when WO-CTRL-001 lands.
  - Clarion export attempts when WO-INT-001 lands.

## Acceptance Criteria

- Operator can open the feed from the persistent shell.
- Feed can show recent detection-pack and candidate lifecycle events.
- Feed gracefully handles no events.
- Events are append-only from the UI perspective.
- `npm run build` passes in `ui/console`.
- Relevant backend tests pass for the event model/API.

## Dependencies

- WO-PLAT-001
- WO-DET-002
- WO-DET-003
- WO-AI-001 for AI provider events
- WO-CTRL-001 for draft-control events
- WO-INT-001 for Clarion export events

## Non-Goals

- No SIEM export in the first slice.
- No immutable external audit store.
- No production retention policy beyond a simple lab-safe local store.

## Reference

- `docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md`
- Clarion operational/audit feed pattern: `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/Audit.tsx`

