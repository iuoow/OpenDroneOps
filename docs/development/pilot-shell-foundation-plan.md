# Pilot Shell Foundation Plan

## Product outcome

Provide a compact, touch-first field companion for DJI Pilot 2 deployment
readiness. In this phase it runs in a normal browser through a Mock Bridge and
shows current operational information; it is not a flight-control product and
does not claim a real DJI connection.

The target field workflow is deliberately narrow:

```text
Open Pilot Shell
  -> inspect bridge/cloud state
  -> see current assigned device and urgent alarms
  -> inspect compact details or retain a non-sensitive local draft
  -> recover visibly after a network interruption
```

## Phase assumptions

- Existing REST snapshot and WebSocket contracts remain the source of read-only
  device and alarm data.
- Desktop and Pilot have independent entry points/navigation but share domain
  contracts, API client primitives, design tokens, and basic components.
- A browser Mock Bridge is the development and automated-test environment.
- No DJI app key, license, broker credential, authorization token, diagnostic
  file path, or device secret is committed, rendered, or logged.
- The phase ends before real Pilot 2 or real hardware testing.

## Implementation sequence

| Task | Deliverable | Acceptance evidence |
|---|---|---|
| 14 | Pilot foundation contracts: adapter types, startup state model, capability/error vocabulary, bridge-boundary test plan | Type-level contract tests; ADR 0013 constraints referenced by the design |
| 15 | Separate Pilot build entry (`pilot.html` and Pilot bootstrap), browser Mock Bridge, dependency-injection boundary | Browser build passes; no business component directly accesses `window.djiBridge`; unavailable/rejected/ready bridge tests pass |
| 16 | Touch-first Pilot shell: bottom navigation, high-contrast outdoor theme, 44x44 px targets, 14 px text, bootstrap/retry screens | Visual plus keyboard/touch checks; Pilot bundle budget recorded separately |
| 17 | Read-only Pilot Home and compact alarm/device details backed by snapshot + WebSocket, including stale/disconnected/recovery states | Unit/component and browser-Mock E2E verify snapshot, incremental update, and recovery |
| 18 | Field continuity: explicit reconnect action and non-sensitive local drafts with retry/discard controls | Tests prove drafts are not silently submitted, are removable, and have no secrets/diagnostic paths |
| 19 | Consent-first diagnostic UX using a mockable state machine; no filesystem read/upload implementation | Consent, cancellation, failure, and redaction UX tests; no bridge log-path data is rendered or logged |
| 20 | Real Pilot 2 integration and authorized hardware-validation gate | Blocked until all external prerequisites below are approved |

Every completed task receives a validation record in
`docs/development/validation/TASKx_VALIDATION.md` and one focused commit before
the next task begins.

## Interface and release boundaries

`PilotBridgeAdapter` is the only boundary to Pilot 2. The browser Mock Bridge
exposes equivalent test behavior but must not pretend it has a DJI license or
access to host logs. Runtime configuration is injected at bootstrap; components
receive capabilities and application state, not raw bridge globals or secrets.

Foundation capabilities are read-only. Alarm acknowledgement, third-party app
launch, command submission, and all flight-affecting actions are deferred. The
current MVP lacks real identity and field-authority controls needed to make
those actions defensible.

## Task 20 external gate

All of the following must be explicitly available before real Pilot 2 work is
scheduled:

1. First supported DJI model and current official compatibility confirmation.
2. Approved DJI credentials and license process, stored outside the repository.
3. Authorized Pilot 2/Dock hardware test environment and named field owner.
4. Security and privacy review for bridge configuration and diagnostics.
5. Production SOP covering operator authority, rollback, audit, and emergency
   disablement.
6. Separate approval before any command or DRC-related capability is enabled.

## Quality gate

- Typecheck, unit tests, and production builds for Desktop and Pilot entries.
- Browser-Mock E2E for startup, no-bridge, rejected-license, snapshot, realtime
  update, disconnect, and recovery paths.
- An automated guard rejects direct business-component use of `window.djiBridge`.
- Keyboard and touch checks, WCAG 2.2 AA review, 44x44 px target checks, and
  high-contrast visual review.
- Pilot JavaScript gzip budget at or below 200 KB, as specified by the UI
  performance document.
- No release document may call the Foundation real DJI, DRC, or flight control.

## Deferred after the Foundation

Real Pilot 2 bridge integration, authenticated identity, diagnostic upload,
work-order synchronization, device-control methods, real Dock testing,
production deployment, scale-out, mission/media, and DRC remain separate
roadmap work with their own contracts, safety review, and acceptance criteria.
