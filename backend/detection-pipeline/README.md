# detection-pipeline (WO-DET-002 / WO-DET-003)

HTTP service for the **research item → candidate → validation → review → approved → signed pack** workflow (WO-DET-002) and **lab-only controller rollout + agent status** (WO-DET-003). Uses the `detection_pack.v1` JSON Schema from WO-DET-001. **Does not** enforce controls or auto-publish to production.

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `DETECTION_PIPELINE_HTTP_ADDR` | `:8089` | Listen address |
| `INGEST_URL` | `http://127.0.0.1:9090` | Base URL for `GET /v1/visibility/events` (lab telemetry) |
| `DETECTION_PIPELINE_DATA_PATH` | _(empty)_ | Optional JSON persistence path for pipeline state |
| `DETECTION_PACK_SCHEMA_PATH` | _(auto)_ | Override path to `detection-pack.v1.schema.json` (set in Docker image) |
| `DETECTION_PIPELINE_ED25519_SEED_HEX` | _(random)_ | 32-byte seed for reproducible signatures in CI |
| `DETECTION_ROLLOUT_LAB_ONLY` | `true` | When `false`, WO-DET-003 controller routes return 403 |
| `DETECTION_ROLLOUT_ALLOWED_PACK_IDS` | _(empty)_ | Optional comma list limiting which `pack_id` values are offered as “latest” / artifacts |
| `DETECTION_ROLLOUT_STALE_AFTER_MS` | `86400000` | Staleness window for rollout-status (`last_check_at`) |
| `DETECTION_PIPELINE_PUBLIC_BASE_URL` | `http://127.0.0.1:8089` | Base URL recorded in optional `aegis.detection_pack.status` visibility events |

## Main HTTP routes

- `POST/GET /v1/detection/research-items` — create / list research items  
- `POST/GET /v1/detection/candidates` — create (validates `proposed_rules` against pack schema) / list (`?status=`)  
- `GET /v1/detection/candidates/{id}` — candidate detail  
- `POST /v1/detection/candidates/{id}/validate` — fetch telemetry from ingest, evaluate rules, set `validating` → `ready_for_review` or `validation_failed`  
- `GET /v1/detection/candidates/{id}/validations` — validation runs  
- `POST /v1/detection/candidates/{id}/approve` — `ready_for_review` → `approved`  
- `POST /v1/detection/candidates/{id}/reject` — body `{"reason":"..."}`  
- `POST /v1/detection/candidates/{id}/sign` — `approved` → `signed`, stores `detection_pack.v1` JSON with Ed25519 signature  
- `GET /v1/detection/candidates/{id}/signed-pack` — raw signed pack JSON for that candidate  
- `GET /v1/detection/signed-packs` — list artifact metadata  
- `GET /v1/detection/signed-packs/{artifact_id}` — full signed pack JSON  
- `GET /v1/detection/signer-info` — public key id and dev public key material  

### WO-DET-003 (same port; lab rollout)

Also exposed at the same paths **without** the `/v1` prefix (e.g. `/detection-packs/latest`).

- `GET /v1/detection-packs/latest?os=linux|windows|macos&agent_version={semver}&pack_id={optional}` — newest **signed** pack compatible with OS/agent semver, observe-only, unexpired, allowlist-respecting.
- `GET /v1/detection-packs/{pack_id}/artifact?os=&agent_version=&version={optional}` — pack JSON body + `X-Content-SHA256` / signature headers.
- `GET /v1/detection-packs/{pack_id}/rollout-status` — per-agent summaries for that `pack_id` (includes computed staleness vs `last_check_at`).
- `GET|POST /v1/agents/{agent_uid}/detection-pack-status` — read/update agent rollout state (`applied`, `rejected`, `stale`, `incompatible`, `expired`, `rollback`, `not_checked`, …) with reason codes; POST may set `emit_visibility` to forward `aegis.detection_pack.status` to ingest.

## Fixtures

See `schemas/detection/fixtures/wo-det-002/`: example research payload, MCP/tool-bridge candidate (`candidate_mcp_tool_bridge.example.json`), and `lab-telemetry.json` (POST to ingest as a JSON array) for device `lab-mcp-01`.

## Docker

Built from repo root: `docker build -f backend/detection-pipeline/Dockerfile .`

Compose service `detection-pipeline` depends on `ingest`; set `INGEST_URL=http://ingest:9090`.
