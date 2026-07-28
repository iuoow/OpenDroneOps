# Multi-Instance Realtime and Recovery

Task 23 defines the single-process-to-multi-instance transition for realtime
WebSocket delivery. It intentionally separates fast fan-out from correctness.

## Delivery model

```text
domain event -> Relay -> local WebSocket Hub
                     -> Redis Pub/Sub -> Relay on other instance -> local Hub

reconnect / instance switch -> authorized Snapshot or Replay provider -> Hub
```

`internal/realtime.Relay` emits versioned envelopes on a named Redis Pub/Sub
channel. Each instance has a required `REALTIME_NODE_ID`; a relay ignores its
own messages and keeps a bounded recent `(origin,event_id)` set to tolerate
at-least-once delivery from a bus adapter.

Redis Pub/Sub is deliberately treated as **ephemeral**. It does not provide a
durable event log, acknowledgement, or replay guarantee. It is only the live
fan-out path. The existing authorized `websockethub.RecoveryProvider` remains
authoritative for a reconnect, an instance change, a Pub/Sub interruption, or
any cursor-based catch-up.

## Configuration

| Setting | Development default | Purpose |
|---|---:|---|
| `REALTIME_NODE_ID` | `opendroneops-local` | Instance identity; production must set a unique value. |
| `REALTIME_CHANNEL` | `opendroneops:realtime:v1` | Redis Pub/Sub channel for version 1 envelopes. |
| `REALTIME_DEDUPE_CAPACITY` | 4096 | Recent inbound relay IDs retained per instance. |
| `WS_EVENT_DEDUPE_CAPACITY` | 2048 | Recent event IDs retained per WebSocket session. |

The process composition root must create a Redis client, pass it to
`realtime.NewRedisBus`, create one Relay per Hub, and call `Start` before
accepting WebSocket sessions. Domain publishers can depend on the Relay's
existing `Publish(websockethub.Event)` method. A shutdown must close the Relay
before closing its Redis client and Hub.

## Duplicate and recovery semantics

- Relay validation rejects envelopes without a protocol version, origin,
  `event_id`, or workspace.
- A receiving Relay suppresses duplicate `(origin,event_id)` values within its
  bounded dedupe window.
- A WebSocket session suppresses duplicate `event_id` values within its own
  bounded window. This covers overlap between live Pub/Sub and a Snapshot or
  Replay response.
- An event ID is recorded only after successful session admission; a rejected
  durable event is not incorrectly considered delivered.
- An expired cursor still follows the Hub's existing `ErrCursorExpired`
  contract; clients must request a snapshot.

## Operational limits and non-goals

Alert on `opendroneops_capacity_events_total` outcomes under component
`realtime`: `publish_failure`, `invalid_message`, and `duplicate_event`.
These labels are controlled vocabulary; never add tenant, device, user, or
event IDs.

This Task does not make Redis Pub/Sub durable, implement a distributed session
quota, expose a production Hub composition root, or configure load balancer
affinity. Durable fan-out choices and load/fault sizing remain later work.
