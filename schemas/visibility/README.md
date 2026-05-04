# Aegis Visibility Schemas

This directory contains v1 JSON Schemas for host/workload visibility events emitted by Aegis agents.

These schemas are the contract for Phase 1 visibility and observability:

- Windows agent process, network, DNS, and detection events
- macOS agent heartbeat and future visibility events
- Backend ingest validation
- Future Clarion context graph mapping

## Event Envelope

All visibility events share the same top-level envelope:

- `schema_version`
- `event_id`
- `event_type`
- `timestamp_ms`
- `source`
- `device_id`
- `agent_id`
- `sensor_version`
- `sequence`
- `payload`

The current schema version is:

```text
visibility.v1
```

## Schemas

| Event Type | Schema |
|------------|--------|
| `aegis.agent.heartbeat` | [agent-heartbeat.schema.json](agent-heartbeat.schema.json) |
| `aegis.collector.status` | [collector-status.schema.json](collector-status.schema.json) |
| `aegis.process.started` | [process-started.schema.json](process-started.schema.json) |
| `aegis.process.ended` | [process-ended.schema.json](process-ended.schema.json) |
| `aegis.flow.started` | [flow-started.schema.json](flow-started.schema.json) |
| `aegis.flow.ended` | [flow-ended.schema.json](flow-ended.schema.json) |
| `aegis.dns.observed` | [dns-observed.schema.json](dns-observed.schema.json) |
| `aegis.browser_extension.observed` | [browser-extension-observed.schema.json](browser-extension-observed.schema.json) |
| `aegis.sase_component.observed` | [sase-component-observed.schema.json](sase-component-observed.schema.json) |
| `aegis.agent.detected` | [agent-detected.schema.json](agent-detected.schema.json) |
| `aegis.risk_finding.created` | [risk-finding-created.schema.json](risk-finding-created.schema.json) |

## Examples

Example fixtures live in [examples/](examples/).

## Design Notes

- These schemas are intentionally permissive where platform-specific fields may be unavailable.
- Inferred relationships should include a confidence or correlation method.
- Enforcement fields are excluded from Phase 1 except as non-binding recommendations in detection/risk findings.
- Agents should emit valid JSONL records that each validate against the relevant event schema.
