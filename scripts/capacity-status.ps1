param(
  [string]$AdminUrl = 'http://127.0.0.1:9090',
  [switch]$FailOnAttention
)

$ErrorActionPreference = 'Stop'
$endpoint = "$($AdminUrl.TrimEnd('/'))/capacity"
$summary = Invoke-RestMethod -Uri $endpoint -Method Get -Headers @{ Accept = 'application/json' }

Write-Output "Capacity health: $($summary.health)"
Write-Output "Process started: $($summary.process_started_at)"
Write-Output "Snapshot generated: $($summary.generated_at)"

if ($summary.events.Count -eq 0) {
  Write-Output 'No capacity outcomes have been recorded since process start.'
} else {
  $summary.events |
    Select-Object component, outcome, count, severity, recommendation |
    Format-Table -AutoSize
}

if ($summary.health -eq 'critical') {
  exit 2
}
if ($FailOnAttention -and $summary.health -eq 'attention') {
  exit 1
}
