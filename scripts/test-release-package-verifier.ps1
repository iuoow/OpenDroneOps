$ErrorActionPreference = 'Stop'

$verifier = Join-Path $PSScriptRoot 'verify-release-package.ps1'
$packageDirectory = Join-Path ([IO.Path]::GetTempPath()) ("opendroneops-release-verifier-" + [Guid]::NewGuid().ToString('N'))

try {
  New-Item -ItemType Directory -Path $packageDirectory | Out-Null
  $version = 'v0.1.0-rc.1'
  $commit = '0123456789abcdef0123456789abcdef01234567'
  $artifacts = @()

  foreach ($target in @('server', 'worker', 'migrate')) {
    $archive = "opendroneops-$target-$version.tar"
    $archivePath = Join-Path $packageDirectory $archive
    [IO.File]::WriteAllBytes($archivePath, [Text.Encoding]::UTF8.GetBytes("$target archive"))
    $artifacts += [pscustomobject]@{
      target = $target
      image = "opendroneops/$target`:$version"
      archive = $archive
      sha256 = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    }
  }

  [pscustomobject]@{
    version = $version
    commit = $commit
    artifacts = $artifacts
  } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $packageDirectory 'release-manifest.json') -Encoding utf8

  & $verifier -PackageDirectory $packageDirectory

  $manifestPath = Join-Path $packageDirectory 'release-manifest.json'
  $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
  $manifest.artifacts[0].sha256 = ('0' * 64)
  $manifest | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $manifestPath -Encoding utf8

  $tamperDetected = $false
  try {
    & $verifier -PackageDirectory $packageDirectory
  } catch {
    $tamperDetected = $true
  }
  if (-not $tamperDetected) {
    throw 'Release package verifier accepted a tampered archive digest.'
  }

  Write-Output 'Release package verifier tests passed.'
} finally {
  if (Test-Path -LiteralPath $packageDirectory) {
    Remove-Item -LiteralPath $packageDirectory -Recurse -Force
  }
}
