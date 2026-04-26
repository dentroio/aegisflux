# WO-VIS-004: Visibility Event Schemas

**Status:** Initial contract complete  
**Phase:** Visibility and Observability  
**Primary owner:** Backend / Integration  

## Goal

Define versioned event schemas for the first Aegis visibility loop and keep them compatible with future Clarion context ingestion.

## Scope

Schemas only. This work order defines contracts; collectors and backends implement them in other work orders.

## Required Schemas

- `aegis.agent.heartbeat`
- `aegis.process.started`
- `aegis.process.ended`
- `aegis.flow.started`
- `aegis.flow.ended`
- `aegis.dns.observed`
- `aegis.agent.detected`
- `aegis.risk_finding.created`

## Common Envelope

Every event should include:

- `schema_version`
- `event_id`
- `event_type`
- `timestamp`
- `source`
- `device_id`
- `agent_id`
- `sensor_version`
- `sequence`
- `collected_at`
- `sent_at`
- `signature` or signature reference, once signed telemetry is enabled

## Design Requirements

- Use stable IDs where possible.
- Support partial fields and unknown values without dropping events.
- Include confidence and evidence fields for inferred relationships.
- Preserve raw observed values alongside normalized values.
- Avoid Clarion-specific database assumptions.
- Keep schemas suitable for JSON, NATS, HTTP ingest, and eventual Clarion context graph mapping.

## Deliverables

- JSON Schema files or Markdown schema specs - created in `../aegisflux/schemas/visibility/`
- Example valid events for each schema - created in `../aegisflux/schemas/visibility/examples/`
- Validation test cases
- Versioning rules
- Field mapping notes to Clarion objects:
  - `device`
  - `user`
  - `process`
  - `application`
  - `agent`
  - `flow`
  - `destination`
  - `risk_finding`

## Acceptance Criteria

- Schemas cover process, flow, DNS, detection, and risk finding events.
- Example events validate successfully.
- Backend and agent teams can implement against the same contract.
- Clarion mapping is documented at the field level for v1 fields.

## Dependencies

- None

## Risks

- Over-modeling too early can slow implementation. Keep v1 small and extensible.
- Clarion may need fields not available from the first Windows collector. Mark unavailable fields as optional.

## Completion Evidence

- Schema files created under `aegisflux/schemas/visibility/`
- Example event fixtures created under `aegisflux/schemas/visibility/examples/`
- JSON syntax verified with `jq`
- Remaining: add automated JSON Schema validation in backend/CI once validator tooling is selected
- Remaining: short review sign-off from agent and backend owners
