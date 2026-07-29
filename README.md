# OpenDroneOps

OpenDroneOps is an open-source operations console for DJI Cloud API-oriented
drone fleets. It combines a Go backend, MQTT ingestion, PostgreSQL/Redis
state, WebSocket recovery, a Vue Operations Console, and a separate
touch-first Pilot Shell foundation.

The repository is deliberately runnable and reviewable without DJI hardware:
the desktop demo and the Pilot Browser Mock use deterministic synthetic data.
They must not be presented as a real DJI, Pilot 2, Dock, flight-control, or
DRC integration.

## What is available

- DJI topic/envelope parsing with compatibility handling and a deterministic
  protocol simulator.
- Bounded MQTT ingestion, digital-twin state, PostgreSQL persistence, Redis
  derived state, WebSocket snapshot/cursor recovery, alarms, command/outbox
  audit, and trajectory replay.
- Desktop Operations Console: overview, device inventory, alarm triage,
  command evidence, replay, runtime boundaries, keyboard support, and state
  recovery evidence.
- Pilot Shell Foundation: a separate Browser Mock entry point with read-only
  field data, touch-first navigation, local-only drafts, and consent-first
  diagnostic-summary UX.
- CI for Go, live PostgreSQL/Redis/MQTT integration, deployable image
  contracts, frontend unit tests, Chromium E2E/axe checks, and initial bundle
  budgets.

## Quick demo

The quickest way to see the product does **not** require Docker or hardware.

```powershell
git clone https://github.com/iuoow/OpenDroneOps.git
Set-Location OpenDroneOps/web
npm.cmd ci
npm.cmd run dev -- --host 127.0.0.1 --port 4173
```

Then open:

- Desktop Operations: `http://127.0.0.1:4173/app/demo/overview`
- Pilot Browser Mock: `http://127.0.0.1:4173/pilot.html`

For a scripted walkthrough, screenshots/evidence policy, integration option,
and explicit release boundaries, see
[Release demo and experience acceptance](docs/development/release-demo-guide.md).

## Architecture at a glance

```text
DJI-compatible simulator or device
  → MQTT Worker / DJI adapter
  → domain events and transactional outbox
  → PostgreSQL source of truth + Redis derived state
  → REST snapshot + WebSocket increments/cursor recovery
  → Operations Console and read-only Pilot Browser Mock
```

Key rules:

- DJI DTOs and topics remain inside the adapter boundary.
- PostgreSQL is the business source of truth; Redis is rebuildable derived
  state.
- MQTT QoS/PUBACK never proves a device action completed.
- All queues and retries are bounded; alarms and commands remain recoverable.
- The public browser session does not expose management-plane capacity or
  Prometheus endpoints.

Detailed architecture and contracts live in [docs/architecture](docs/architecture/)
and [api](api/).

## Verify a checkout

Run the standard checks before contributing:

```powershell
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/...

Set-Location web
npm.cmd ci
npm.cmd run typecheck
npm.cmd test
npm.cmd run build
npm.cmd run test:bundle
npm.cmd run check:bundle
npm.cmd run test:e2e
```

When Docker Desktop has sufficient space, verify real local PostgreSQL, Redis,
and MQTT integration with:

```powershell
pwsh -NoProfile -File scripts/integration-e2e.ps1
```

The complete checklist is in
[Contributor validation](docs/development/contributor-validation.md). CI runs
the same core checks on `main` and pull requests.

## Current scope and boundary

Tasks 0–19 and 21–40 are complete. They cover the MVP, Pilot Shell Foundation,
scale/release readiness, UI refresh, browser quality gates, performance budget
checks, and release documentation. The current sequence is recorded in
[docs/development/task-sequence.md](docs/development/task-sequence.md).

Task 20—real Pilot 2/DJI integration—remains blocked. It requires a supported
model and firmware range, approved DJI credentials/license handling,
authorized hardware and field owner, security/privacy review, and an operating
SOP. The Browser Mock does not bypass that gate. See
[Task 20 gate readiness](docs/development/validation/TASK20_GATE_READINESS.md)
and [ADR 0013](docs/decisions/0013-pilot-shell-foundation-before-real-dji.md).

OpenDroneOps does not currently provide real flight control, automatic takeoff,
DRC, video transport, diagnostic-file access, or diagnostic upload.

## Technology baseline

| Area | Choice |
|---|---|
| Backend | Go, Gin, PostgreSQL, pgx/sqlc, Redis |
| Messaging | Eclipse Paho MQTT v5, Eclipse Mosquitto |
| Realtime | REST snapshots, coder/websocket, cursor recovery |
| Frontend | Vue 3, TypeScript, Vite, Pinia |
| Quality | Go tests/race checks, Vitest, Playwright, axe-core, bundle budgets |
| Observability | OpenTelemetry, structured logs, Prometheus-compatible metrics |

Pinned versions, licenses, and dependency decisions are documented in
[docs/development/dependencies.md](docs/development/dependencies.md).

## Contributing and security

Read [CONTRIBUTING.md](CONTRIBUTING.md) and the
[contributor validation checklist](docs/development/contributor-validation.md)
before opening a pull request. Do not commit credentials, real serial numbers,
private recordings, production endpoints, or unredacted operator data.

Report security issues according to [SECURITY.md](SECURITY.md), not in a public
issue. The project license is [Apache-2.0](LICENSE).
