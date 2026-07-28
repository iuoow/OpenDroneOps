# Capacity and Quota Contract

Task 21 establishes explicit, configurable in-process capacity boundaries. The
values below are safe development defaults, not production sizing advice. A
deployment owner must set them from an observed load profile and resource
budget.

## Bounded resources

| Boundary | Configuration | Default | Overload behaviour |
|---|---|---:|---|
| WebSocket outbound messages per session | `WS_SEND_QUEUE_SIZE` | 256 | Telemetry is coalesced by device; a later durable event disconnects the slow client. |
| WebSocket sessions per workspace | `WS_MAX_SESSIONS_PER_WORKSPACE` | 64 | New connection is rejected with `ErrWorkspaceCapacityExceeded`. |
| Device filters per WebSocket subscription | `WS_MAX_DEVICE_FILTERS` | 100 | Subscription is rejected with `ErrSubscriptionTooBroad`. |
| Recent event IDs per WebSocket session | `WS_EVENT_DEDUPE_CAPACITY` | 2048 | Duplicate live/recovery overlap is suppressed for the session. |
| MQTT ingestion shards | `MQTT_SHARD_COUNT` | 32 | Fixed worker partition count. |
| MQTT messages queued per shard | `MQTT_SHARD_QUEUE_SIZE` | 1024 | Ingestion remains bounded; the worker reports its existing queue-pressure outcome. |
| Trajectory query | API query contract | bounded | Existing time-window and item-count validation rejects oversized reads. |

Zero values in the `websockethub.Config` are compatibility defaults (64 and
100). Negative values are invalid. Environment configuration is stricter: all
capacity values must be positive at process startup.

## Observability

The metrics registry exposes this low-cardinality counter:

```text
opendroneops_capacity_events_total{component="websocket",outcome="..."}
```

The current WebSocket outcomes are:

- `workspace_session_limit`
- `device_filter_limit`
- `telemetry_coalesced`
- `slow_client_disconnect`

The WebSocket composition root must provide the process `observability.Registry`
as `websockethub.Config.CapacityObserver`; the registry intentionally accepts
the interface without coupling the Hub to a metrics implementation. Do not use
workspace IDs, device IDs, user IDs, or request IDs as metric labels.

## Operational policy

1. Alert on any sustained `workspace_session_limit` or
   `slow_client_disconnect` events.
2. Treat repeated `device_filter_limit` events as a client-query design issue;
   use multiple scoped subscriptions only when the session quota permits it.
3. Size per-workspace session capacity from authenticated operator concurrency,
   then reserve headroom for reconnect bursts.
4. Tune queue and shard limits only with a load/fault result that records
   memory, latency, disconnects, and recovery behaviour.

This Task does not implement global multi-instance quotas, broker-side limits,
tenant billing, or load-balancer affinity. Those are later scale work and must
preserve these stable local rejection semantics.
