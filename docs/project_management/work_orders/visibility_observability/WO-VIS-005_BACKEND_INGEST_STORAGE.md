# WO-VIS-005: Backend Ingest and Storage

**Status:** Initial HTTP ingest and lab JSONL store complete  
**Phase:** Visibility and Observability  
**Primary owner:** Backend  

## Goal

Allow the AegisFlux backend to receive, validate, store, and query visibility events from the Windows sensor.

## Scope

This work order covers ingest and persistence for visibility events. It does not include policy decisions or enforcement.

## Deliverables

- Ingest endpoint or message bus consumer for v1 visibility events
- Schema validation
- Initial HTTP JSONL ingest endpoint:
  - `POST /v1/visibility/events`
  - accepts newline-delimited visibility events from the Windows agent spool
  - maps visibility envelopes into the existing ingest `Event` shape
  - publishes accepted events to `events.raw`
- Durable storage for:
  - agent heartbeat
  - process events
  - flow events
  - DNS observations
  - detection events
  - risk findings
- Query model for investigation:
  - by device
  - by process
  - by flow/destination
  - by time range
  - by detection/risk finding
- Basic deduplication by `event_id` - initial in-memory and JSONL-backed duplicate suppression complete
- Ingest health metrics
- Error/dead-letter handling for invalid events

## Acceptance Criteria

- Backend accepts valid v1 visibility events. **Initial endpoint complete.**
- Backend rejects or quarantines invalid events with a useful error. **Initial HTTP 400 handling complete.**
- Re-sending the same event does not create duplicate lab-store records. **Initial JSONL-backed dedupe complete.**
- Query can reconstruct a simple path:
  - device -> user -> parent process -> process -> destination -> DNS/domain -> detection
- Storage can handle the first lab machine event volume without manual cleanup.

## Dependencies

- WO-VIS-004

## Risks

- Existing backend services may have stale contracts or port inconsistencies. Confirm current service boundaries before implementation.
- Storing every raw event without retention policy can grow quickly. Define lab retention defaults.

## Completion Evidence

- Ingest API, lab event store, and query endpoint documented in `aegisflux/backend/ingest/README.md`
- Example producer command:
  ```bash
  curl -sS -X POST --data-binary @events.jsonl http://localhost:9090/v1/visibility/events
  ```
- Unit tests:
  - `go test ./internal/validate ./internal/server`
  - `go test ./cmd/ingest`
- Stored event query examples
- Invalid event handling example

## Remaining Work

- Replace the lab JSONL event store with durable storage tables or Timescale mapping for normalized visibility records.
- Extend deduplication by `event_id` into the database-backed storage layer.
- Add a dead-letter path for invalid events.
- Add authenticated agent-to-ingest transport.
- Add query APIs for device/process/time-range investigation.
