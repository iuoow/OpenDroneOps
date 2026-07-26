# Development Documentation

This directory contains public engineering records that help contributors understand
the project without exposing local Codex workspace prompts.

- Dependency and licensing decisions: `dependencies.md` and `license-decision.md`
- Per-task validation: `validation/`
- Frontend work breakdown: `uiux-tasks.md`
- Current task order and scope: `../architecture/14-roadmap.md`

The local Codex workspace is intentionally outside the public repository. It may
contain prompts, temporary manifests, and operator notes; those files are not
authoritative project contracts. Public contracts live in `docs/`, `api/`, and
`schemas/`, while architectural decisions live in `../decisions/`.
