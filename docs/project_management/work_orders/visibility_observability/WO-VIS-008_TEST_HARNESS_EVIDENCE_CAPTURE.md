# WO-VIS-008: Test Harness and Evidence Capture

**Status:** Initial backend fixture smoke harness complete  
**Phase:** Visibility and Observability  
**Primary owner:** QA / Lab / Detection  

## Goal

Create a repeatable way to run visibility scenarios and capture expected telemetry, detections, and evidence.

## Scope

Test harness, fixtures, and evidence capture for Phase 1 visibility. This should support manual and automated validation.

## Deliverables

- Scenario runner scripts for Windows lab machine
- Backend fixture smoke script for process -> flow -> DNS -> findings investigation path - initial script available at `backend/ingest/scripts/smoke_visibility_investigation.sh`
- Scenario definitions for:
  - browser AI
  - IDE AI assistant
  - local Python agent script - initial backend fixture available at `backend/ingest/testdata/visibility/investigation_path.jsonl`
  - local Node agent script
  - PowerShell automation
  - normal browser baseline
  - normal Git/package manager baseline
- Expected event checklist per scenario
- Evidence bundle format
- Test result template
- Known-good baseline results

## Evidence Bundle Format

Each scenario run should capture:

- scenario ID
- start and end time
- machine identity
- logged-in user
- commands executed
- process events observed
- flow events observed
- DNS events observed
- detection findings
- missing expected events
- screenshots or API responses where useful
- tool versions

## Acceptance Criteria

- A tester can run a backend fixture scenario and produce comparable investigation counts. **Initial backend smoke harness complete.**
- Expected vs observed telemetry is easy to review.
- Harness can confirm whether required process/flow/DNS/detection events appeared. **Initial backend fixture path complete.**
- Results can be used to tune detection without rerunning everything manually.

## Dependencies

- WO-VIS-001
- WO-VIS-002
- WO-VIS-003
- WO-VIS-007

## Risks

- Fully automated browser/IDE scenarios may be brittle. Allow manual step capture for v1.
- External service dependencies can make test results inconsistent. Provide local mock endpoints where feasible.

## Completion Evidence

- Scenario runner scripts
- Backend fixture smoke output from `./backend/ingest/scripts/smoke_visibility_investigation.sh`
- At least three captured evidence bundles
- Expected-vs-observed report
