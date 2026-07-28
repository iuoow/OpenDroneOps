# Operations Console Refresh Baseline

## Status and scope

This is the design baseline for the post-MVP UI refresh. It is a design
contract only: it does not change routes, API contracts, hardware capability,
or safety authority. Implementation starts only after this baseline and the
overview specification have been reviewed.

The product is a calm, map-first operations console. It is neither a generic
CRUD back office nor a decorative "command-center" screen. A person should be
able to identify what needs attention, where it is, how fresh the evidence is,
and the safe next step without reading a wall of cards.

## What stays

- The six desktop areas: Overview, Devices, Alarms, Commands, Replay, and
  Operations.
- A separate, read-only, touch-first Pilot Shell governed by ADR 0013.
- MapLibre as the spatial canvas, REST snapshot plus WebSocket recovery, and
  the existing URL-addressable detail model.
- The status, accessibility, low-motion, and no-real-flight-control boundaries
  already defined in `00-ui-ux-principles.md`.

## What changes

| Area | Current tendency | Refresh decision |
|---|---|---|
| Visual hierarchy | Many equally weighted panels and controls | One primary workspace, one contextual side pane, and secondary information on demand. |
| Map | Demonstrative canvas treatment | A real neutral basemap, purposeful layers, clusters, meaningful symbols, and an equivalent list path. |
| Device selection | Full detail competes with overview | Preview first; pin or expand only when deeper investigation is needed. |
| Alerts | Another card or queue among many | An event that can deliberately take over the working context without stealing text focus. |
| Buttons | Repeated blue actions | One primary action per context; safe secondary actions; additional actions in a labelled menu. |
| Status | Often inferred from a coloured dot | Icon, explicit label, evidence time, and explanation; colour is supporting evidence only. |
| Responsive UI | Desktop content compressed for smaller screens | Desktop is dense and spatial; Pilot is task-focused and touch-first. |

## Reference patterns adopted deliberately

- Fleet operators benefit from a coordinated fleet, media, and map workspace,
  with prioritised assets rather than every asset demanding the same visual
  weight. This informs the optional focus pane and future media drawer, not a
  promise that video is currently available. [FlytBase Fleet View](https://docs.flytbase.com/in-flight-modules/how-to-manage-your-flight-operations/multi-view-dashboard)
- A selected live source should recenter and gain context rather than force a
  navigation change. This informs map-to-detail and detail-to-map behaviour.
  [DJI FlightHub 2 livestream management](https://fh.dji.com/user-manual/en/real-time-project-information/multi-stream.html)
- Flight-facing UI should make readiness and safety explanations visible before
  exposing context-sensitive actions. This informs the state language and does
  not introduce QGroundControl actions into OpenDroneOps. [QGroundControl Fly View](https://docs.qgroundcontrol.com/v4.4.3/en/qgc-user-guide/fly_view/fly_view.html)
- A map product needs layers, scale, and inspectable spatial results, not a
  decorative background. This informs future MapLibre work. [OpenDroneMap](https://opendronemap.org/download/)

## Experience principles

1. **Situation before decoration.** The map, selected object, freshness, and
   active incident outrank KPI tiles, shadows, and decorative motion.
2. **Context, not navigation churn.** Selecting a device or alarm keeps the
   operator in place; deep links remain available for a durable investigation.
3. **Evidence is visible.** Every important state includes its source/time and
   distinguishes connection, device presence, telemetry freshness, health, and
   command progress.
4. **Incidents are interruptive but not disruptive.** A critical event updates
   the counter, queue, map, and accessible announcement once. It never steals
   a typing focus or permanently flashes.
5. **Actions match authority.** No visual affordance may imply live DJI,
   flight-control, DRC, or hardware authority that the product does not have.
6. **The same meaning has a non-map route.** Device, alert, and selected-map
   states remain reachable through keyboard-operable lists and details.

## Success criteria for the refresh

- In five seconds, a desktop operator can find active critical incidents,
  selected-device freshness, and the active real-time connection state.
- The overview has no more than one primary action in the context pane and no
  unbounded dashboard card strip.
- A critical, warning, stale, or offline state is understandable in a
  grayscale screenshot.
- The Pilot Shell answers current task, device evidence, connection quality,
  and next safe step without presenting a reduced desktop dashboard.
