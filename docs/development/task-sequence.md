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
| 8 | Alarm detection, deduplication, acknowledgement, persistence, and recovery | Next |
| 9 | Command + transactional outbox, MQTT publishing, replies, timeout, and audit | Planned |
| 10 | Vue operations console and REST/WebSocket client integration | Planned |
| 11 | Trajectory replay and query limits | Planned |
| 12 | Capacity, resilience, observability, security, and release verification | Planned |

Validation reports live in `docs/development/validation/` and use a matching
`TASKx_VALIDATION.md` name. Later tasks must not import private Codex prompts or
rely on removed workspace files.
