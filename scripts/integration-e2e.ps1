param(
  [switch]$KeepServices
)

$ErrorActionPreference = 'Stop'

function Assert-DockerEngine {
  $outputPath = Join-Path $env:TEMP ("opendroneops-docker-" + [guid]::NewGuid().ToString() + ".out")
  $errorPath = Join-Path $env:TEMP ("opendroneops-docker-" + [guid]::NewGuid().ToString() + ".err")
  try {
    $process = Start-Process -FilePath 'docker' -ArgumentList @('version', '--format', '{{.Server.Version}}') -PassThru -NoNewWindow -RedirectStandardOutput $outputPath -RedirectStandardError $errorPath
    if (-not $process.WaitForExit(20000)) {
      Stop-Process -Id $process.Id -Force
      throw 'Docker Engine did not respond within 20 seconds. Restart Docker Desktop and wait for Engine Running.'
    }
    if ($process.ExitCode -ne 0) {
      $details = (Get-Content -LiteralPath $errorPath -Raw -ErrorAction SilentlyContinue).Trim()
      throw "Docker Engine is unavailable: $details"
    }
  }
  finally {
    Remove-Item -LiteralPath $outputPath, $errorPath -Force -ErrorAction SilentlyContinue
  }
}

Assert-DockerEngine
$env:POSTGRES_IMAGE = 'postgres:18.4'
$env:REDIS_IMAGE = 'redis:7.2.14'
$env:MOSQUITTO_IMAGE = 'eclipse-mosquitto:2.1.2'
$env:POSTGRES_PASSWORD = 'integration-only'

docker compose -f deployment/docker-compose.blueprint.yaml up -d --wait --wait-timeout 45
try {
  $env:POSTGRES_DSN = 'postgres://opendroneops:integration-only@localhost:5432/opendroneops?sslmode=disable'
  $env:REDIS_ADDR = 'localhost:6379'
  $env:REDIS_PASSWORD = ''
  $env:MQTT_URL = 'mqtt://localhost:1883'
  $env:INTEGRATION_TEST = '1'
  go test -tags=integration ./integration/...
}
finally {
  if (-not $KeepServices) {
    docker compose -f deployment/docker-compose.blueprint.yaml down
  }
}
