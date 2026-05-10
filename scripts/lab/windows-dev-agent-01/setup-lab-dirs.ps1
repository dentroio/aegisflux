#Requires -Version 5.1
<#
.SYNOPSIS
  Creates standard Aegis visibility lab directories on windows-dev-agent-01 (WO-VIS-001).
#>
$ErrorActionPreference = "Stop"
$roots = @(
    "C:\AegisLab\repos",
    "C:\AegisLab\scripts",
    "C:\AegisLab\downloads",
    "C:\AegisLab\evidence"
)
foreach ($p in $roots) {
    New-Item -ItemType Directory -Force -Path $p | Out-Null
    Write-Host "OK $p"
}
