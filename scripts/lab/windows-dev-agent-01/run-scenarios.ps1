#Requires -Version 5.1
<#
.SYNOPSIS
  WO-VIS-008 manual scenario checklist runner (lab). Records intent; does not replace backend smoke tests.
#>
param(
    [string]$IngestHealthUrl = "http://127.0.0.1:9090/readyz"
)

$ErrorActionPreference = "Continue"
Write-Host "=== Aegis lab scenario runner ==="
Write-Host "1) Backend reachability: $IngestHealthUrl"
try {
    Invoke-WebRequest -Uri $IngestHealthUrl -UseBasicParsing -TimeoutSec 5 | Out-Null
    Write-Host "   OK"
}
catch {
    Write-Warning "   Ingest not reachable: $_"
}

Write-Host "2) Run sample Python agent script (set AEGIS_LAB_TARGET_URL to lab mock if needed)"
Write-Host "   py -3 C:\AegisLab\scripts\sample_agent_runner.py"

Write-Host "3) Run sample Node agent script"
Write-Host "   node C:\AegisLab\scripts\sample_node_agent.mjs"

Write-Host "4) PowerShell outbound sample"
Write-Host "   pwsh -NoProfile -Command `"Invoke-WebRequest -Uri https://example.com/ -UseBasicParsing`""

Write-Host "5) Git remote (attributes to git.exe when observed via netstat)"
Write-Host "   git ls-remote https://github.com/octocat/Hello-World.git HEAD"

Write-Host "Done. Capture agent JSONL or POST to /v1/visibility/events and archive under C:\AegisLab\evidence\"
