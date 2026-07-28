# Task 37 Validation

Task 37 establishes a repeatable Chromium browser quality gate for the
desktop operations console.

- `web/e2e/operations-console.spec.ts` verifies the main operator navigation
  path in deterministic demo mode.
- Vitest is explicitly scoped to `src/**/*.test.ts`, so Playwright scenarios
  remain an independent browser gate rather than being imported into jsdom.
- The browser suite verifies keyboard skip navigation and blocks `serious` and
  `critical` axe-core accessibility violations on the desktop overview.
- Failure-only screenshots, video, and retry traces are retained locally and
  uploaded by CI as `web-e2e-diagnostics`.
- The muted semantic text token was darkened from `#607088` to `#5b6c84` after
  browser verification identified two AA contrast failures in the top bar.

Run the browser gate with `npm.cmd --prefix web run test:e2e`. It uses an
isolated local Vite server and public demo data; it makes no calls to real DJI
hardware, broker, or operator environments.

Verified with `npm.cmd --prefix web run typecheck`, `npm.cmd --prefix web test`,
`npm.cmd --prefix web run build`, `npm.cmd --prefix web run test:e2e`,
`go test ./...`, `go vet ./...`, `go build ./cmd/...`, and `git diff --check`.
