# WO-PLAT-004: Detection Pack Status Visibility

**Status:** In Progress  
**Phase:** Product Platform  
**Primary owner:** UI / Backend  

## Goal

Expose detection-pack rollout health in the AegisFlux console so operators can see what intelligence each device is running, whether it is trusted, and whether rollout is stale, rejected, incompatible, or applied.

## Scope

- Console visibility for detection-pack status.
- Agent list and device detail integration.
- Backend API aggregation if existing endpoints are not enough for UI ergonomics.
- Observe-only status, not enforcement controls.

## Deliverables

- Agent list columns or badges for:
  - active pack id.
  - active pack version.
  - rollout state.
  - stale status.
  - last check or last applied time.
- Device detail panel for:
  - active and previous pack.
  - signature, hash, schema, and compatibility status.
  - rejection reason and reason codes.
  - last rejected pack.
  - latest compatible pack advertised by the controller.
- Detection-pack rollout page or section showing:
  - agents reported.
  - applied, rejected, incompatible, expired, and stale counts.
  - per-agent rollout rows.
- Empty states for no pack, controller unavailable, and stale agent telemetry.
- Deep links from device detail to pack rollout status.

## Acceptance Criteria

- User can identify which pack version each lab device is running.
- User can distinguish `applied`, `rejected`, `incompatible`, `expired`, and `stale`.
- Linux and Windows lab devices render correctly.
- Pack status survives direct page load and refresh.
- UI clearly indicates observe-only status.
- `npm run build` passes.

## Dependencies

- `WO-PLAT-002`
- `WO-DET-003`
- `WO-DET-006`
- Agent status events from Linux and Windows lab agents.

## Non-Goals

- No enforcement or policy approval workflow.
- No pack authoring UI.
- No AI-generated conclusions.
- No production rollout controls.

## Notes

This work order turns the dynamic detection-pack pipeline into a visible product capability. It should stay focused on trust, freshness, compatibility, and operator confidence.

## Implementation Update (May 6, 2026)

### Delivered in this iteration

- `actions-api` now enriches `/agents` and `/agents/{uid}` responses with `detection_pack_status` by querying detection-pipeline (`/v1/agents/{agent_uid}/detection-pack-status`).
- Agent list view now shows detection-pack rollout badges and active pack/version quick visibility.
- Agent detail view now shows observe-only detection pack status details:
  - rollout state.
  - active and previous pack IDs/versions.
  - signature, hash, schema, and compatibility status.
  - last applied, last check, and last rejection reason.
- Empty state is shown when no detection-pack telemetry is available for an agent.

### Follow-up items

- Add a dedicated detection-pack rollout page/section with aggregate counts and per-agent rollout rows.
- Add explicit stale-state UX treatment in the list/detail view from controller stale thresholds.
- Add deep links from device detail to rollout status views when the rollout page exists.

