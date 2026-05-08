# WO-DET-001: Detection Pack Schema and Local Evaluator Contract

**Status:** Complete
**Phase:** Product Platform  
**Primary owner:** Backend / Agent  

## Goal

Define how dynamic AI detections are shipped safely to endpoint agents without rebuilding agents or running arbitrary code.

## Scope

- Detection pack schema.
- Signing/versioning metadata.
- Local evaluator contract for Windows and Linux agents.

## Deliverables

- JSON schema for detection packs.
- Pack metadata:
  - id
  - version
  - created_at
  - author/source
  - minimum agent version
  - supported OS
  - signature
  - rollback version
- Rule model:
  - evidence selectors
  - process markers
  - DNS/domain markers
  - flow markers
  - browser/extension markers
  - SASE/SSE markers
  - confidence scoring
  - risk scoring
  - false-positive notes
- Agent evaluator contract with CPU/memory/time limits.
- Fixture pack for current AI markers.

## Acceptance Criteria

- Pack schema validates with tests.
- Pack can express current hardcoded AI/browser/SASE detections.
- Agent evaluator can reject unsupported or unsigned packs.
- Rule evaluation is data-only and cannot execute code.
- Pack version is reported in endpoint telemetry.

## Dependencies

- Existing visibility event schemas.
- Windows/Linux agent collector events.

## Security Notes

- Detection packs are not scripts.
- Packs must be signed before endpoint rollout.

## Implementation Notes

- Added `schemas/detection/detection-pack.v1.schema.json`.
- Added a default AI marker example pack and WO-DET-002 fixtures.
- Added schema validation tests in `backend/detectionpack-schema`.
- Added data-only Linux and Windows evaluator modules aligned to the schema.
- Added Ed25519 signing/verification contract using `aegis.detection_pack.v1\0 || sha256(unsigned_pack_json_bytes)`.
- Agent schema checks reject unsigned, expired, unsupported-OS, unsupported-agent, and non-observe-only packs.
- Dynamic detections include pack id, pack version, and rule id evidence.

## Verification

- `go test ./...` in `backend/detectionpack-schema`
- `cargo test` in `agents/linux-agent`
- `cargo test` in `agents/windows-agent`
- `./scripts/lab/smoke-detection-rollout.sh`
