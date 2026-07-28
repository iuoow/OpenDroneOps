param(
  [switch]$RequireApproved
)

$ErrorActionPreference = 'Stop'
$registryPath = Join-Path $PSScriptRoot '..\docs\development\gates\task20\gate-registry.json'
$registry = Get-Content -LiteralPath $registryPath -Raw | ConvertFrom-Json

if ($registry.schemaVersion -ne 1) {
  throw "Unsupported Task 20 gate schema version: $($registry.schemaVersion)"
}

$expectedIds = @(
  'supported_model',
  'credentials_license',
  'authorized_lab',
  'security_privacy',
  'operating_sop',
  'command_drc_approval'
)
$validStatuses = @('missing', 'draft', 'submitted', 'approved', 'rejected')
$actualIds = @($registry.items | ForEach-Object { $_.id })

if ($actualIds.Count -ne $expectedIds.Count -or
    (@($expectedIds | Where-Object { $_ -notin $actualIds }).Count -gt 0)) {
  throw 'Task 20 gate registry must contain exactly the six required gate items'
}

foreach ($item in $registry.items) {
  if ($item.status -notin $validStatuses) {
    throw "Invalid status for $($item.id): $($item.status)"
  }
  if ($item.secretAllowed -ne $false) {
    throw "Secrets are never allowed for gate item $($item.id)"
  }
  if ($item.status -eq 'approved' -and
      [string]::IsNullOrWhiteSpace([string]$item.externalRecordRef)) {
    throw "Approved gate item $($item.id) requires an opaque external record reference"
  }
}

$missingOrUnapproved = @($registry.items | Where-Object { $_.status -ne 'approved' })
if ($RequireApproved -and $missingOrUnapproved.Count -gt 0) {
  throw "Task 20 gate is not approved; unresolved items: $($missingOrUnapproved.id -join ', ')"
}

if ($registry.overallStatus -eq 'approved' -and $missingOrUnapproved.Count -gt 0) {
  throw 'overallStatus cannot be approved while any gate item is unresolved'
}

Write-Output ("TASK20_GATE_VALID: overallStatus={0}; unresolved={1}" -f
  $registry.overallStatus, $missingOrUnapproved.Count)
