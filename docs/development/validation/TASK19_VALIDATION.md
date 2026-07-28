# Task 19 Validation

## Scope

Task 19 adds a consent-first diagnostic UX state machine to the Pilot
Foundation. It is intentionally a mockable preparation flow, not a diagnostic
file integration:

- the user must explicitly open the consent explanation before preparation;
- preparation can be cancelled while waiting or running;
- failure exposes a stable retry state without raw bridge errors;
- a successful result contains only a redacted receipt ID, item count, and
  redacted-field count;
- no filesystem path, raw log, credential, upload endpoint, or bridge log API is
  read, rendered, persisted, or uploaded.

The browser Mock runner returns an empty redacted summary so the UX can be
verified without host access.

## Automated verification

Executed from `web/`:

- `npm.cmd test -- --run` — 10 test files, 41 tests passed.
- `npm.cmd run typecheck` — passed.
- `npm.cmd run build` — passed; Desktop and Pilot entry points built.
- `git diff --check` — passed.

The repository's Go test suite was already green for Task 18 and is unchanged
by this frontend-only task; the remote CI run is the final full-repository
verification.

## Covered behaviors

- Preparation cannot start before explicit consent.
- Consent can be cancelled before preparation begins.
- In-flight preparation can be cancelled; a late runner result is ignored.
- Invalid receipt IDs, negative/oversized counts, and unsafe metadata are
  replaced or filtered at the boundary.
- Runner failures become a stable `PREPARATION_FAILED` state and can be
  retried.
- The More view exposes consent, cancel, retry, and clear-result controls but
  no upload or file-selection control.
- The mock runner contains no filesystem, bridge-global, credential, or network
  upload behavior.

## Result

Task 19 is complete for the browser Mock Bridge/foundation scope. Real
diagnostic collection/upload and real Pilot 2 integration remain deferred by
ADR 0013 and Task 20's external gate.
