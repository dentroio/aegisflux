# WO-DET-002: Dynamic AI Detection Candidate Pipeline

**Status:** Backend complete; UI page pending
**Phase:** Product Platform  
**Primary owner:** Backend / AI Platform  

## Goal

Create the backend workflow for research-to-detection: research item -> candidate rule -> validation -> approved signed pack.

## Scope

- Candidate lifecycle only.
- Lab validation first.
- Human approval before signed pack promotion.

## Deliverables

- Research item record.
- Candidate detection-pack change record.
- Validation result record.
- Approval state machine:
  - draft
  - validating
  - validation_failed
  - ready_for_review
  - approved
  - rejected
  - signed
- API to list candidates and validations.
- UI page under Analyze > Detection Packs.
- Initial candidate for MCP/local model/tool bridge detection.

## Acceptance Criteria

- Candidate can be created without endpoint rollout.
- Candidate can be validated against stored lab telemetry.
- Failed validation records show why.
- Approved candidate can produce a signed pack artifact.
- Endpoint rollout remains separate.

## Dependencies

- WO-DET-001
- WO-AI-002

## Non-Goals

- No fully autonomous publishing.
- No production customer rollout.

## Implementation notes

- **Service:** `backend/detection-pipeline` (HTTP on `DETECTION_PIPELINE_HTTP_ADDR`, default `:8089`).
- **Pack contract:** `schemas/detection/detection-pack.v1.schema.json` (WO-DET-001).
- **Fixtures:** `schemas/detection/fixtures/wo-det-002/` (MCP / local-model / tool-bridge candidate + lab telemetry sample).
- **Compose:** `detection-pipeline` and `ingest` volume `ingest_visibility_data` so lab events persist for validation queries.
- **Routes:** research item create/list, candidate create/list/detail, validation, approval, rejection, signing, signed-pack fetch, and signer-info are implemented under `/v1/detection/...`.
- **Validation:** candidate rules are evaluated against stored ingest telemetry and move through `validating`, `validation_failed`, `ready_for_review`, `approved`, `rejected`, and `signed`.
- **Signing:** approved candidates can produce signed `detection_pack.v1` artifacts with Ed25519 signatures.

## Remaining Work

- Dedicated console page under Analyze > Detection Packs for candidate lifecycle and validations.
- Product polish for reviewer workflows; API is available, but the primary UI is not complete.

## Verification

- `go test ./...` in `backend/detection-pipeline`
- `./scripts/lab/smoke-detection-rollout.sh`
