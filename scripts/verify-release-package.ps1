param(
  [Parameter(Mandatory = $true)]
  [ValidateNotNullOrEmpty()]
  [string]$PackageDirectory
)

$ErrorActionPreference = 'Stop'

function Get-RequiredProperty {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Object,
    [Parameter(Mandatory = $true)]
    [string]$Name
  )

  $property = $Object.PSObject.Properties[$Name]
  if ($null -eq $property -or $null -eq $property.Value -or [string]::IsNullOrWhiteSpace([string]$property.Value)) {
    throw "Release manifest is missing required property '$Name'."
  }

  return $property.Value
}

if (-not (Test-Path -LiteralPath $PackageDirectory -PathType Container)) {
  throw "Release package directory does not exist: $PackageDirectory"
}

$resolvedDirectory = (Resolve-Path -LiteralPath $PackageDirectory).Path
$manifestPath = Join-Path $resolvedDirectory 'release-manifest.json'
if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
  throw "Release manifest does not exist: $manifestPath"
}

try {
  $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json -ErrorAction Stop
} catch {
  throw "Release manifest is not valid JSON: $manifestPath"
}

$version = [string](Get-RequiredProperty -Object $manifest -Name 'version')
if ($version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$') {
  throw "Release manifest has an invalid SemVer version: $version"
}

$commit = [string](Get-RequiredProperty -Object $manifest -Name 'commit')
if ($commit -notmatch '^[0-9a-f]{40}$') {
  throw "Release manifest commit must be a full lowercase Git SHA: $commit"
}

$artifactsProperty = $manifest.PSObject.Properties['artifacts']
if ($null -eq $artifactsProperty -or $null -eq $artifactsProperty.Value) {
  throw 'Release manifest is missing artifacts.'
}

$artifacts = @($artifactsProperty.Value)
$expectedTargets = @('server', 'worker', 'migrate')
if ($artifacts.Count -ne $expectedTargets.Count) {
  throw "Release manifest must contain exactly $($expectedTargets.Count) artifacts."
}

$seenTargets = @{}
foreach ($artifact in $artifacts) {
  $target = [string](Get-RequiredProperty -Object $artifact -Name 'target')
  if ($target -notin $expectedTargets -or $seenTargets.ContainsKey($target)) {
    throw "Release manifest has an unexpected or duplicate target: $target"
  }
  $seenTargets[$target] = $true

  $expectedImage = "opendroneops/$target`:$version"
  $image = [string](Get-RequiredProperty -Object $artifact -Name 'image')
  if ($image -ne $expectedImage) {
    throw "Release artifact '$target' has unexpected image reference: $image"
  }

  $archive = [string](Get-RequiredProperty -Object $artifact -Name 'archive')
  $expectedArchives = @(
    "opendroneops-$target-$version.tar",
    "opendroneops-$target-$version.tar.gz"
  )
  if ($archive -notin $expectedArchives -or [IO.Path]::GetFileName($archive) -ne $archive) {
    throw "Release artifact '$target' has an unsafe or unexpected archive name: $archive"
  }

  $expectedHash = [string](Get-RequiredProperty -Object $artifact -Name 'sha256')
  if ($expectedHash -notmatch '^[0-9a-f]{64}$') {
    throw "Release artifact '$target' has an invalid SHA-256 digest."
  }

  $archivePath = Join-Path $resolvedDirectory $archive
  if (-not (Test-Path -LiteralPath $archivePath -PathType Leaf)) {
    throw "Release artifact '$target' is missing its archive: $archive"
  }

  $actualHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($actualHash -ne $expectedHash) {
    throw "Release artifact '$target' SHA-256 does not match release-manifest.json."
  }
}

foreach ($target in $expectedTargets) {
  if (-not $seenTargets.ContainsKey($target)) {
    throw "Release manifest is missing required target: $target"
  }
}

Write-Output "Release package verified: $version ($commit)"
