# Task 20 Gate Readiness

## Current decision

Task 20 remains **Blocked**. The repository is ready for a later controlled
implementation, but the external authorization gate is not satisfied. This
record is a preflight checklist, not evidence that real DJI/Pilot 2 support is
available. The maintainable machine-readable registry and evidence templates
live under `docs/development/gates/task20/`; validate them with
`pwsh -NoProfile -File scripts/validate-pilot2-gate.ps1`.

No credentials, license files, device serial numbers, private URLs, or
hardware-identifying details belong in this repository. Evidence should be
stored in the approved external system and referenced by an opaque record ID.

## Required evidence before implementation

| Gate | Required evidence | Owner / external record |
|---|---|---|
| Supported model | Current official DJI compatibility confirmation for the first target model and firmware range | TBD |
| Credentials and license | Approved app key/license process, least-privilege scope, secret-manager location, rotation and revocation procedure | TBD |
| Authorized lab | Named field owner, approved Pilot 2/Dock hardware, isolated test network, maintenance window, and emergency stop/rollback plan | TBD |
| Security and privacy | Review of bridge origin/CSP, token handling, local storage, log redaction, data retention, and diagnostic consent | TBD |
| Operating SOP | Operator authority, preflight checklist, audit requirements, rollback, disablement, and incident escalation | TBD |
| Command/DRC approval | Separate written approval before any command, DRC, takeoff, landing, return-to-home, or other flight-affecting capability | TBD |

The gate is not satisfied by a developer-owned device, an unreviewed
credential, a browser mock, or a successful simulator run.

## Repository preflight audit

Audited after Task 19:

- `window.djiBridge` direct access: none in `web/src`.
- Real credentials, app keys, or license material: none.
- Filesystem reads, host log path access, and diagnostic upload: none.
- DRC topics and flight-control methods: rejected or deferred.
- Pilot bridge contract: read-only startup/configuration seam only.
- Diagnostic UX: mockable consent/cancellation/redaction state machine only.

The existing Mock Bridge and diagnostic runner must remain browser-safe and must
not be upgraded to simulate real authorization by changing labels or adding
hidden host calls.

## Controlled implementation order after approval

1. Record all six external evidence items and obtain a named approver.
2. Freeze the approved model, firmware, bridge API, and threat model in a new
   ADR; do not place secrets in the repository.
3. Add a lab-only adapter behind `PilotBridgeAdapter` with explicit capability
   flags and an emergency disable path.
4. Validate read-only identity, workspace scoping, telemetry, alarms, and
   recovery on authorized hardware.
5. Run security/privacy review and release verification again.
6. Treat command/DRC as a separate, separately approved product change.

Until step 1 is complete, the correct action is to keep Task 20 blocked and
continue using the simulator and browser Mock Bridge.

Task 20A provides a hardware-independent readiness evaluator and conformance
tests so a later approved adapter can fail closed on the same gate decisions.
Task 20A does not reduce or bypass any of the six external requirements.
