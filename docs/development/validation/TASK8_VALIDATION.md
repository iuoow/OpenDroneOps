# Task 8 Validation

## Scope

Task 8 adds the backend alarm lifecycle:

- deterministic offline, low-battery, and protocol-alarm rules;
- one active alarm per `(workspace_id, dedup_key)`;
- occurrence counting and severity escalation without duplicate active rows;
- explicit acknowledgement and resolution transitions;
- recovery of active alarms from PostgreSQL or the in-memory test store;
- `alarm.created`, `alarm.updated`, and `alarm.resolved` WebSocket notifications;
- PostgreSQL constraints for supported severity, status, and positive occurrence count.

Duplicate raw MQTT delivery is expected to be filtered by the ingestion/event
idempotency boundary. Repeated valid observations for the same active dedup key
remain intentionally visible through `occurrence_count`.

## Validation checklist

| Check | Result |
|---|---|
| `gofmt -w internal/alarm` | PASS |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./cmd/...` | PASS |
| `git diff --check` | PASS |
| Goose migration markers and alarm constraints | PASS |
| JSON Schema parsing and OpenAPI alarm-contract inspection | PASS |
| `docker compose -f deployment/docker-compose.blueprint.yaml config --quiet` with pinned validation image variables | PASS |
| `go test -race ./...` | NOT RUN: local Windows environment has `CGO_ENABLED=0` and no GCC |
| Live PostgreSQL/API integration | NOT RUN: no database service was started for this task |

## Contract notes

- PostgreSQL remains the source of truth; the memory store exists only for
  deterministic unit tests and local composition.
- The partial unique index already present on `alarms` prevents more than one
  open/acknowledged row for a workspace and dedup key. Migration `00002` adds
  explicit enum-like checks.
- Acknowledgement is idempotent. Resolution is idempotent and removes the
  dedup key from the active set, allowing a later condition to create a new
  alarm row.
- Alarm events carry the public WebSocket envelope and serialized domain alarm
  data. The publisher is optional so persistence can be tested independently.
