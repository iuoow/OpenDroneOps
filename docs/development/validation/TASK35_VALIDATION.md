# Task 35 Validation

Task 35 adds a cross-page accessibility and state-language quality gate.

- Every keyboard-operable link, button, input, select, and custom button role
  receives a consistent visible focus indicator.
- The desktop shell has a skip-to-content link and a stable main-content target.
- Realtime recovery and disconnection are announced without claiming device or
  MQTT status.
- Missing or invalid telemetry timestamps are explicitly labelled as unavailable
  rather than being presented as fresh values.

`FreshnessIndicator.test.ts` locks the unavailable/offline distinction. The
existing view tests retain coverage for status labels, low-risk command bounds,
and keyboard-operable list controls.

Verified with `npm.cmd --prefix web run typecheck`, `npm.cmd --prefix web test`,
`npm.cmd --prefix web run build`, `go test ./...`, `go vet ./...`,
`go build ./cmd/...`, and `git diff --check`.
