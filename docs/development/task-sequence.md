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
| 14 | Pilot Shell foundation contracts, safety boundaries, and bridge test plan | Complete |
| 15 | Separate Pilot entry point and browser Mock Bridge | Complete |
| 16 | Touch-first, high-contrast Pilot shell and bootstrap UX | Complete |
| 17 | Read-only Pilot Home with snapshot/WebSocket recovery | Complete |
| 18 | Field continuity: reconnect UX and non-sensitive local drafts | Complete |
| 19 | Consent-first diagnostic UX and Pilot quality gate | Complete |
| 20 | Real Pilot 2 integration and authorized hardware-validation gate | Blocked: requires DJI credentials, supported model, authorized hardware, and security SOP; see `docs/development/validation/TASK20_GATE_READINESS.md` and `docs/development/gates/task20/` |
| 21 | Capacity and quota contract | Complete; see `docs/architecture/21-capacity-and-quota-contract.md` and `docs/development/validation/TASK21_VALIDATION.md` |
| 22 | Fairness, backpressure, and hot-key isolation | Planned |
| 23 | Multi-instance realtime delivery and recovery | Planned |
| 24 | Load, fault, and capacity verification harness | Planned |
| 25 | Release automation and operational delivery | Planned |
| 26 | Capacity visibility and operator experience | Planned |

Task 20A is a completed preparation subtask: it adds a hardware-independent
readiness/capability evaluator and conformance tests. It does not count Task 20
as complete and does not authorize real hardware, credentials, commands, or DRC.
Task 20B is also complete: it wires that evaluator into the browser Mock
composition root and exposes the fail-closed capability state in the Pilot UI.

Validation reports live in `docs/development/validation/` and use a matching
`TASKx_VALIDATION.md` name. Later tasks must not import private Codex prompts or
rely on removed workspace files.

Tasks 14–19 form the Pilot Shell Foundation and are constrained by
`docs/decisions/0013-pilot-shell-foundation-before-real-dji.md`. They do not
enable real DJI, device mutation, flight control, or DRC capabilities. Task 20
may begin only after its external gate is explicitly satisfied.

Tasks 21-26 are Scale & Release Readiness work. They are independent of the
blocked real Pilot 2 gate and must not imply that Task 20's external evidence
has been supplied.
