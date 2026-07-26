# Task 3 Validation

- Validation date: 2026-07-26
- Added `internal/protocol/dji` for DJI-compatible topic parsing and envelope decoding.
- Supported topics: OSD, State, Services, Services Reply, Events, Events Reply, Requests, Requests Reply, and system Status.
- DRC topics are intentionally rejected because DRC remains outside the MVP.
- Envelope decoding accepts boolean or `0/1` `need_reply` values, preserves unknown fields, and requires `data`.
- Parsed messages retain the original topic and a SHA-256 payload hash for deduplication/audit boundaries.

## Checks

- `gofmt -w internal/protocol/dji`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS
- `go build ./cmd/...`: PASS
- Table-driven topic, malformed-message, unknown-field, `need_reply`, and payload-hash tests: PASS
- `go test -race ./...`: remains unavailable in this Windows environment because CGO/GCC is not installed.
- No MQTT broker was started; Task 3 is parser-only.
