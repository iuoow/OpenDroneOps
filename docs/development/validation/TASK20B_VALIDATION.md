# Task 20B Validation

## Scope

Task 20B wires the hardware-independent readiness decision into the Pilot
runtime composition root and visible More view.

The composition root evaluates the current browser Mock target, asserts that
the Bridge kind matches the permitted readiness mode, and fails closed with a
stable `PILOT_READINESS_BLOCKED` error when they disagree. The UI exposes the
active mode and whether real Pilot 2, Command, and DRC capabilities are enabled.

No real adapter, credential, hardware, filesystem, or network integration is
introduced.

## Automated verification

Executed from `web/`:

- `npm.cmd test -- --run` — 11 test files, 49 tests passed.
- `npm.cmd run typecheck` — passed.
- `npm.cmd run build` — passed.
- Pilot JavaScript bundle — 9.80 KB gzip, below the 200 KB Pilot budget.
- `git diff --check` — passed.

## Covered behaviors

- Browser Mock plus Mock readiness mode starts successfully.
- A Pilot 2 Bridge paired with Mock readiness fails closed.
- The default repository evidence remains explicitly unapproved.
- The More view shows Browser Mock/read-only mode.
- Real Pilot 2, Command, and DRC are visibly disabled.
- Runtime guard errors contain stable blocker codes, not credentials or raw
  external details.

## Result

Task 20B is complete as a hardware-independent preparation subtask. Task 20
remains blocked until its external evidence is approved.
