# WO-VIS-009: Clarion Mapping Draft

**Status:** Complete (v1 field mapping documented)  
**Phase:** Visibility and Observability  
**Primary owner:** Integration / Clarion  

## Goal

Map AegisFlux visibility events to Clarion context objects so the projects can integrate cleanly later.

## Scope

Documentation and contract mapping only. No Clarion implementation is required in this work order.

## Mapping Targets

AegisFlux visibility events should map to Clarion objects:

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

- Field-level mapping from AegisFlux v1 schemas to Clarion context objects
- ID strategy:
  - `device_id`
  - `agent_id`
  - `process_guid`
  - `flow_id`
  - Clarion endpoint/device IDs
- Confidence model for inferred relationships
- Clarion ingestion assumptions and open questions
- Boundary statement confirming AegisFlux emits evidence and Clarion owns cross-domain policy decisions

## Acceptance Criteria

- Clarion can identify which AegisFlux fields are needed for context graph ingestion.
- AegisFlux can identify which fields must remain stable for future Clarion integration.
- Unknown or unavailable fields are explicitly marked.
- No direct database coupling is introduced.

## Dependencies

- WO-VIS-004

## Risks

- Clarion’s existing domain ownership model may need extension for host/process objects.
- AegisFlux may produce evidence faster than Clarion can ingest it. Future buffering/aggregation may be needed.

## Completion Evidence

- Mapping document
- Reviewed open questions list
- Recommended v1 integration path

---

## v1 field mapping (AegisFlux → Clarion context)

**Boundary:** AegisFlux emits **evidence** (observations, findings, recommended non-blocking actions). Clarion owns **policy decisions**, cross-domain correlation, and enterprise identity resolution. No shared database; integrate via documented IDs and HTTP/event contracts.

### Stable identifiers (do not rename without version bump)

| AegisFlux field | Clarion object | Notes |
|-----------------|----------------|-------|
| `device_id` (envelope) | `device.primary_key` / endpoint external id | Map Clarion device registry ID in the integration layer. |
| `agent_id` (envelope) | `agent.instance_id` | Per-install agent identity. |
| `event_id` (envelope) | `evidence_event.id` | Deduplication and audit trail. |
| `payload.process_guid` | `process.instance_id` | Synthetic stable instance: `{device_id}:{start_time}:{pid}`. |
| `payload.flow_id` | `flow.id` / `network_session.correlation_key` | Collector-derived; may collide across reboots if PID space reuses—Clarion should treat as **session-scoped**. |
| `payload.finding_id` | `risk_finding.id` | Finding record key. |
| `payload.detection_id` | `risk_finding.source_detection_id` | Links finding to `aegis.agent.detected`. |

### Envelope (all `visibility.v1` events)

| AegisFlux field | Clarion mapping |
|-----------------|-----------------|
| `schema_version` | Contract version for ingest adapters. |
| `event_type` | Routing key → context object factory. |
| `timestamp_ms` | Observation time (UTC epoch ms). |
| `source` | `evidence.source_system` (e.g. `aegis-windows-agent`). |
| `sensor_version` | `agent.sensor_version`. |
| `sequence` | Ordering hint within agent batch (best-effort). |

### `aegis.process.started` / `aegis.process.ended` → `process`, `user`, `application`

| Payload field | Clarion |
|---------------|---------|
| `pid`, `ppid` | `process.pid`, `process.parent_pid` (ephemeral; pair with `process_guid`). |
| `name`, `path` | `application.name`, `application.binary_path`. |
| `command_line` | `process.command_line` (policy: redact at edge). |
| `user`, `logon_session_id` | `user.account`, `user.session_id` (may be null on early collectors). |
| `integrity_level`, `publisher`, `sha256` | Optional trust signals on `process` / `application`. |
| `collection_method` | `evidence.attribution_method`. |

### `aegis.flow.started` / `aegis.flow.ended` → `flow`, `network_session`, `destination`

| Payload field | Clarion |
|---------------|---------|
| `protocol`, `direction`, `local_*`, `remote_*` | `network_session` L4 tuple; `destination.address`. |
| `process_guid`, `pid`, `process_name`, `process_path` | Link to `process` / `application`. |
| `remote_hostname` | `destination.hostname` (correlated). |
| `connection_state` | `flow.state` (snapshot). |
| `bytes_sent`, `bytes_received` | `flow.bytes` when present (often null in lab netstat path). |
| `attribution_method`, `attribution_confidence` | `evidence.correlation_method`, `confidence` (0–1). |

### `aegis.dns.observed` → `destination`, optional `process`

| Payload field | Clarion |
|---------------|---------|
| `query`, `answers`, `query_type` | `dns.query`, `dns.answers`. |
| `resolver` | Resolver identifier or browser-history provenance string. |
| `pid`, `process_guid`, `correlation_method`, `correlation_confidence` | Optional process link when inferred (e.g. flow IP match). |

### `aegis.agent.detected` / `aegis.risk_finding.created` → `risk_finding`

| Payload field | Clarion |
|---------------|---------|
| `classification`, `application_category`, `risk_signal` | Finding taxonomy dimensions. |
| `agent_likelihood`, `confidence`, `risk_score` | Scoring for analyst UI. |
| `detected_patterns` | Structured tags. |
| `evidence[]` | Embedded evidence items (`type`, `value`, `confidence`). |
| `recommended_action` | `monitor` / `review` / `policy_candidate` (observe-only in Phase 1). |

### Confidence model (v1)

- **Direct attribution:** PID match from netstat or explicit collector path → confidence **~0.7–0.95** depending on method.
- **Inferred DNS ↔ flow:** IP match in batch → `aegis.dns.flow_remote_ip_match`, capped (~0.94).
- **Browser AI:** Flow + DNS correlation raises confidence; DNS-only lowers confidence and must be labeled in evidence (implemented in agent detection copy).

### Open questions for Clarion

1. Canonical **device** key when Clarion already has MDM/EDR identity—merge table vs Aegis-native ID.
2. Retention and **PII** policy for `command_line` and file paths in multi-tenant Clarion.
3. Rate of visibility events vs Clarion graph ingest—need batching or rollup windows?
4. Mapping **dynamic_pack** findings vs native heuristics in unified `risk_finding` schema.

### Recommended v1 integration path

1. Ingest `visibility.v1` JSONL/HTTP as **immutable evidence events** into a Clarion adapter queue.
2. Normalize to Clarion `risk_finding` + `process` + `flow` vertices using the table above.
3. Defer `policy_decision` / `enforcement_action` until Aegis enforcement WO lands; keep fields reserved.
