# AGENTS.md

## Mission

Build OpenDroneOps incrementally according to this repository. Preserve protocol correctness and keep it runnable without real DJI hardware.

## Read first

Before editing code, read:

1. `README.md`
2. `docs/development/task-sequence.md`
3. `docs/architecture/14-roadmap.md`
4. Relevant `docs/`
5. Relevant `docs/decisions/`
6. `docs/development/`

Conflict priority:

1. Current DJI official documentation for external protocol behavior
2. Repository contracts under `docs/`, `api/`, and `schemas/`
3. ADRs
4. Existing code
5. Assumptions

Report unresolved conflicts instead of silently choosing.

## Working rules

- Implement one roadmap task at a time; record the result in `docs/development/validation/`.
- Do not implement later phases opportunistically.
- Keep the repository buildable and testable.
- Before adding a production dependency, document purpose, license, pinned version, and alternatives.
- Never use a Git main branch or unpinned container `latest` tag.
- Never commit secrets, DJI credentials, private keys, production URLs, or real serial numbers.
- Use English code identifiers; Chinese documentation/UI is allowed.
- Prefer explicit SQL, small interfaces, and deterministic tests.
- Avoid global mutable state.
- Every goroutine needs an owner, cancellation path, bounded resources, and shutdown behavior.
- Every queue needs a capacity and full-queue policy.
- Every retry needs retryable-error rules, a deadline/attempt limit, exponential backoff, jitter, and idempotency.
- Do not copy DJI Demo code. Implement from official protocol contracts.

## Architecture constraints

- Start as a modular monolith plus separate MQTT worker and simulator.
- DJI protocol DTOs stay inside the DJI adapter.
- Domain modules must not import DJI DTO packages.
- PostgreSQL is the source of truth for device registry, commands, alarms, permissions, audit, and outbox.
- Redis is disposable derived state.
- Use Transactional Outbox for database-to-MQTT/event publication.
- Use REST snapshots plus WebSocket increments.
- Do not add Kafka, NATS, ClickHouse, Elasticsearch, MinIO, Kubernetes, DRC, or video in the MVP.

## DJI rules

- Parse MQTT topics with a dedicated tested parser.
- Distinguish `device_sn` from `gateway_sn`.
- Preserve `tid`, `bid`, `timestamp`, `gateway`, `method`, `need_reply`, and `seq` when present.
- Map internal `command_id` to DJI `tid/bid`.
- Treat QoS 1 messages as potentially duplicated.
- MQTT publish acknowledgement never means device execution success.
- Unknown fields and methods must not crash consumers.
- Preserve raw metadata with retention and redaction controls.

## Go rules

- `context.Context` is the first parameter for request-scoped work.
- Use wrapped errors and `errors.Is/As`.
- Use structured logs.
- Avoid panic for expected failures.
- Run `go test -race` for concurrent components.
- Avoid arbitrary sleeps in tests.
- Handlers translate protocols; services own business logic.
- Network, database, and Redis operations require deadlines.

## WebSocket rules

- One application writer loop per connection.
- Send queues are bounded.
- Telemetry can be coalesced.
- Alarm/command results must be recoverable through REST/event replay.
- Slow clients must not block global broadcasting.
- Reconnect uses snapshot plus cursor.

## Verification

Maintain commands equivalent to:

```bash
make fmt
make lint
make test
make test-race
make test-integration
make build
make compose-config
```

Report commands, results, changed files, limitations, and next task before declaring completion.


## UI/UX rules

- Read `docs/design/ui/`, `docs/design/`, and `docs/development/uiux-tasks.md` before frontend work.
- Desktop and Pilot use separate shells but shared domain contracts and design tokens.
- Do not hardcode new semantic colors, spacing, shadows, radii, or z-index outside tokens.
- Distinguish browser WebSocket, platform MQTT, device online state, and telemetry freshness.
- Use REST snapshots plus WebSocket increments and cursor recovery.
- Coalesce telemetry; keep alarms and command results recoverable.
- Do not create one toast per telemetry or repeated alarm.
- Batch high-frequency state updates per animation frame where practical.
- Use MapLibre layers and clustering instead of large numbers of HTML markers.
- Make realtime and historical modes unmistakable.
- Meet WCAG 2.2 AA.
- Pilot touch targets should be at least 44x44 CSS pixels.
- Implement loading, empty, error, stale, disconnected, and partial-failure states.
- No DRC, real flight controls, automatic takeoff, or dangerous UI actions in MVP.
