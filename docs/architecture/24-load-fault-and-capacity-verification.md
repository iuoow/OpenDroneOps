# Load, Fault, and Capacity Verification

Task 24 adds a deterministic, hardware-independent capacity regression harness.
It is an executable contract for bounded behaviour, not a production sizing
claim.

## Run it

From the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/capacity-check.ps1
```

Or call the command directly and retain its JSON stdout as a release artifact:

```powershell
go run ./cmd/capacitycheck -sessions 8 -events 64 -timeout 5s -max-p95 500ms > capacity-report.json
```

The command exits non-zero when a scenario fails. `duration` and
`p95_latency` are Go duration strings in the report; `checks` contains stable
integer counters.

## Regression scenarios

| Scenario | Fault or load | Required result |
|---|---|---|
| `websocket_fanout` | N local sessions × M durable alarms | Every subscribed session receives every event; zero slow-client disconnects; p95 is within `-max-p95`. |
| `websocket_slow_client` | Writer blocked with full queue | Telemetry is coalesced and the subsequent durable alarm disconnects exactly one slow client. |
| `mqtt_hot_key_recovery` | A gateway key fills its per-key pending cap | One `ErrHotKeyBackpressure`; accepted work drains; rejected payload can be retried and processed. |
| `realtime_relay_dedupe` | One event published twice across two in-memory instances | Remote session receives one event and the receiving Relay records one duplicate. |

The harness uses in-memory transports and bus adapters. It intentionally does
not need Docker, Redis, MQTT, PostgreSQL, credentials, firmware, or hardware.
It runs in CI as a correctness and regression gate.

## Acceptance and production evidence

The default `8 × 64` fan-out and `500ms` p95 threshold is a conservative local
regression budget. It is not an SLO and must not be used to infer a production
operator or device limit.

Before a production capacity decision, the release owner must retain a separate
measured report that records:

1. CPU, memory, network, broker, Redis, and PostgreSQL topology;
2. authenticated connection count, message rate, payload mix, and hot-key
   distribution;
3. p50/p95/p99 end-to-end latency, queue rejections, coalescing, reconnects,
   recovery success, and data-loss decision;
4. Redis/MQTT/PostgreSQL fault injection, instance restart, and rollback;
5. exact image digests, configuration limits, command line, and UTC interval.

Only that deployment-specific evidence can tune Task 21 and 22 defaults.
