# WO-GROWTH-004: Agent Health and Readiness Scoring

**Status:** Done  
**Phase:** Product Growth / Trust  
**Primary owner:** Backend / Agent / UI  

## Goal

Create an agent health and readiness score so operators know whether each endpoint can be trusted for evidence, detection rollout, and future control decisions.

## Why This Matters

Before Aegis can recommend controls, customers must trust the data source. Fleet health should answer:

- Is the agent alive?
- Is telemetry fresh?
- Are collectors healthy?
- Is the detection pack current?
- Is the agent version current?
- Is the endpoint ready for simulation or future enforcement?

## Scope

- Readiness score model.
- Fleet and endpoint UI.
- Backend summary support.
- Lab connectivity and stale-agent explanation.

## Deliverables

- Backend `backend/actions-api/internal/api/agent_readiness.go` introduces:
  - `AgentReadiness`, `AgentReadinessDimension`, `FleetReadinessSummary`, `FleetReadinessAgent`, `FleetReadinessHotspot` types.
  - `computeAgentReadiness` builds six explainable dimensions: `heartbeat`, `event_ingestion`, `detection_pack`, `agent_version`, `collectors`, `connectivity`.
  - Five-bucket model: `ready`, `needs_attention`, `stale`, `degraded`, `unknown`. Weighted score (0-100) used to keep the chart honest when individual signals are mixed.
  - `summarizeReadiness` and `pickFixFirst` write operator-language summary and "fix first" guidance.
- `AgentInfo` extended with optional `Readiness` so the existing agents-workbench summary returns the score in place.
- `getAgentsWorkbenchSummary` now stamps each agent with a readiness object using the env-configurable `AEGISFLUX_AGENT_BASELINE_VERSION` for drift comparison.
- New endpoint `GET /console/summary/agent-readiness` returns fleet aggregate: bucket counts, average score, per-agent readiness, and a "fix first" hotspot list (max 8) sorted by severity.
- Tests in `agent_readiness_test.go`:
  - `TestComputeAgentReadiness_ReadyBucket`
  - `TestComputeAgentReadiness_StaleHeartbeat`
  - `TestComputeAgentReadiness_NeedsAttentionOnVersionDrift`
  - `TestComputeAgentReadiness_UnknownNoData`
  - `TestComputeAgentReadiness_DegradedWhenManyBad`
- Console `AgentsManagementPanel`:
  - New exported types `AgentReadiness`, `AgentReadinessDimension`, plus `READINESS_BUCKET_LABEL` and `READINESS_BUCKET_TONE`.
  - `FleetReadinessStrip` summary above the existing KPI row with bucket counts and average score.
  - `ReadinessBadge` rendered inside the Status column of every agent row.
- Agent detail page `app/agents/[device_id]/page.tsx`:
  - Fetches the fleet readiness response and finds the row matching this device/agent.
  - New `ReadinessExplanation` section at the top of the Health tab: bucket badge, score, summary, "Fix first" callout, and per-dimension explanation cards with value + detail.
  - Closes with an explicit observe-only disclaimer so readiness is not read as enforcement-ready.

## Acceptance Criteria

- [x] Operator can identify agents that are not trustworthy for decisions — bucket badge in the row, hotspot list in the new endpoint.
- [x] Agent detail explains the readiness score in plain language — `ReadinessExplanation` section with dimensions and "Fix first" copy.
- [x] Health scoring works with empty/missing telemetry — `unknown` and `stale` buckets covered by tests.
- [x] Score does not imply enforcement readiness — observe-only disclaimer in UI and "what to fix first" phrasing in backend summaries.
- [x] Relevant backend tests pass — see `agent_readiness_test.go`.
- [x] `npm run build` passes in `ui/console`.

## Dependencies

- WO-AGENT-001
- WO-PLAT-005
- WO-API-001

## Non-Goals

- No auto-remediation.
- No production deployment posture scoring.
- No enforcement readiness claims beyond lab/audit-mode readiness.

## Suggested Verification

- `cd backend/actions-api && go test ./internal/api -run TestComputeAgentReadiness`.
- `cd ui/console && npm run build`.
- Manual check with online and stale lab agents.
