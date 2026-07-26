# Repository Layout Validation

- Validation date: 2026-07-26
- Public repository root now contains runtime code, contracts, deployment assets,
  and a small set of maintainer files.
- Architecture records are under `docs/architecture/`.
- ADRs are under `docs/decisions/`.
- Design and UI specifications are under `docs/design/`.
- Dependency, task-order, and validation records are under `docs/development/`.
- Codex prompts, temporary manifests, and operator notes were moved to the local
  workspace at `E:\ai\OpenDroneOps\_codex_workspace`; they are not part of the public repository.
- `.gitignore` blocks accidental reintroduction of local Codex artifacts.

## Reference checks

- No repository reference remains to the removed root paths `CODEX_TASKS.md`,
  `CODEX_START_HERE.md`, `codex/`, `adr/`, `design/`, `docs/ui/`,
  `DEPENDENCIES.md`, `LICENSE_DECISION.md`, or `UIUX_TASKS.md`.
- Go formatting, tests, vet, build, Compose configuration, JSON parsing, and
  migration marker checks pass.
- The remote repository remains unpushed until this layout commit is reviewed.
