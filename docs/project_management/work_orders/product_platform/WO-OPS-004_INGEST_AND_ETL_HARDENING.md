# WO-OPS-004: Ingest and ETL Hardening

**Status:** Complete  
**Phase:** Operational Readiness  
**Primary owner:** Backend / Data Pipeline  

## Goal

Harden ingest and ETL/enrichment so endpoint events are validated, stored, summarized, and recoverable under realistic lab noise.

## Problem

The visibility ingest path and summary endpoints exist, but operational readiness depends on stronger validation, dedupe, error handling, backpressure behavior, and clear processing metrics. ETL/enrich also needs continued hardening before it can be trusted as a long-running component.

## Scope

- `backend/ingest`.
- `backend/etl-enrich`.
- Visibility schemas and validation behavior.
- Storage behavior for JSONL/SQLite and any configured lab database path.
- Summary APIs that read ingest data.

## Deliverables

- Review ingest validation against current schemas and add tests for malformed, duplicate, and partial events.
- Ensure dedupe behavior is explicit and measured.
- Add structured processing errors that include event type, device id when safe, and rejection reason.
- Add bounded batch and request body handling where missing.
- Harden ETL/enrich startup, health, metrics, and error handling.
- Document storage mode differences and recovery steps for corrupted or invalid lab data.
- Add a small replay fixture set for representative visibility events.

## Acceptance Criteria

- Ingest rejects malformed events with clear 4xx errors and logs enough context to debug.
- Duplicate events do not produce duplicate operator-visible records.
- Summary endpoints remain stable with empty, partial, duplicate, and malformed-adjacent data.
- ETL/enrich has health/readiness behavior and documented failure states.
- Tests cover validation, dedupe, empty data, and replay fixture behavior.

## Dependencies

- WO-VIS-004.
- WO-VIS-005.
- WO-API-001.
- WO-OPS-003 recommended.

## Non-Goals

- No production-scale streaming redesign.
- No new analytics warehouse.
- No schema-breaking event rename unless separately approved.

## Implementation notes (May 12, 2026)

- **Dedupe metric:** Prometheus counter `events_deduped_total` incremented when a persisted duplicate `event_id` is ignored (`backend/ingest`).
- **Malformed input:** `TestHandleVisibilityEventsRejectsMalformedJSONL` covers bad JSONL.
- **Replay:** `fixtures/visibility/lab-replay-sample.ndjson` and `scripts/lab/replay-visibility-fixtures.sh`.
- **Docs:** `docs/ops/INGEST_ETL_LAB.md` covers validation, storage recovery, and ETL health/metrics references.
- **ETL healthz:** `/healthz` JSON now includes `dependencies` and `started_at` for clearer degraded vs healthy.

## Suggested Verification

- Run ingest server tests.
- Replay fixture events into a local ingest instance.
- Query dashboard, device, and inventory summaries after replay.
- Run ETL/enrich health and metrics checks.
