# Contributor validation checklist

Run the applicable checks before opening a pull request. All commands are
repository-local; never use real credentials, production URLs, real device
identifiers, or unredacted recordings as test input.

## Required for every change

```powershell
git diff --check
go vet ./...
go test ./...
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

For changes that affect concurrent Go code, also run:

```powershell
go test -race ./...
```

For PostgreSQL, Redis, MQTT, migration, or worker behavior, run the Docker
integration path when local disk space permits. Otherwise rely on the required
GitHub Actions integration job and report that local Docker validation was not
available.

```powershell
pwsh -NoProfile -File scripts/integration-e2e.ps1
```

## Pull request evidence

- State the problem, approach, alternatives considered, and known limits.
- List each command run and its result; link the successful GitHub Actions run.
- For UI changes, state the route and demo/API mode used, plus keyboard, axe,
  E2E, and bundle-budget results where relevant.
- For contract changes, update the matching `docs/`, `api/`, or `schemas/`
  source and explain backward-compatibility handling.
- For a new dependency, record its exact version, license, purpose, and
  alternative in `docs/development/dependencies.md` before use.
- Add or update a task validation record under `docs/development/validation/`.

## Safety and open-source hygiene

- Do not commit keys, tokens, certificates, private recordings, host paths,
  real serial numbers, production endpoints, or real operator information.
- Do not claim Browser Mock or the simulator is a real DJI/Pilot 2 integration.
- Do not add flight control, automatic takeoff, DRC, diagnostic-log upload, or
  device mutation without the documented external authorization gate.
- Report security vulnerabilities privately as described in `SECURITY.md`; do
  not post exploitable details in public issues.
- Keep commits focused, use pinned versions, and leave generated output,
  browser diagnostics, and local `.env` files untracked.
