# WO-DET-006: Repeatable Detection Rollout Smoke

**Status:** In Progress  
**Phase:** Product Platform  
**Primary owner:** Backend / Agent / Lab  

## Goal

Turn the manual WO-DET-002 and WO-DET-003 rollout validation into a repeatable smoke test. A developer should be able to run one script that proves the detection-pack pipeline, rollout controller, agents, and status reporting still work together.

## Scope

- Lab-only smoke automation.
- Existing WO-DET-002 fixture telemetry.
- Detection pipeline research, candidate, validation, approval, signing, latest-pack, artifact, and rollout-status APIs.
- Linux and Windows agent detection-pack status.
- Clear pass/fail output for developer use.

## Deliverables

- Script: `scripts/lab/smoke-detection-rollout.sh`.
- The script should:
  - verify local service health.
  - post WO-DET-002 lab telemetry to ingest.
  - create a research item from the fixture.
  - create a candidate using the research item id.
  - validate the candidate against fixture telemetry.
  - approve and sign the candidate.
  - assert latest-pack lookup for Linux and Windows.
  - assert artifact retrieval with `os`, `agent_version`, and `version`.
  - trigger or observe Linux and Windows agent pack status.
  - assert rollout status reports both agents as non-stale and applied.
- Concise output that can be pasted into a release or demo note.
- Failure messages that identify the broken stage and next command to run.

## Acceptance Criteria

- Smoke passes against the local Docker Compose lab stack.
- Smoke fails clearly when ingest, detection-pipeline, or Actions API is down.
- Smoke detects when either lab agent is stale.
- Smoke verifies the signed artifact has the expected pack id, version, and rules.
- Smoke verifies both `linux-dev-agent-01` and `windows-dev-agent-01` report the active pack.
- The script does not require editing repository files or committing generated artifacts.

## Dependencies

- `WO-DET-001`
- `WO-DET-002`
- `WO-DET-003`
- `WO-DET-004`
- `WO-DET-005`
- `WO-LAB-001`

## Non-Goals

- No production rollout.
- No load testing.
- No enforcement validation.
- No external AI or research-agent calls.

## Notes

The first version may post status through the controller API for continuity, but the target is agent-driven status after `WO-LAB-001`, `WO-DET-004`, and `WO-DET-005` are stable in the lab.

## Implementation (current)

- Added `scripts/lab/smoke-detection-rollout.sh`.
- The smoke executes fixture ingest, research creation, candidate creation, validation, approval, signing, latest-pack checks for Linux/Windows, artifact header/body checks, and rollout-status verification for both lab agents.
- The script uses a unique `pack_version` per run by default to keep runs repeatable without editing fixtures.
- Failures are stage-scoped and include a concrete `next:` command for triage.

## Operator usage

```bash
./scripts/lab/smoke-detection-rollout.sh
```

Useful overrides:

- `DETECTION_URL` (default `http://127.0.0.1:8089`)
- `INGEST_URL` (default `http://127.0.0.1:9090`)
- `ACTIONS_URL` (default `http://127.0.0.1:8083`)
- `LINUX_AGENT_UID` (default `linux-dev-agent-01`)
- `WINDOWS_AGENT_UID` (default `windows-dev-agent-01`)
- `AGENT_VERSION` (default `0.1.0`)

