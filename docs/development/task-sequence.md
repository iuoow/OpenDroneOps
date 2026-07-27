# Implementation Task Sequence

This is the public source of truth for implementation order. Work on one task at a
time, complete its validation record, then commit before starting the next task.

| Task | Scope | Status |
|---|---|---|
| 0 | Baseline dependencies, licensing, contracts, and MVP boundaries | Complete |
| 1 | Go module, service entrypoints, configuration, health checks, CI scaffold | Complete |
| 2 | Domain models, command state rules, PostgreSQL migration, and invariants | Complete |
| 3 | DJI Topic and Envelope parsing, unknown-field compatibility, parser tests | Complete |
| 4 | Deterministic Gateway/Aircraft protocol simulator and fault injection | Complete |
| 5 | MQTT ingestion worker, bounded sharding, deduplication, and graceful shutdown | Complete |
| 6 | Device digital twin, PostgreSQL latest state, derived Redis cache, and history | Complete |
| 7 | WebSocket hub, authorization, bounded writers, snapshot/cursor recovery | Complete |
| 8 | Alarm detection, deduplication, acknowledgement, persistence, and recovery | Complete |
| 9 | Command + transactional outbox, MQTT publishing, replies, timeout, and audit | Complete |
| 10 | Vue operations console and REST/WebSocket client integration | Complete |
| 11 | Trajectory replay and query limits | Complete |
| 12 | Capacity, resilience, observability, security, and release verification | Complete |
| 13 | Live PostgreSQL, Redis and MQTT integration/E2E verification | Complete (validated by GitHub Actions while local Docker is unavailable) |
| 14 | Pilot Shell foundation contracts, safety boundaries, and bridge test plan | Planned |
| 15 | Separate Pilot entry point and browser Mock Bridge | Planned |
| 16 | Touch-first, high-contrast Pilot shell and bootstrap UX | Planned |
| 17 | Read-only Pilot Home with snapshot/WebSocket recovery | Planned |
| 18 | Field continuity: reconnect UX and non-sensitive local drafts | Planned |
| 19 | Consent-first diagnostic UX and Pilot quality gate | Planned |
| 20 | Real Pilot 2 integration and authorized hardware-validation gate | Blocked: requires DJI credentials, supported model, authorized hardware, and security SOP |

Validation reports live in `docs/development/validation/` and use a matching
`TASKx_VALIDATION.md` name. Later tasks must not import private Codex prompts or
rely on removed workspace files.

Tasks 14–19 form the Pilot Shell Foundation and are constrained by
`docs/decisions/0013-pilot-shell-foundation-before-real-dji.md`. They do not
enable real DJI, device mutation, flight control, or DRC capabilities. Task 20
may begin only after its external gate is explicitly satisfied.
