$ErrorActionPreference = 'Stop'

# This check is intentionally read-only apart from normal compiler/test caches.
# It does not start Docker services and does not publish artifacts.
go test ./...
go vet ./...
go build ./cmd/...
powershell -ExecutionPolicy Bypass -File scripts/capacity-check.ps1
powershell -ExecutionPolicy Bypass -File scripts/test-release-package-verifier.ps1

npm.cmd --prefix web ci
npm.cmd --prefix web run typecheck
npm.cmd --prefix web test
npm.cmd --prefix web run build

$env:POSTGRES_IMAGE = 'postgres:18.4'
$env:REDIS_IMAGE = 'redis:7.2.14'
$env:MOSQUITTO_IMAGE = 'eclipse-mosquitto:2.1.2-alpine'
$env:POSTGRES_PASSWORD = 'release-check-only'
docker compose -f deployment/docker-compose.blueprint.yaml config --quiet

$env:OPENDRONEOPS_SERVER_IMAGE = 'opendroneops/server:release-check'
$env:OPENDRONEOPS_WORKER_IMAGE = 'opendroneops/worker:release-check'
$env:OPENDRONEOPS_MIGRATE_IMAGE = 'opendroneops/migrate:release-check'
$env:RELEASE_ENV_FILE = './release.env.example'
docker compose --env-file deployment/release.env.example -f deployment/docker-compose.release.yaml config --quiet

git diff --check
Write-Output 'OpenDroneOps release checks passed.'
