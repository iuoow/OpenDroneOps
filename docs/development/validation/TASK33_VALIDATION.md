# Task 33 Validation

## Scope

Task 33 refreshes the desktop trajectory replay view as a read-only
investigation workspace. It keeps the existing bounded query, history-mode,
worker simplification, shared playback time, and return-to-realtime route.

## Interaction guarantees

- The page explicitly identifies its data as historical evidence and exposes
  the actual loaded point window, rather than treating history as live state.
- Start, end, current playback position, and associated event locations are
  visible on the trajectory canvas. Event markers and the equivalent event list
  are keyboard-operable and move the same playback cursor.
- Selected telemetry, event evidence, trajectory point count, and the data
  boundary are shown together. Missing points do not produce inferred state.
- Playback controls include play/pause, speed, timeline scrubbing, and previous
  or next associated event. The view remains read-only and adds no task,
  command, flight, or device control.

## Automated verification

Run from the repository root:

```powershell
npm.cmd --prefix web run typecheck
npm.cmd --prefix web test
npm.cmd --prefix web run build
go test ./...
go vet ./...
go build ./cmd/...
git diff --check
```

`ReplayView.test.ts` verifies that historical evidence is labelled, its boundary
is visible, and selecting an associated event moves the shared playback cursor.

## Result

Task 33 is complete when the commands and CI pass. The next visual-system task
should refresh the operations view so connection, queue, outbox, and capacity
evidence use the same hierarchy.
