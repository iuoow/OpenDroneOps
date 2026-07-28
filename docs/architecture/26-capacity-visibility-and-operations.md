# Capacity Visibility and Operations Experience

Task 26 turns the low-cardinality capacity counters from Tasks 21-24 into a
safe management-plane summary for an operator on call.

## Surfaces and access boundary

| Surface | Audience | Purpose |
|---|---|---|
| `GET /metrics` on `ADMIN_ADDR` | Monitoring system | Full Prometheus text counters and rate-based alerting. |
| `GET /capacity` on `ADMIN_ADDR` | Operator / runbook tooling | Small JSON snapshot with health, severity, counts, and recommended actions. |
| `scripts/capacity-status.ps1` | Human operator | Readable terminal table for the `/capacity` snapshot. |

`/capacity` is deliberately **not** part of `/api/v1` and must never be
mounted on the public application listener. In production `ADMIN_ADDR` is
loopback-only. Use an approved local terminal, SSH port-forward, or managed
monitoring agent to access it.

```powershell
powershell -ExecutionPolicy Bypass -File scripts/capacity-status.ps1
```

Use `-AdminUrl` after an approved port-forward. `-FailOnAttention` returns
non-zero for warning-level health and is suitable for a supervised probe.

## Health model

The snapshot contains cumulative counts since process start. It is a triage
view, not a rate calculation:

| Health | Meaning | Immediate action |
|---|---|---|
| `healthy` | No capacity outcomes recorded since start. | Continue normal monitoring. |
| `attention` | A warning outcome occurred. | Follow the event recommendation and inspect the Prometheus rate/trend. |
| `critical` | Ingress shard capacity was exhausted or cross-instance publish failed. | Stabilise ingress/realtime dependency, verify recovery, and open an incident if sustained. |

The current critical outcomes are `mqtt_ingestion:shard_queue_limit` and
`realtime:publish_failure`. Workspace/session limits, slow-client disconnects,
hot keys, invalid realtime envelopes, and subscription limits require
attention. Telemetry coalescing and duplicate suppression are informational,
but their rate may still indicate an unhealthy trend.

## Monitoring guidance

Alerting must consume the Prometheus counter by rate, for example over five
minutes, rather than alerting solely because a process-lifetime count is
non-zero. Useful controlled labels include:

```text
opendroneops_capacity_events_total{component="mqtt_ingestion",outcome="shard_queue_limit"}
opendroneops_capacity_events_total{component="realtime",outcome="publish_failure"}
opendroneops_capacity_events_total{component="websocket",outcome="slow_client_disconnect"}
```

Never add workspace, user, device, topic, event, or request identifiers as
metric labels. For a restart, compare the new process start timestamp with the
monitoring time series so counter resets are not mistaken for recovery.

## Boundaries

This task supplies a process-local status summary and operator guidance. It
does not add tenant-visible capacity data, bypass management-plane access
controls, create an incident-management integration, or convert the ephemeral
Redis Pub/Sub path into durable recovery.
