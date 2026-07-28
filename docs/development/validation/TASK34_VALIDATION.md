# Task 34 Validation

Task 34 replaces the System Operations placeholder with a read-only runtime
evidence view. It shows browser WebSocket recovery facts and public command
workflow state while explicitly retaining the management-plane boundary for
MQTT, Outbox, Prometheus, and `/capacity`.

- Browser connection, snapshot recovery, and command workflow are labelled as
  web-visible evidence.
- Capacity, queue, and Outbox sections explain operational limits without
  fetching the loopback-only management endpoint or inventing live counts.
- No runtime configuration, capacity management, flight, device, or batch
  action is introduced.

Verified with `npm.cmd --prefix web run typecheck`, `npm.cmd --prefix web test`,
`npm.cmd --prefix web run build`, `go test ./...`, `go vet ./...`,
`go build ./cmd/...`, and `git diff --check`.
