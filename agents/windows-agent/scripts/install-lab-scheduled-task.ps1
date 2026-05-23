param(
    [string]$TaskName = "Aegis Windows Agent Lab",
    [string]$RunnerPath = "C:\AegisLab\aegisflux\agents\windows-agent\scripts\run-lab-once.ps1",
    [string]$AgentId = "windows-dev-agent-01",
    [string]$DeviceId = "",
    [string]$BackendUrl = "http://192.168.1.180:9091",
    [string]$ActionsHeartbeatUrl = "http://192.168.1.180:8083/agents/heartbeat",
    [string]$ControllerUrl = "",
    [switch]$DetectionPacksEnabled,
    [string]$DetectionPackCache = "",
    [string]$DetectionPackPublicKey = "",
    [int]$EveryMinutes = 1,
    [switch]$RunNow
)

$ErrorActionPreference = "Stop"

if (!(Test-Path -LiteralPath $RunnerPath)) {
    throw "Runner script was not found at $RunnerPath"
}

$powershell = "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
if ([string]::IsNullOrWhiteSpace($DeviceId)) {
    $DeviceId = $AgentId
}

$runnerArgs = @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-File", "`"$RunnerPath`"",
    "-AgentId", "`"$AgentId`"",
    "-DeviceId", "`"$DeviceId`"",
    "-BackendUrl", "`"$BackendUrl`"",
    "-ActionsHeartbeatUrl", "`"$ActionsHeartbeatUrl`""
)
if (![string]::IsNullOrWhiteSpace($ControllerUrl)) {
    $runnerArgs += @("-ControllerUrl", "`"$ControllerUrl`"")
}
if ($DetectionPacksEnabled) {
    $runnerArgs += "-DetectionPacksEnabled"
}
if (![string]::IsNullOrWhiteSpace($DetectionPackCache)) {
    $runnerArgs += @("-DetectionPackCache", "`"$DetectionPackCache`"")
}
if (![string]::IsNullOrWhiteSpace($DetectionPackPublicKey)) {
    $runnerArgs += @("-DetectionPackPublicKey", "`"$DetectionPackPublicKey`"")
}

$arguments = $runnerArgs -join " "
$action = New-ScheduledTaskAction -Execute $powershell -Argument $arguments
$trigger = New-ScheduledTaskTrigger -Once -At (Get-Date).AddMinutes(1) `
    -RepetitionInterval (New-TimeSpan -Minutes $EveryMinutes) `
    -RepetitionDuration (New-TimeSpan -Days 3650)
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -ExecutionTimeLimit (New-TimeSpan -Minutes 5) `
    -MultipleInstances IgnoreNew `
    -StartWhenAvailable
$currentUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
$principal = New-ScheduledTaskPrincipal `
    -UserId $currentUser `
    -LogonType S4U `
    -RunLevel Limited

Register-ScheduledTask `
    -TaskName $TaskName `
    -Action $action `
    -Trigger $trigger `
    -Settings $settings `
    -Principal $principal `
    -Description "Runs the Aegis Windows lab visibility agent once per interval." `
    -Force | Out-Null

if ($RunNow) {
    Start-ScheduledTask -TaskName $TaskName
}

Get-ScheduledTask -TaskName $TaskName | Select-Object TaskName, State
