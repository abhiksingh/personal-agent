param(
    [string]$TaskName = "PersonalAgentDaemon",
    [string]$DaemonBinary = "personal-agent-daemon.exe",
    [string]$ListenMode = "tcp",
    [string]$ListenAddress = "127.0.0.1:7071",
    [Parameter(Mandatory = $true)]
    [string]$AuthTokenFile,
    [string]$DbPath = "$env:APPDATA\personal-agent\runtime.db",
    [switch]$DryRun
)

$daemonArgs = @(
    "--listen-mode", $ListenMode,
    "--listen-address", $ListenAddress,
    "--auth-token-file", $AuthTokenFile,
    "--db", $DbPath
)

if ($DryRun) {
    Write-Output "[dry-run] would register scheduled task '$TaskName'"
    Write-Output "[dry-run] daemon binary: $DaemonBinary"
    Write-Output "[dry-run] daemon args: $($daemonArgs -join ' ')"
    exit 0
}

$action = New-ScheduledTaskAction -Execute $DaemonBinary -Argument ($daemonArgs -join ' ')
$trigger = New-ScheduledTaskTrigger -AtLogOn

if (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue) {
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
}

Register-ScheduledTask -TaskName $TaskName `
    -Description "Personal Agent Daemon auto-start task" `
    -Action $action `
    -Trigger $trigger `
    -RunLevel LeastPrivilege

Write-Output "installed scheduled task: $TaskName"
