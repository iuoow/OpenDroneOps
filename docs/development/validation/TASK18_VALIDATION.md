# Task 18 Validation

## Scope

Task 18 adds field continuity to the Pilot Foundation. A field operator can
explicitly reconnect and refresh the read-only snapshot after a network
interruption, while retaining a short workspace-scoped field note locally.
Drafts remain browser-local and are never submitted automatically.

The draft store accepts only a fixed schema (`workspaceId`, optional device
ID, note body, and timestamps), limits note length, rejects credential-like
values and diagnostic/log paths, keeps at most 20 drafts, and supports explicit
retry-for-edit and discard actions. It has no submit, command, acknowledgement,
bridge, or filesystem API.

## Automated verification

Executed from `web/`:

- `npm.cmd test -- --run` — 9 test files, 35 tests passed.
- `npm.cmd run typecheck` — passed.
- `npm.cmd run build` — passed; Desktop and Pilot entry points built.
- `git diff --check` — passed.

Executed from the repository root:

- `go test ./...` — passed.

## Covered behaviors

- Explicit reconnect stops the existing realtime connection, reloads the
  workspace snapshot, and starts a fresh scoped realtime session.
- Drafts persist across store instances only under their Workspace key.
- Draft text is normalized and capped at 500 characters.
- Credential-like content (`token`, `secret`, authorization, and similar) and
  diagnostic/log paths are rejected with a stable user-facing message.
- Drafts can be loaded back into the editor for retry/editing.
- Drafts require an explicit delete action and are not silently submitted.
- The Pilot UI has no draft submission control and continues to expose only
  read-only operational data.

## Result

Task 18 is complete for the browser Mock Bridge/foundation scope. Diagnostic
upload, device mutation, and real Pilot 2 integration remain deferred by ADR
0013 and Task 20's external gate.
