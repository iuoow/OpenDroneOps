# Task 15 Validation

## Scope

Task 15 adds the Pilot Shell Foundation execution boundary:

- a separate Vite multi-page entry at `pilot.html`;
- a browser-only `BrowserMockPilotBridge` that implements the Task 14 adapter
  without accessing a Pilot global, browser storage, logs, or real credentials;
- an injected `bootstrapPilot` lifecycle that configures the workspace, API, and
  WebSocket only after availability, license, and runtime-config checks pass;
- a minimal bootstrap mount that reports normalized startup status and exposes
  retry only for retryable failures.

The compact touch-first Pilot UI remains Task 16 work.

## Validation checklist

| Check | Result |
|---|---|
| `npm test` | PASS: 6 files, 19 tests |
| `npm run typecheck` | PASS |
| `npm run build` | PASS: emits Desktop `index.html` and Pilot `pilot.html` |
| Pilot build output | PASS: Pilot JavaScript 1.66 KB gzip before Task 16 UI work |
| Mock Bridge ready lifecycle | PASS: license, workspace, API, and WebSocket calls occur in order |
| Mock Bridge unavailable lifecycle | PASS: retryable `BRIDGE_UNAVAILABLE`; no configuration calls |
| Mock Bridge license rejection | PASS: non-retryable `LICENSE_REJECTED`; no configuration calls |
| Mock Bridge configuration rejection | PASS: retryable `CONFIGURATION_REJECTED` without raw error details |
| `rg "window.djiBridge" web/src` | PASS: no direct access exists |
| `git diff --check` | PASS |

## Boundary notes

- `web/src/pilot/main.ts` is the composition root. It injects the Mock Bridge
  and non-sensitive endpoint configuration into the Pilot bootstrap component.
- `PilotBootstrapApp.vue` receives an adapter through props and does not know
  about any browser or Pilot 2 global.
- The Mock Bridge does not claim an actual DJI license or host capability.
  Replacing it with a real adapter remains blocked by Task 20's external gate.
- No command, alarm acknowledgement, device mutation, DRC, filesystem access,
  or diagnostic upload was introduced.
