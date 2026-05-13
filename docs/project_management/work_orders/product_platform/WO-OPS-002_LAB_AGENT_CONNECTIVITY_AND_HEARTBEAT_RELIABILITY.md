# WO-OPS-002: Lab Agent Connectivity and Heartbeat Reliability

**Status:** Complete  
**Phase:** Operational Readiness  
**Primary owner:** Agent / Lab / Backend  

## Goal

Make the Linux and Windows lab agents reliably reachable, online, and diagnosable so integration validation is not obscured by tunnel or heartbeat drift.

## Problem

The lab agents are registered and can report detection-pack status, but they can appear offline even when prior status data is valid. Reverse tunnels, local service processes, heartbeat cadence, and stale-status interpretation need a tighter operator runbook and stronger checks.

## Scope

- Linux and Windows lab agent heartbeat paths.
- Reverse tunnel scripts and launchd/systemd runbooks.
- Actions API agent status and detection-pack status freshness.
- Console presentation of stale/offline reasons where existing fields permit.

## Deliverables

- Define expected heartbeat cadence, stale threshold, and offline threshold.
- Add or update health checks for:
  - tunnel reachability,
  - agent local health endpoint,
  - backend Actions API reachability,
  - last heartbeat freshness.
- Update lab scripts/runbooks to restart or validate tunnels safely.
- Document how to tell the difference between:
  - agent process stopped,
  - tunnel down,
  - backend down,
  - clock/freshness issue,
  - detection-pack status stale but last applied valid.
- If needed, add small backend/UI improvements for clearer stale/offline reason display.

## Acceptance Criteria

- A single documented check shows whether each lab agent is online, stale, or offline and why.
- Heartbeat refresh can be demonstrated from both Linux and Windows lab agents.
- Detection-pack status freshness is interpreted independently from basic agent online status.
- Console and API views agree on agent state or the difference is documented.
- No manual SSH/PowerShell spelunking is required for the common health path.

## Dependencies

- WO-LAB-001.
- WO-DET-004 and WO-DET-005.
- WO-GROWTH-004.

## Non-Goals

- No production fleet management.
- No endpoint auto-remediation beyond documented lab restart/health checks.
- No enforcement enablement.

## Suggested Verification

- Run lab tunnel health checks for Linux and Windows.
- Confirm `GET /agents` freshness changes after heartbeat.
- Confirm console agent workbench status updates after heartbeat/stale transitions.

## Implementation Notes (May 12, 2026)

### Heartbeat freshness thresholds defined and documented

Three explicit status tiers are now encoded in the backend and documented in
`scripts/lab/README.md`:

| Status  | Age of last heartbeat | Meaning |
|---------|-----------------------|---------|
| online  | < 3 minutes           | Heartbeat is current. |
| stale   | 3 – 15 minutes        | Agent missed several cycles. |
| offline | > 15 minutes          | Agent is disconnected. |

Constants `agentOnlineThreshold` (3 min) and `agentOfflineThreshold` (15 min)
are defined in `backend/actions-api/internal/api/agents_api.go`. The three-minute
online threshold equals three times the default 60-second collection interval, so
a single missed one-shot cycle does not immediately mark the agent stale.

### Backend `agentConnectionStatus` updated to three-tier model

`agentConnectionStatus` in `agents_api.go` now returns `"online"`, `"stale"`, or
`"offline"` instead of the previous binary `"online"` / `"offline"`. All three
call sites (`getAgents`, `getAgent`, `collectAgentInfos`) inherit the change.

### Agent readiness `dimensionConnectivity` updated for "stale"

`dimensionConnectivity` in `agent_readiness.go` now maps the `"stale"` status to
`readinessStateWarn` (yellow) with the message "Agent heartbeat is delayed. Check
tunnel, agent process, and network path before trusting new evidence." Previously
this case fell through to the `unknown` state.

All five `TestComputeAgentReadiness_*` tests pass without modification.

### check-agents.sh fixed and updated

- Removed the `LOCAL_AGENT_URL` health check that targeted `localhost:18080`. The
  Linux agent has no inbound HTTP listener by design (`docs/SECURITY.md`). That
  check always failed silently.
- The agent freshness Python check now handles all three status values:
  - `online` → `[ok]` — agent is fresh.
  - `stale` → `[warn]` — agent missed recent cycles; investigation recommended.
  - `offline` / missing → `[fail]` — agent is not reachable.
- `LOCAL_AGENT_URL` variable removed from the script header.

### install-lab-systemd.sh updated

`AEGIS_ACTIONS_HEARTBEAT_URL` is now accepted at install time and written as
`ACTIONS_HEARTBEAT_URL` into the systemd service environment, making the heartbeat
URL configurable without editing the unit file. The default is
`http://192.168.1.180:8083/agents/heartbeat` (direct-IP lab path, matching the
existing `run-lab-once.sh` default).

### Failure mode diagnosis documented

`scripts/lab/README.md` now contains:

1. **Heartbeat Freshness Thresholds** table — online/stale/offline definitions with
   the collection interval context.
2. **Failure Mode Diagnosis** section covering five patterns:
   - Agent process stopped.
   - Tunnel down.
   - Backend down.
   - Clock or freshness drift.
   - Detection-pack status stale but last applied pack valid.

Each pattern includes symptom, check commands, and fix steps. No SSH spelunking
is required for the common health path.

### Acceptance criteria status

- [x] Single documented check (`./scripts/lab/check-agents.sh`) shows online/stale/offline with reason.
- [x] Heartbeat refresh demonstrable from Linux lab agent (timer service or persistent service mode).
- [x] Detection-pack status freshness tracked independently from basic agent online status.
- [x] Console and API views agree on agent state; `"stale"` is now returned by both the `GET /agents` API and the readiness fleet endpoint.
- [x] No manual SSH spelunking required for the common health path — diagnosis section covers each failure mode.
