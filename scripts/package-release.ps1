param(
  [Parameter(Mandatory = $true)]
  [ValidatePattern('^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$')]
  [string]$Version,
  [string]$OutputDirectory = 'dist/release'
)

$ErrorActionPreference = 'Stop'

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  throw 'Docker is required to package release-candidate images.'
}

$commit = (git rev-parse HEAD).Trim()
$buildTime = [DateTime]::UtcNow.ToString('o')
$resolvedOutput = Join-Path (Get-Location) $OutputDirectory
New-Item -ItemType Directory -Force -Path $resolvedOutput | Out-Null

$artifacts = @()
foreach ($target in @('server', 'worker', 'migrate')) {
  $image = "opendroneops/$target`:$Version"
  $archive = Join-Path $resolvedOutput "opendroneops-$target-$Version.tar"
  docker build --target $target --build-arg "VERSION=$Version" --build-arg "COMMIT=$commit" --build-arg "BUILD_TIME=$buildTime" -t $image .
  docker save --output $archive $image
  $artifacts += [pscustomobject]@{
    target = $target
    image = $image
    archive = [IO.Path]::GetFileName($archive)
    sha256 = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
  }
}

[pscustomobject]@{
  version = $Version
  commit = $commit
  created_at = $buildTime
  artifacts = $artifacts
} | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $resolvedOutput 'release-manifest.json') -Encoding utf8

& (Join-Path $PSScriptRoot 'verify-release-package.ps1') -PackageDirectory $resolvedOutput

Write-Output "Release candidate package created at $resolvedOutput"
