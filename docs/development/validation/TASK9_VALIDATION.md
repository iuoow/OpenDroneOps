# Task 9 Validation

## Scope

Task 9 adds the reliable command path:

- method registry with `sim_status_refresh` as the only default LOW-risk method;
- canonical request hashing and workspace-scoped Idempotency-Key handling;
- transactional Command + CommandEvent + Outbox + audit creation;
- bounded Outbox leases with worker ownership, exponential backoff, jitter,
  finite attempts, and a high-risk no-retry guard;
- MQTT publish acknowledgement advancing only to `PUBLISHED`;
- DJI `services_reply` correlation by workspace, `tid`, `bid`, and method;
- idempotent Reply transitions, orphan Reply persistence, and deadline timeout;
- WebSocket `command.updated` notification hooks for command lifecycle changes.

The existing MQTT worker remains the transport ingestion boundary. A production
composition can pass `mqttworker.MQTTBroker.Publish` to `OutboxPublisher` and
connect `Service.NotifyTransition` to the WebSocket Hub.

## Validation checklist

| Check | Result |
|---|---|
| `gofmt -w internal/command internal/mqttworker/paho.go` | PASS |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./cmd/...` | PASS |
| `git diff --check` | PASS |
| Goose markers and command/outbox/orphan-reply constraints | PASS |
| JSON Schema and OpenAPI command-contract inspection | PASS |
| `docker compose -f deployment/docker-compose.blueprint.yaml config --quiet` with pinned validation image variables | PASS |
| `go test -race ./...` | NOT RUN: local Windows environment requires CGO/GCC |
| Live PostgreSQL/MQTT integration | NOT RUN: no broker or database service was started |

## Contract notes

- A successful MQTT PUBACK is never interpreted as device execution success.
- Replies that arrive before the publisher records `PUBLISHED` first repair that
  state in the same command transaction, then apply the device result.
- Terminal commands ignore duplicate replies; conflicting late replies do not
  reopen a command.
- Outbox completion and retry require the current lease owner, preventing a
  stale worker from overwriting a newer lease.
- Unknown or unmatched `tid`/`bid` replies are persisted in
  `orphan_command_replies` and are never silently discarded.
