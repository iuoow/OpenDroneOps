# Overview v2 Detailed Specification

## Job to be done

The desktop Overview answers, in this order: what needs attention, where it
is, whether the evidence is fresh, and what safe investigation is available.
It is not a general analytics dashboard or a command centre for real flight
control.

## Wide desktop composition

```text
Top bar: Workspace | global search | realtime evidence | active critical count | user
Side nav: Overview / Devices / Alarms / Commands / Replay / Operations

┌── Attention rail (optional, 280 px) ─┬── Spatial workspace (fluid) ────────┬── Context pane (360–440 px) ┐
│ Critical / warning queue             │ Neutral MapLibre map                 │ Preview or pinned investigation     │
│ Device health / saved focus          │ clusters, symbols, layers, scale     │ selected device or incident          │
│ keyboard-equivalent of map selection │ map controls and viewport summary     │ evidence, freshness, next action     │
└──────────────────────────────────────┴──────────────────────────────────────┴────────────────────────────────────┘
Event strip: selected-object events; collapsed by default when no selection.
```

The attention rail is visible when there are active warnings/critical events or
when the operator pins it. Otherwise the map receives the freed space. A
desktop page never displays an empty permanent alert panel just to preserve a
three-column composition.

## Regions and behaviour

| Region | Default content | Selection behaviour | Empty / degraded behaviour |
|---|---|---|---|
| Top bar | Workspace, search, browser realtime state, active critical count | Critical count opens Alarms with the selected filter | Realtime state explains snapshot-only or recovery mode |
| Attention rail | Critical first, then warnings, then pinned devices | Select highlights map and opens matching context | Collapses when no actionable items; shows an explicit empty state when manually opened |
| Spatial workspace | Base map, device clusters, current tails, critical rings, selected halo | Device click previews; map list and keyboard selection do the same | Preserve base map; show layer-local loading/error instead of blocking the shell |
| Context pane | Nothing until selection, then a compact preview | `Pin` expands durable investigation; close clears selection but preserves viewport | Local skeleton, data-quality notice, or empty detail—never a whole-page error |
| Event strip | Selected device/alarm/command events only | Selecting event opens its durable detail route | Hidden without selection; no high-frequency telemetry feed |

## Selected device preview

The first selection opens a preview, not a long dashboard card. Its fixed order
is:

1. Device identity, type, presence, and freshness.
2. Four primary telemetry values appropriate to the device type; remaining
   values are behind "更多遥测".
3. The highest active health condition, with its reason and evidence time.
4. One primary safe action: `查看事件`, `查看轨迹`, or `刷新状态`, depending on
   context. Other safe actions belong to `更多操作`.
5. `固定详情` creates/updates the URL-addressable durable detail state.

No action in this preview can imply takeoff, land, return-to-home, DRC, or real
hardware authority.

## Incident handling from Overview

1. A critical event updates the attention rail and map ring once.
2. Selecting it shows condition, impacted device, first/latest evidence,
   duplicate count, freshness, and recommended investigation.
3. `确认接手此告警` records acknowledgement; it does not suppress the map or
   mark the condition resolved.
4. `在告警中心查看` moves to the durable queue/detail workflow only when the
   operator chooses it.

## Breakpoints

| Viewport | Behaviour |
|---|---|
| `>= 1440px` | Attention rail, map, and context pane may coexist. |
| `1200–1439px` | Collapsed icon nav; attention rail becomes an on-demand drawer. |
| `1024–1199px` | Map stays primary; context and attention are mutually exclusive drawers. |
| `< 1024px` | Desktop overview does not masquerade as Pilot. Present a compact read-only summary and direct field workflows to Pilot Shell. |

## Accessibility and performance acceptance

- The attention rail is the keyboard-equivalent path to every selected map
  object and active incident.
- Map selection, rail selection, and context preview share one selected-object
  state and one accessible name.
- Critical state uses text, icon/shape, timestamp, and colour.
- Selection changes are announced concisely; high-frequency telemetry is not.
- Marker updates are batched; the map does not recenter on ordinary telemetry.
- Motion is limited to the existing motion tokens and disabled under reduced
  motion preferences.
