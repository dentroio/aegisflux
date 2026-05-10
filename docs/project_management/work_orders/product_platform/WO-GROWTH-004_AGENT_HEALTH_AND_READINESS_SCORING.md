# WO-GROWTH-004: Agent Health and Readiness Scoring

**Status:** Draft  
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

- Define health dimensions:
  - heartbeat freshness
  - collector health
  - event ingestion recency
  - detection-pack freshness
  - agent version drift
  - queue/spool pressure
  - tunnel/connectivity status where available
- Add score buckets:
  - ready
  - needs attention
  - stale
  - degraded
  - unknown
- Add backend summary endpoint or extend existing agent summary.
- Add UI:
  - fleet readiness strip
  - agent row readiness badge
  - endpoint detail readiness explanation
  - "what to fix first" guidance
- Add docs mapping score to evidence trust.

## Acceptance Criteria

- Operator can identify agents that are not trustworthy for decisions.
- Agent detail explains the readiness score in plain language.
- Health scoring works with empty/missing telemetry.
- Score does not imply enforcement readiness unless all required dimensions are present.
- Relevant backend tests pass.
- `npm run build` passes in `ui/console`.

## Dependencies

- WO-AGENT-001
- WO-PLAT-005
- WO-API-001

## Non-Goals

- No auto-remediation.
- No production deployment posture scoring.
- No enforcement readiness claims beyond lab/audit-mode readiness.

## Suggested Verification

- Backend tests for readiness score buckets.
- `npm run build` in `ui/console`.
- Manual check with online and stale lab agents.

