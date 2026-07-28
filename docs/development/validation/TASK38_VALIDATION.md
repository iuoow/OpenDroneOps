# Task 38 Validation

Task 38 extends the browser quality gate to the separate, touch-first Pilot
Shell while preserving the ADR 0013 Foundation boundary.

- `web/e2e/pilot-shell.spec.ts` starts only the Browser Mock composition at a
  390 px field viewport with deterministic demo data.
- The test traverses Home, Device, Alerts, and More, and verifies every Pilot
  bottom-navigation control has a rendered height of at least 44 px.
- A saved field note is retained as a local draft, and the diagnostic flow can
  only proceed from an explicit consent screen to a prepared, non-uploaded
  summary.
- axe-core reports no `serious` or `critical` violations for the ready Pilot
  Shell.
- Stable navigation and view test identifiers make the Browser Mock contract
  observable without coupling the tests to visual copy or real bridge APIs.

No real DJI/Pilot 2 bridge, device control, DRC, hardware, credential,
filesystem, diagnostic-log, or upload capability is added by this task.

Verified with `npm.cmd --prefix web run typecheck`, `npm.cmd --prefix web test`,
`npm.cmd --prefix web run build`, `npm.cmd --prefix web run test:e2e`,
`go test ./...`, `go vet ./...`, `go build ./cmd/...`, and `git diff --check`.
