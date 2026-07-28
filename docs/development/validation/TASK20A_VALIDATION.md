# Task 20A Validation

## Scope

Task 20A adds a hardware-independent Pilot 2 readiness and capability
conformance evaluator. It translates the external gate registry into a safe
runtime decision without attempting real integration.

The evaluator supports three safe outcomes:

- browser Mock Bridge, read-only only;
- approved Pilot 2 read-only integration after the five baseline approvals;
- approved Pilot 2 control mode only after the separate Command/DRC approval.

Missing, draft, submitted, or rejected evidence always produces stable blocker
codes. No credential, serial number, firmware detail, path, or raw external
error is accepted or returned.

## Automated verification

Executed from `web/`:

- `npm.cmd test -- --run` — 11 test files, 47 tests passed.
- `npm.cmd run typecheck` — passed.
- `npm.cmd run build` — passed; Desktop and Pilot entry points built.
- `git diff --check` — passed.

## Covered behaviors

- Mock mode remains available with no external evidence, but command and DRC
  requests are rejected.
- Real read-only mode reports every unapproved baseline gate.
- Real control mode requires the separate Command/DRC approval.
- Blocker codes contain no secrets, paths, serial numbers, or raw errors.
- The evaluator is pure and hardware-independent; it does not alter the
  existing Mock Bridge or connect to a real adapter.

## Result

Task 20A is complete as a preparation subtask. Task 20 remains blocked until
the external gate registry is fully approved.
