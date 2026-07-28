# Task 25 Validation

## Scope

Task 25 adds reproducible non-root application image targets, build provenance,
a production application Compose overlay, no-secret release environment
template, a local image-archive packager, and a manually triggered GitHub
candidate workflow. It does not publish a registry image or create a release.

## Automated verification

Run from the repository root:

```powershell
go test ./internal/buildinfo
go test ./...
go vet ./...
go build ./cmd/...
docker build --target server -t opendroneops/server:local .
docker build --target worker -t opendroneops/worker:local .
docker build --target migrate -t opendroneops/migrate:local .
docker compose --env-file deployment/release.env.example -f deployment/docker-compose.release.yaml config --quiet
git diff --check
```

The repository CI also performs the three Docker builds and release Compose
validation in `delivery-contract`. The manual workflow packages checksummed
archives but has no registry write permission.

## Result

Task 25 is complete when the commands above and CI pass. Local Docker build
validation may be deferred when Docker Desktop has insufficient disk space;
the GitHub delivery-contract job remains the authoritative automated build
check until local capacity is restored.
