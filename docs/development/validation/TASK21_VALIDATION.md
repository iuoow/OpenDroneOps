# Task 21 Validation

## Scope

Task 21 turns the existing bounded MQTT, WebSocket, and trajectory behaviour
into an explicit capacity contract. It adds configurable WebSocket workspace
session and device-filter quotas, stable rejection errors, low-cardinality
capacity counters, and a public operating contract.

## Automated verification

Run from the repository root:

```powershell
go test ./internal/config ./internal/observability ./internal/websockethub
go test ./...
git diff --check
```

The focused Hub tests verify that a second session above the workspace quota is
rejected, an oversized device filter list is rejected, statistics reflect both
outcomes, and capacity observers receive stable events. Configuration tests
verify defaults and startup rejection of non-positive quota values.

## Result

Task 21 is complete when the commands above pass. It does not validate a
production capacity number: deployment owners must tune the documented limits
using a later load/fault exercise. It also does not add multi-instance global
quota enforcement, which remains future scale work.
