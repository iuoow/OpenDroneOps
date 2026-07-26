param(
  [switch]$KeepServices
)

$ErrorActionPreference = 'Stop'
$env:POSTGRES_IMAGE = 'postgres:18.4'
$env:REDIS_IMAGE = 'redis:7.2.14'
$env:MOSQUITTO_IMAGE = 'eclipse-mosquitto:2.1.2'
$env:POSTGRES_PASSWORD = 'integration-only'

docker compose -f deployment/docker-compose.blueprint.yaml up -d --wait
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
