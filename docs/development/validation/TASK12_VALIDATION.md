# Task 12 Validation

## Scope

Task 12 establishes the release-oriented baseline:

- API security headers, bounded request bodies and request IDs;
- injectable readiness probes that distinguish process liveness from dependency
  availability;
- Prometheus-compatible HTTP request counters and duration summaries on a
  separate management listener;
- production configuration guards for MQTT TLS and a loopback-only management
  address;
- a repeatable PowerShell release gate and an operator release checklist;
- a 5,000-point bounded trajectory-query benchmark.

## Validation checklist

| Check | Result |
|---|---|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./cmd/...` | PASS |
| `go test -run '^$' -bench BenchmarkMemoryStoreQuery5000Points -benchtime=1s ./internal/trajectory` | PASS: about 1.08 ms/op on local Windows hardware |
| `npm run typecheck` | PASS |
| `npm test` | PASS: 4 files, 10 tests |
| `npm run build` | PASS |
| `docker compose ... config --quiet` with pinned validation variables | PASS |
| `powershell -ExecutionPolicy Bypass -File scripts/release-check.ps1` | PASS |
| `git diff --check` | PASS |
| `go test -race ./...` | NOT RUN: `CGO_ENABLED=1` requires GCC, which is unavailable on this Windows host |
| Live PostgreSQL/Redis/MQTT readiness probes | NOT RUN: no infrastructure service was started |
| Browser E2E/accessibility run | NOT RUN: the local browser runtime remains unavailable |

## Operational notes

- `/metrics` is available only from the optional admin listener. The public API
  listener does not register that route. In `production`, `ADMIN_ADDR` must be
  loopback-only.
- `X-Request-ID` is preserved only when printable ASCII and at most 128 bytes;
  otherwise the server replaces it. API responses include anti-sniffing,
  anti-framing, no-referrer and no-store headers.
- `/health/live` stays process-only. Named readiness probes may return `503`
  and identify the failing dependency without exposing its error details.
- The benchmark is a regression signal for the bounded in-memory query
  implementation, not a database capacity claim. Production capacity and fault
  exercises remain release-owner evidence in the checklist.
- `npm ci` completed with non-blocking Node engine warnings from transitive
  Babel packages (`v24.5.0` locally versus `>=24.11.0` requested); typecheck,
  tests and production build all still passed.
