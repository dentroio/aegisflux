# WO-VIS-004: Visibility Event Schemas

**Status:** Complete  
**Phase:** Visibility and Observability  
**Primary owner:** Backend / Integration  

## Goal

Define versioned event schemas for the first AegisFlux visibility loop and keep them compatible with future Clarion context ingestion.

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
- JSON syntax verified with `jq` via `schemas/visibility/validate_examples.sh`
- Flow and finding schemas extended for `process_path`, `connection_state`, byte counters (optional), and taxonomy fields on detections/findings
- Full JSON Schema compile in CI for every example remains optional follow-up (ingest uses a separate legacy event schema for gRPC path)
