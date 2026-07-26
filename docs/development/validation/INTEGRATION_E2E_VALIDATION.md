# Live Infrastructure Integration Validation

## Harness

`scripts/integration-e2e.ps1` starts the pinned local Compose services, applies
Goose migrations, then runs `go test -tags=integration ./integration/...` with
explicit connection settings. It does not truncate tables or remove Compose
volumes. The test uses only the fixed E2E workspace and device IDs and removes
those rows on cleanup.

The live test verifies:

- PostgreSQL migration application, latest-state persistence and event
  idempotency;
- Redis write/read of the derived latest-state cache;
- Mosquitto connection, subscription and QoS 1 publish/receive through the
  production Paho adapter.

## Current result

| Check | Result |
|---|---|
| `go test -tags=integration -run '^$' ./integration/...` | PASS: harness compiles |
| `powershell -ExecutionPolicy Bypass -File scripts/integration-e2e.ps1` | BLOCKED: Docker Desktop Linux Engine named pipe is missing |
| `docker version` | BLOCKED: `//./pipe/dockerDesktopLinuxEngine` does not exist |

Docker Desktop processes are present, but the Docker daemon endpoint is not
responding. The E2E script now fails within 20 seconds during a Docker Engine
preflight instead of allowing Compose to hang until an outer command timeout.
No successful live-service result is claimed. After Docker Desktop reports that
its engine is running, rerun the script from the repository root; it stops the
Compose containers on completion while preserving their volumes.
