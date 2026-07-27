# Task 14 Validation

## Scope

Task 14 establishes the Pilot Shell Foundation contract layer without creating
a Pilot UI entry point or a real DJI integration:

- a safe `PilotBridgeAdapter` interface for availability, license outcome,
  workspace selection, and API/WebSocket endpoint configuration;
- a non-sensitive runtime configuration shape and endpoint validation;
- a pure startup-state reducer covering bridge discovery, license verification,
  configuration, required-module loading, recoverable failures, and terminal
  license rejection;
- stable failure codes that avoid retaining raw bridge exceptions, host paths,
  credentials, or diagnostic data;
- a public architecture contract that binds the implementation to ADR 0013.

## Validation checklist

| Check | Result |
|---|---|
| `npm test` | PASS: 5 files, 15 tests |
| `npm run typecheck` | PASS |
| `npm run build` | PASS |
| `git diff --check` | PASS |
| Adapter type contract assertions | PASS: only safe startup methods are exposed |
| Startup sequence and out-of-order event tests | PASS |
| Failure retry and license-rejection tests | PASS |
| Runtime configuration validation tests | PASS |

## Boundary notes

- Task 14 defines types and a pure reducer only; it does not detect or call a
  browser/Pilot 2 bridge. The browser Mock Bridge belongs to Task 15.
- `PilotBridgeAdapter` intentionally contains no device-control, command, DRC,
  filesystem, diagnostic-upload, token, or credential capability.
- A successful production build still produces only the existing Desktop entry;
  the separate Pilot entry point is Task 15 work.
- Real Pilot 2, real device, and diagnostic integration remain blocked by the
  external gate in ADR 0013 and Task 20.
