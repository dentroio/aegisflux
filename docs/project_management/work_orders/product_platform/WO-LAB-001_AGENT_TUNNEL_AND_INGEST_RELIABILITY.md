# WO-LAB-001: Agent Tunnel and Ingest Reliability

**Status:** Complete  
**Phase:** Product Platform  
**Primary owner:** Lab / Agent / Backend  

## Goal

Make the Windows and Linux lab agents reliable enough for repeatable AegisFlux product demos and rollout validation. Agents should be able to register, heartbeat, post visibility events, fetch detection packs, and report detection-pack status without manual tunnel repair.

## Scope

- Windows and Linux lab reverse tunnels.
- Agent service or scheduled-task execution.
- Local Docker Compose backend endpoints used by the lab.
- Health checks and recovery steps for developer-machine lab operation.

## Deliverables

- Documented expected tunnel map for each lab agent:
  - Actions API.
  - Detection pipeline.
  - Ingest.
- Reliable startup and stale-forward cleanup for Linux and Windows reverse tunnels.
- Windows ingest-path fix for the observed `9091` connection reset during agent runs.
- Agent runbook for restarting:
  - Docker Compose services.
  - Linux lab systemd agent.
  - Windows lab scheduled task or one-shot script.
  - macOS launchd reverse tunnels.
- Health script coverage for:
  - local service health.
  - tunnel process health.
  - agent freshness.
  - ingest reachability from each lab machine.
- Troubleshooting notes for stale heartbeats, tunnel collisions, and local port drift.

## Acceptance Criteria

- `./scripts/lab/check-agents.sh` passes from a clean lab restart.
- Linux agent can post heartbeat and visibility events without manual intervention.
- Windows agent can post heartbeat and visibility events without manual intervention.
- Both agents can reach Actions API, detection-pipeline, and ingest through their lab path.
- Restarting Docker Compose does not permanently break agent connectivity.
- Restarting tunnels clears stale remote forwards without manual SSH process cleanup.

## Dependencies

- Active Linux and Windows lab machines.
- Existing lab reverse-tunnel scripts.
- Existing Docker Compose lab stack.

## Non-Goals

- No production deployment automation.
- No cloud-hosted control plane.
- No endpoint enforcement.
- No generalized fleet management beyond the local lab.

## Notes

This work order exists because the dynamic pack path is now backend-complete enough that lab reliability is the main blocker to agent-driven validation.

## Implementation Notes (May 6, 2026)

- Reverse tunnel scripts now maintain a consistent Windows/Linux service map:
  - remote `9091 ->` local ingest `9091`
  - remote `8083 ->` local Actions API `8083`
  - remote `8089 ->` local detection pipeline `8089`
- Stale remote forward cleanup is implemented in both tunnel runners:
  - `scripts/lab/aegis-linux-reverse-tunnel.sh`
  - `scripts/lab/aegis-windows-reverse-tunnel.sh`
- Windows stale remote forward cleanup now uses PowerShell on the Windows host
  instead of a Unix shell. This clears bad OpenSSH listeners for `9091/8083/8089`
  before rebinding the reverse tunnel.
- Agent ingest delivery now retries transient tunnel/network failures for both
  Linux and Windows transport paths before surfacing failure.
- `scripts/lab/check-agents.sh` now validates:
  - local compose and service health,
  - local launchd tunnel state,
  - remote tunnel-forward listener presence on each lab host (`9091/8083/8089`),
  - ingest reachability from each remote lab host,
  - freshness of expected agents (`last_seen` threshold).
- Lab restart and troubleshooting runbook details are documented in
  `scripts/lab/README.md`.
- Manual verification after the tunnel cleanup fix confirmed Windows can reach
  ingest at `http://127.0.0.1:9091/healthz` and the Windows one-shot agent can
  post visibility events and heartbeat successfully.
- Acceptance verification passed with `./scripts/lab/check-agents.sh`; both
  `windows-dev-agent-01` and `linux-dev-agent-01` reported online and fresh.
