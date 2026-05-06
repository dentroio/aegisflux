# WO-DET-002: Dynamic AI Detection Candidate Pipeline

**Status:** Draft  
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

