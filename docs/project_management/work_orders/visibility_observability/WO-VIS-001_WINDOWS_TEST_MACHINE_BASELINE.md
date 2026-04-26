# WO-VIS-001: Windows Test Machine Baseline

**Status:** Draft  
**Phase:** Visibility and Observability  
**Primary owner:** Lab / Agent  
**Target environment:** `windows-dev-agent-01`

## Goal

Create a repeatable Windows 11 test machine that can exercise browser, IDE, script, AI-agent, automation, downloaded software, and normal business application scenarios.

## Scope

Build and document the baseline machine image. This machine is the first target for the Windows Aegis visibility sensor.

## Deliverables

- Windows 11 VM or physical test machine named `windows-dev-agent-01`
- Local admin account and standard user account
- Network access to Aegis backend
- Browser set: Edge and Chrome
- Developer tools: Cursor, VS Code, Git, Python, Node.js, PowerShell 7
- Test directories:
  - `C:\AegisLab\repos`
  - `C:\AegisLab\scripts`
  - `C:\AegisLab\downloads`
  - `C:\AegisLab\evidence`
- Sample local repo with non-sensitive source files
- Sample Python and Node scripts for controlled process/network generation
- Baseline inventory document with installed versions
- Snapshot/checkpoint named `baseline-visibility-v1`

## Test Scenarios to Support

- Browser AI: browser reaches model/SaaS endpoints
- IDE AI assistant: Cursor/VS Code launches helper process and makes outbound calls
- Local agent script: Python/Node process reads files and calls network services
- PowerShell automation: scripted outbound call and command-line capture
- Downloaded software: unknown executable behavior in `C:\AegisLab\downloads`
- Normal application baseline: browser, Git, package manager, terminal

## Acceptance Criteria

- Machine can be rebuilt or restored from snapshot.
- Machine can reach the Aegis backend.
- Test user can run browser, IDE, Python, Node, Git, and PowerShell scenarios.
- All installed tools and versions are documented.
- No enforcement driver or blocking component is required for this work order.

## Dependencies

- None

## Risks

- Tools such as Cursor or extensions may update behavior over time. Capture versions.
- External SaaS/model endpoints may be unavailable or rate-limited. Provide local mock endpoints where possible.

## Completion Evidence

- Screenshot or command output showing installed tool versions
- Snapshot name and location
- Baseline inventory document
- Successful outbound connection to Aegis backend health endpoint
