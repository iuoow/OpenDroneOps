# Task 23 Validation

## Scope

Task 23 adds a versioned cross-instance Relay, Redis Pub/Sub adapter, bounded
relay/session deduplication, and unique production instance identity
validation. Pub/Sub accelerates live delivery; existing Snapshot/Replay
recovery is kept as the correctness path.

## Automated verification

Run from the repository root:

```powershell
go test ./internal/config ./internal/websockethub ./internal/realtime
go test ./...
git diff --check
```

The in-memory bus test starts two independent Hubs/Relays, delivers an alarm
from one to the other, then publishes the same event again and verifies one
session delivery plus one duplicate suppression. Additional tests cover invalid
events, bus publish failure accounting, production node-ID validation, and
overlap between recovery and live Hub delivery.

## Result

Task 23 is complete when the commands above pass. No Redis container is needed
for the unit contract, but a deployment must run a Redis-backed integration
exercise before enabling multiple Hub instances. Redis Pub/Sub alone must never
be presented as durable recovery.
