# Trajectory Query Contract

## Endpoint

`GET /api/v1/devices/{device_id}/trajectory`

The caller supplies the workspace through `X-Workspace-ID`. The endpoint returns
chronologically ordered points and a stable cursor:

```json
{
  "items": [
    {
      "id": "point-001",
      "workspace_id": "workspace-001",
      "device_id": "aircraft-001",
      "occurred_at": "2026-07-26T10:00:00Z",
      "received_at": "2026-07-26T10:00:01Z",
      "latitude": 31.2304,
      "longitude": 121.4737,
      "altitude": 86,
      "speed": 9.4,
      "heading": 180,
      "battery_percent": 74
    }
  ],
  "next_cursor": "opaque",
  "truncated": true,
  "from": "2026-07-26T09:00:00Z",
  "to": "2026-07-26T10:00:00Z",
  "limit": 500
}
```

## Bounded query rules

- `from` and `to` are RFC3339 timestamps. If omitted, the server uses the last
  hour. `from` is inclusive and `to` is exclusive.
- A query window may not exceed 24 hours.
- The default page size is 500 points; the maximum is 5,000. Values outside
  `1..5000` are rejected rather than silently expanded.
- Pagination uses an opaque base64 cursor containing the last
  `(occurred_at, id)` pair. The ordering is stable even when points share a
  timestamp.
- PostgreSQL remains the source of truth. The frontend may simplify a page for
  rendering, but it must not change the query boundary or cursor semantics.
- `trajectory_points` validates coordinates, speed, heading and battery ranges
  and deduplicates points linked to the same source event.

The replay screen uses the same `playbackTime` for its map, telemetry metrics
and event list. Large result pages are simplified in a Web Worker, while the
raw server result remains bounded and recoverable through the cursor.
