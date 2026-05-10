# Windows lab baseline: `windows-dev-agent-01`

This document satisfies **WO-VIS-001** (repeatable Windows 11 visibility lab host). Capture real values on the machine after provisioning.

## Identity

| Field | Value |
|-------|--------|
| Host name | `windows-dev-agent-01` |
| Snapshot / checkpoint name | `baseline-visibility-v1` |
| Admin account | |
| Standard user (lab) | |
| AegisFlux ingest base URL | |

## Required directories

Create on the lab host:

- `C:\AegisLab\repos`
- `C:\AegisLab\scripts`
- `C:\AegisLab\downloads`
- `C:\AegisLab\evidence`

Use `scripts/lab/windows-dev-agent-01/setup-lab-dirs.ps1` for a quick scaffold.

## Software inventory (fill in versions)

| Component | Version / notes |
|-----------|-----------------|
| Windows 11 build | |
| Microsoft Edge | |
| Google Chrome | |
| Cursor | |
| Visual Studio Code | |
| Git for Windows | |
| Python | |
| Node.js | |
| PowerShell 7 (`pwsh`) | |

## Connectivity checks

1. `curl -fsS http://<ingest-host>:<port>/readyz` from the lab user session.
2. Run the Windows agent once with `--once` and confirm events appear in ingest or local JSONL spool.

## Scenario readiness

- [ ] Browser can reach a known HTTPS endpoint (AI SaaS or lab mock).
- [ ] IDE (Cursor/VS Code) can launch helper runtimes.
- [ ] `C:\AegisLab\scripts` contains sample Python and Node scripts (see repo `scripts/lab/windows-dev-agent-01/samples/`).
- [ ] PowerShell 7 available for automation scripts.
- [ ] Git can reach `github.com` or an internal remote for flow attribution drills.

## Evidence for completion

- Export of `winver` and `python --version`, `node --version`, `git --version`, `pwsh --version`.
- Screenshot or note documenting the hypervisor snapshot name and storage location.
- This table filled in and stored with the lab runbook.
