# WO-DET-003: Detection Pack Controller Rollout and Agent Status

**Status:** Controller complete; Analyze UI pending
**Phase:** Product Platform  
**Primary owner:** Backend / Agent / UI  

## Goal

Create the first controller-managed rollout path for approved detection packs so Windows and Linux agents can discover, verify, cache, and report detection-pack status without endpoint rebuilds.

This work order connects the output of `WO-DET-002` to the endpoint agents, but stays observe-only and lab-only. It must not introduce production rollout, autonomous publishing, or enforcement coupling.

## Scope

- Latest approved detection-pack discovery API.
- Pack artifact retrieval API.
- Agent detection-pack status reporting.
- Controller-side rollout state by agent.
- UI/API visibility for active, stale, rejected, incompatible, and expired packs.
- Lab-only rollout policy.

## Deliverables

- Controller/API endpoints:
  - `GET /detection-packs/latest?os={os}&agent_version={version}`
  - `GET /detection-packs/{pack_id}/artifact`
  - `POST /agents/{agent_uid}/detection-pack-status`
  - `GET /agents/{agent_uid}/detection-pack-status`
  - `GET /detection-packs/{pack_id}/rollout-status`
- Agent status model:
  - `agent_uid`
  - `active_pack_id`
  - `active_pack_version`
  - `last_check_at`
  - `last_applied_at`
  - `last_rejected_at`
  - `last_rejected_pack_id`
  - `last_rejected_reason`
  - `signature_status`
  - `hash_status`
  - `schema_status`
  - `compatibility_status`
  - `previous_pack_id`
  - `previous_pack_version`
- Rollout policy model:
  - lab-only scope.
  - observe-only packs only.
  - OS and minimum-agent-version compatibility.
  - expiration handling.
  - rollback to previous cached pack.
- UI updates under Analyze > Detection Packs:
  - active pack by agent.
  - agents behind latest pack.
  - rejected packs and reasons.
  - incompatible agents.
  - stale check-in status.
- Fixture rollout from the first approved lab pack created by `WO-DET-002`.

## Acceptance Criteria

- Controller can return the latest approved compatible pack for Linux and Windows agents.
- Controller refuses to serve unsigned, expired, rejected, or non-observe-only packs.
- Agents can report pack status without applying enforcement.
- Status records distinguish not checked, applied, rejected, stale, expired, incompatible, and rollback states.
- UI/API can show which agents are on which pack version.
- Rollout remains separate from candidate approval and endpoint enforcement.
- Previous pack metadata is preserved for rollback.

## Dependencies

- `WO-DET-001`
- `WO-DET-002`
- Real lab agent heartbeat/registration from Linux and Windows agents.

## Non-Goals

- No production customer rollout.
- No automatic publishing from candidate approval.
- No endpoint enforcement.
- No arbitrary code execution in packs.
- No external LLM calls from endpoint agents.

## Security Notes

- Pack retrieval must include hash/signature metadata.
- Agents must reject packs that fail signature, hash, schema, OS, version, mode, or expiration checks.
- Rejection reason must be reported to the controller for audit and troubleshooting.
- Rollback uses a previously verified cached pack only.

## Implementation notes

- **Service:** `backend/detection-pipeline` (same process as WO-DET-002). Lab rollout is gated with `DETECTION_ROLLOUT_LAB_ONLY` (default `true`; set to `false` to disable controller routes outside lab).
- **Routes:** `GET /v1/detection-packs/latest`, `GET /v1/detection-packs/{pack_id}/artifact`, `GET /v1/detection-packs/{pack_id}/rollout-status`, `GET|POST /v1/agents/{agent_uid}/detection-pack-status` (WO paths without `/v1` are also registered). Artifact responses include `X-Content-SHA256`, `X-Signature-Key-Id`, and `X-Signature-Algorithm`.
- **Optional allowlist:** `DETECTION_ROLLOUT_ALLOWED_PACK_IDS` (comma-separated `pack_id` values). **Stale threshold:** `DETECTION_ROLLOUT_STALE_AFTER_MS` (default 24h).
- **Visibility:** `schemas/visibility/detection-pack-status.schema.json` for `aegis.detection_pack.status`. POST status with `"emit_visibility": true` forwards one event to ingest when `device_id` is set.
- **Public URL for events:** `DETECTION_PIPELINE_PUBLIC_BASE_URL` (used in `controller_endpoint` when emitting status).

## Implementation Notes

- Controller/latest-pack discovery and artifact retrieval are implemented in `backend/detection-pipeline`.
- Agent pack status `GET|POST` routes are implemented and stored by agent UID.
- Rollout status summaries compute applied, rejected, incompatible, expired, and stale counts.
- Artifact responses include content hash and signature headers.
- `aegis.detection_pack.status` visibility schema exists and can be emitted by the controller.
- Agent list and device drill-in expose detection-pack rollout/status telemetry through the existing console surfaces.

## Remaining Work

- Dedicated Analyze > Detection Packs rollout page is still pending.
- Current controller path is lab-only and intentionally observe-only.

## Verification

- `go test ./...` in `backend/detection-pipeline`
- `./scripts/lab/smoke-detection-rollout.sh`
