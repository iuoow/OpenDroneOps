# Task 5 Validation

- Validation date: 2026-07-26
- Added `internal/mqttworker` with an Eclipse Paho MQTT v5/autopaho transport boundary.
- Subscriptions are explicit and exclude DRC topics.
- Incoming messages are parsed through the DJI Topic/Envelope parser before entering the worker pool.
- Messages are sharded by a stable FNV-1a key so the same gateway/device remains ordered within one shard.
- Every shard has a bounded queue; full queues return `queue_full` without blocking the broker callback.
- In-memory deduplication is used as a Task 5 test/runtime boundary; PostgreSQL-backed deduplication remains part of the persistence tasks.
- Malformed messages and permanent handler failures are quarantined; transient handler failures have finite retry attempts and bounded backoff.
- Worker and broker shutdown paths are context-aware and wait for owned goroutines.

## Checks

- `go mod tidy`: PASS
- `gofmt -w internal/mqttworker cmd/worker`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS
- `go build ./cmd/...`: PASS
- Duplicate, malformed JSON, queue full, transient retry, permanent failure,
  stable deduplication, and close/cancellation tests: PASS
- `go test -race ./...`: remains unavailable in this Windows environment because CGO/GCC is not installed.
- No live MQTT broker was started; broker connection code is compiled and covered by validation boundaries, while runtime integration is deferred to an environment with Mosquitto.
