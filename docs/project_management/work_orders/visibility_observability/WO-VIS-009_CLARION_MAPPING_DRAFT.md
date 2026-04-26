# WO-VIS-009: Clarion Mapping Draft

**Status:** Draft  
**Phase:** Visibility and Observability  
**Primary owner:** Integration / Clarion  

## Goal

Map Aegis visibility events to Clarion context objects so the projects can integrate cleanly later.

## Scope

Documentation and contract mapping only. No Clarion implementation is required in this work order.

## Mapping Targets

Aegis visibility events should map to Clarion objects:

- `device`
- `user`
- `network_session`
- `process`
- `application`
- `agent`
- `flow`
- `destination`
- `risk_finding`
- `policy_decision` future
- `enforcement_action` future

## Deliverables

- Field-level mapping from Aegis v1 schemas to Clarion context objects
- ID strategy:
  - `device_id`
  - `agent_id`
  - `process_guid`
  - `flow_id`
  - Clarion endpoint/device IDs
- Confidence model for inferred relationships
- Clarion ingestion assumptions and open questions
- Boundary statement confirming Aegis emits evidence and Clarion owns cross-domain policy decisions

## Acceptance Criteria

- Clarion can identify which Aegis fields are needed for context graph ingestion.
- Aegis can identify which fields must remain stable for future Clarion integration.
- Unknown or unavailable fields are explicitly marked.
- No direct database coupling is introduced.

## Dependencies

- WO-VIS-004

## Risks

- Clarion’s existing domain ownership model may need extension for host/process objects.
- Aegis may produce evidence faster than Clarion can ingest it. Future buffering/aggregation may be needed.

## Completion Evidence

- Mapping document
- Reviewed open questions list
- Recommended v1 integration path
