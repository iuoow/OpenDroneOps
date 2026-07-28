# Task 36 Validation

Task 36 strengthens real-backend UI resilience without changing the REST or
WebSocket contracts.

- API mode now retains the last successful snapshot during WebSocket recovery
  or disconnection and gives the operator an explicit snapshot reload path.
- A cursor-bearing WebSocket subscription remains in recovery until the first
  valid incremental envelope arrives; it then reports recovered realtime data.
- Malformed realtime payloads are ignored rather than falsely reporting that a
  healthy socket is disconnected.
- Demo and API modes remain explicit: recovery notices apply only to API mode.

Verified with `npm.cmd --prefix web run typecheck`, `npm.cmd --prefix web test`,
`npm.cmd --prefix web run build`, `go test ./...`, `go vet ./...`,
`go build ./cmd/...`, and `git diff --check`.
