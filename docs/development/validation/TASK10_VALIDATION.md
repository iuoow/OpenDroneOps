# Task 10 Validation

## Scope

Task 10 adds the Vue Operations Console under `web/`:

- Vue 3, TypeScript, Vite, Vue Router and Pinia shell;
- token-first light/dark-ready styling based on `docs/design/tokens/tokens.css`;
- map-first Overview with an accessible device-list equivalent;
- Devices, Alarms and Commands views;
- explicit loading, empty, stale, disconnected and partial-error states;
- REST snapshot client with workspace and request/idempotency headers;
- WebSocket subscription client with cursor, reconnect backoff and recovery status;
- version-aware device reconciliation and event application;
- demo data mode for local work without a live backend;
- Vitest coverage for snapshots, version merging, command headers, recovery
  cursor and reconnect behavior.

The Replay and Operations routes are intentionally shell placeholders; their
query and operational dashboards belong to later tasks.

## Validation checklist

| Check | Result |
|---|---|
| `npm ci` | PASS |
| `npm run typecheck` | PASS |
| `npm test` | PASS: 3 files, 7 tests |
| `npm run build` | PASS |
| `npm ls --depth=0` | PASS: exact top-level versions; npm emitted non-blocking engine warnings for transitive Babel packages |
| Root Go tests/vet/build | PASS before frontend change |
| Token import and responsive layout inspection | PASS |
| HTTP smoke server | NOT RUN: local process launcher rejected the inherited Windows PATH environment |
| In-app browser visual inspection | NOT RUN: browser runtime could not initialize because the required local kernel path was unavailable |
| Live REST/WebSocket integration | NOT RUN: backend runtime was not started |

## Contract notes

- `VITE_DEMO_MODE` defaults to demo data unless explicitly set to `false`;
  production composition must set `VITE_API_BASE_URL` and `VITE_WS_URL`.
- Snapshot data is loaded before incremental events. Device updates with an
  older or equal `state_version` are ignored.
- WebSocket disconnects retain the last snapshot, show a layered connection
  status, and reconnect with bounded exponential backoff and cursor replay.
- Map state is never the only access path: the device list, alarm queue and
  command list provide keyboard-operable equivalents.
- Command UI clearly distinguishes `PUBLISHED` from device execution success
  and does not expose real flight-control methods.
