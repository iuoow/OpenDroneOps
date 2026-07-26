# Task 11 Validation

## Scope

Task 11 adds bounded trajectory history:

- a workspace/device scoped trajectory query store with stable cursor pagination;
- a PostgreSQL query implementation ordered by `(occurred_at, id)`;
- explicit 24-hour and 5,000-point limits instead of silent unbounded scans;
- coordinate, speed, heading, battery and source-event constraints;
- a REST route at `/api/v1/devices/{device_id}/trajectory`;
- a Vue replay screen with history-mode labeling, map path, playback controls,
  telemetry, event synchronization, URL range state and a realtime return path;
- Web Worker simplification for large tracks.

## Validation checklist

| Check | Result |
|---|---|
| `gofmt -w internal/trajectory internal/httpapi/server.go` | PASS |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./cmd/...` | PASS |
| `npm run typecheck` | PASS |
| `npm test` | PASS: 4 files, 10 tests |
| `npm run build` | PASS: replay worker emitted as a separate asset |
| Cursor, query-bound, handler and worker helper tests | PASS |
| Goose migration markers and trajectory constraints | PASS |
| Live PostgreSQL trajectory query | NOT RUN: no database service was started |
| In-app browser visual inspection | NOT RUN: browser runtime path remains unavailable |

## Contract notes

- Missing `from`/`to` defaults to a one-hour window; clients can request up to
  24 hours per call.
- `truncated=true` is explicit and returns `next_cursor`; it is never inferred
  from a partial client render.
- The replay UI labels historical mode and keeps its snapshot when a query
  fails. It does not merge historical points into the realtime device state.
- A Worker is used only for rendering simplification; the query cursor and
  server-side limit remain authoritative.
