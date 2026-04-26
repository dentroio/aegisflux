# WO-VIS-007: AI-Agent Detection Pack 1

**Status:** Draft; findings API surface ready  
**Phase:** Visibility and Observability  
**Primary owner:** Detection / Agent / Backend  

## Goal

Implement the first non-blocking AI-agent and automation detection pack using observed process, flow, DNS, and command-line evidence.

## Scope

Detection only. This work order creates findings and evidence. It does not block, redirect, quarantine, or change SGTs.

## Scenarios

### Scenario 1: Browser AI

Example:

- `chrome.exe` or `msedge.exe` reaches known AI/SaaS/model destinations.

Expected classification:

- `application_category = browser`
- `ai_usage = browser_ai`
- `agent_likelihood = low_to_medium`
- Evidence should not claim autonomous host agent behavior unless helper processes or tool loops are observed.

### Scenario 2: IDE AI Assistant

Example:

- `cursor.exe` or `Code.exe` launches `node.exe` or `python.exe`.
- Helper process calls model/API destination or reads a repo.

Expected classification:

- `application_category = developer_tool`
- `agent_likelihood = medium_to_high`
- Evidence includes parent process, helper process, command line, destination, and repo path if available.

### Scenario 3: Local Agent Script

Example:

- `python.exe agent_runner.py` or `node agent.js` reads files and calls model/API or internal service.

Expected classification:

- `application_category = script_or_agent_runtime`
- `agent_likelihood = high` when framework/tool-loop evidence exists
- Evidence includes runtime, script path, command line, network destination, and detected framework strings when available.

### Scenario 4: PowerShell Automation

Example:

- `powershell.exe` runs encoded/scripted command and makes outbound network connection.

Expected classification:

- `application_category = shell_automation`
- `risk_signal = suspicious_automation` when command shape or destination warrants it

## Detection Output

Each finding should include:

- `finding_id`
- `device_id`
- `process_guid`
- `flow_id` when applicable
- `classification`
- `agent_likelihood`
- `confidence`
- `risk_score`
- `detected_patterns`
- `evidence`
- `recommended_action`: `monitor`, `review`, or `policy_candidate`

## Acceptance Criteria

- All four scenarios produce findings or explicit non-findings with reasons.
- Browser AI is not automatically labeled as an autonomous agent.
- IDE/helper and local script scenarios include process lineage evidence.
- Detection confidence is explainable and not just a fixed label.
- Findings can be stored and queried by the backend. **Initial fixture-backed query API complete via `GET /v1/visibility/findings`.**

## Dependencies

- WO-VIS-002
- WO-VIS-003
- WO-VIS-004
- WO-VIS-005

## Risks

- Vendor/tool process behavior changes frequently. Version all scenario fixtures.
- Detection rules can overfit to one tool. Keep signal families explicit.
- Privacy boundaries for file-path and repo evidence need policy review.

## Completion Evidence

- Scenario run logs
- Finding JSON examples
- False-positive notes
- Known gaps list
