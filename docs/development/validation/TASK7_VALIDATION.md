# Task 7 Validation

## Scope

Task 7 adds the backend WebSocket event hub boundary:

- workspace-aware connection and subscription authorization;
- a bounded per-session queue with one writer goroutine;
- telemetry coalescing for slow clients;
- isolation of slow clients when durable events cannot be queued;
- snapshot recovery for a new subscription and cursor replay for reconnects;
- an HTTP adapter using `github.com/coder/websocket`.

The implementation intentionally stops before alarm persistence, command delivery,
and the Vue client. Those concerns belong to Tasks 8–10.

## Validation checklist

| Check | Result |
|---|---|
| `gofmt -w internal/websockethub` | PASS |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./cmd/...` | PASS |
| `git diff --check` | PASS |
| `docker compose -f deployment/docker-compose.blueprint.yaml config --quiet` with pinned validation image variables | PASS |
| JSON schema and migration static checks | PASS |
| `go test -race ./...` | NOT RUN: local Windows environment has `CGO_ENABLED=0` and no GCC |
| Live browser/WebSocket integration test | NOT RUN: no broker/authenticated runtime was requested for this task |

## Contract notes

- `Principal.WorkspaceIDs` is the authorization boundary supplied by the
  application authentication layer; the hub never trusts a client-provided
  workspace outside that set.
- `SubscriptionRequest.Channels` is filtered server-side. Session and system
  events remain available for protocol feedback.
- `Event` follows the public WebSocket envelope fields and keeps payloads as
  `json.RawMessage` so protocol adapters do not leak DJI DTOs into the hub.
- A missing recovery provider is allowed for live-only subscriptions, but a
  cursor request fails explicitly with `ErrRecoveryUnavailable`.
- Queue overflow is observable through session closure for durable events;
  telemetry is coalesced by aggregate ID to preserve the newest state.
