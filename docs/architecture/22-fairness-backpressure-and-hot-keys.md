# Fairness, Backpressure, and Hot-Key Isolation

Task 22 protects the MQTT ingestion path from a single device or gateway
monopolising a shard. It applies only within one process; global quota and
multi-instance delivery are out of scope.

## Scheduling contract

Each MQTT shard owns a bounded fair queue.

- The fairness key is `device:<serial>` for device topics and
  `gateway:<serial>` for gateway topics.
- Messages for the same key remain FIFO.
- Active keys are dequeued round-robin: after one message is selected, a key
  with more pending work moves to the end of the ready list.
- Queue memory is bounded by `MQTT_SHARD_QUEUE_SIZE` per shard.
- `MQTT_MAX_PENDING_PER_KEY` bounds a single key inside that shard. Its
  effective value is `min(MQTT_MAX_PENDING_PER_KEY, MQTT_SHARD_QUEUE_SIZE)`;
  the development default is 64.

## Backpressure contract

| Condition | Stable result | Required caller behaviour |
|---|---|---|
| One key reaches its pending cap | `ErrHotKeyBackpressure` | Record the rejection; let the MQTT client/broker delivery policy apply rather than treating the message as processed. |
| Whole shard reaches its cap | `ErrQueueFull` | Record the rejection and apply the same broker-side flow-control policy. |
| Worker stops | `ErrClosed` | Stop accepting new input. |

Ingress parses before admission but only marks a message in the deduplicator
when a shard worker begins processing it. Therefore a rejected message is not
silently converted into a duplicate if the broker later redelivers it.

Malformed messages continue to be quarantined. Duplicate messages can enter a
bounded queue during a burst, then are discarded before handler execution;
this intentionally favours correct redelivery semantics over an irreversible
dedup mark before admission.

## Observability

`mqttworker.Stats` exposes `QueueFull` and `HotKeyBackpressure`. A composition
root may supply the process metrics registry as `mqttworker.Config.CapacityObserver`.
It records low-cardinality events in the existing counter:

```text
opendroneops_capacity_events_total{component="mqtt_ingestion",outcome="hot_key_limit"}
opendroneops_capacity_events_total{component="mqtt_ingestion",outcome="shard_queue_limit"}
```

Do not add gateway, device, topic, or message identifiers as metric labels.

## Non-goals

This contract does not guarantee fair processing across separate process
instances, rate-limit an upstream broker, retain rejected payloads, or change
the command Outbox delivery contract. Task 23 will address multi-instance
realtime recovery; Task 24 will provide load and fault verification.
