# schemas

JSON Schemas for AegisFlux events, findings, actions, and visibility telemetry. Validate at ingest and in tests.

## Detection packs

Signed, observe-only detection intelligence for endpoint agents lives in [detection/](detection/). The active contract is `detection-pack.v1.schema.json` (`schema_version: detection_pack.v1`). Example packs are under [detection/examples/](detection/examples/). Lab fixtures for the **WO-DET-002** candidate pipeline (MCP / local model / tool-bridge) live under [detection/fixtures/wo-det-002/](detection/fixtures/wo-det-002/). Go tests in `backend/detectionpack-schema` compile this schema with `github.com/santhosh-tekuri/jsonschema/v5` (draft 2020-12). The `detection-pipeline` HTTP service (`backend/detection-pipeline`) implements research → candidate → validation → approval → signing, plus **WO-DET-003** lab-only latest-pack / artifact / agent-status controller routes (still no production rollout).

## Visibility

Phase 1 host/workload visibility schemas live in [visibility/](visibility/). These define the v1 event contract for Windows and macOS agent heartbeat, process, flow, DNS, AI-agent detection, and risk-finding events.
