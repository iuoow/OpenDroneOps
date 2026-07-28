# Task 17 Validation

## Scope

Task 17 connects the Pilot Shell to a dedicated read-only state layer. The
layer loads a workspace-scoped snapshot of devices and open alarms, subscribes
only to telemetry/state/alarm WebSocket channels, applies ordered device
updates, ignores cross-workspace events, advances the recovery cursor, and
surfaces loading, stale, disconnected, recovering, and restored states.

Pilot does not expose command creation, alarm acknowledgement, or any other
mutation through this state layer.

## Automated verification

Executed from `web/`:

- `npm.cmd test -- --run` — 8 test files, 30 tests passed.
- `npm.cmd run typecheck` — passed.
- `npm.cmd run build` — passed; both Operations and Pilot entry points built.
- `git diff --check` — passed.
- `rg -n "getPilotSnapshot|channels: \['telemetry', 'state', 'alarm'\]|createCommand|acknowledgeAlarm" web/src/pilot web/src/api web/src/realtime` — verified the Pilot read model uses the read-only snapshot and excludes command subscription/mutation calls.

## Covered behaviors

- Snapshot loading requests `/devices`, open `/alarms`, and one event cursor;
  it does not request `/commands`.
- WebSocket subscription is scoped to `telemetry`, `state`, and `alarm`.
- Newer device state versions replace cached state; older versions and events
  from another workspace are ignored.
- A valid event advances the cursor and changes a recovering/disconnected
  connection to restored/connected.
- A snapshot becomes stale after the configured freshness window and the UI
  displays a warning state.
- Snapshot failures expose a stable retryable message without forwarding raw
  backend error details.
- The Home, Device, Alerts, and More views render only read-only information;
  command and alarm mutation controls are absent.

## Result

Task 17 is complete for the Pilot Mock Bridge/foundation scope. Real DJI
integration remains gated by ADR 0013 and Task 20's external prerequisites.
