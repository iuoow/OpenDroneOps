# Task 22 Validation

## Scope

Task 22 replaces each MQTT shard's plain FIFO channel with a bounded per-key
round-robin scheduler. It enforces a per-key pending cap, records overload
outcomes, and defers duplicate marking until processing starts so a rejected
message remains eligible for redelivery.

## Automated verification

Run from the repository root:

```powershell
go test ./internal/config ./internal/mqttworker ./internal/observability
go test ./...
git diff --check
```

The focused tests verify round-robin order across a hot and a cool key, stable
hot-key rejection and observability, and successful processing when the same
message is retried after its prior rejection. Existing worker tests cover
bounded admission, malformed-message quarantine, retries, and shutdown.

## Result

Task 22 is complete when the commands above pass. Capacity values remain
development defaults and need a measured load/fault result before production
sizing. Cross-instance fairness, broker flow-control tuning, and retained
rejected payloads remain later work.
