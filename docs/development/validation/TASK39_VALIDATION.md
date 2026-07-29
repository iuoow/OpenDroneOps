# Task 39 Validation

Task 39 makes the existing initial JavaScript gzip budgets executable rather
than relying on a manual reading of build output, and delivers the public
release-demo and contributor-validation handoff.

- Vite emits `dist/.vite/manifest.json` for the multi-entry production build.
- `npm run check:bundle` follows only each entry's static manifest imports and
  sums the gzip bytes of the JavaScript chunks it needs at first load.
- CI runs both the manifest-graph unit test (`npm run test:bundle`) and the
  real built-artifact check after `npm run build`.
- A dynamic import is not counted as an initial-load dependency. It must be
  evaluated with the feature that asks for it, rather than making the initial
  budget ambiguous.

Current measured baseline:

| Entry | Initial static JavaScript gzip | Budget |
|---|---:|---:|
| Operations | 57.59 KiB | 250 KiB |
| Pilot Browser Mock | 36.04 KiB | 200 KiB |

Public handoff artifacts:

- `docs/development/release-demo-guide.md` documents deterministic local demo
  routes, the operator walkthrough, Docker integration verification, captured
  UI-evidence policy, and the hard real-DJI boundary.
- `docs/development/contributor-validation.md` provides the required local and
  CI validation commands, PR evidence expectations, and open-source hygiene
  rules.
- Task 39 independently inspected the Desktop Runtime and Pilot More views in
  the local in-app browser; both reflect the documented read-only boundaries.

Task 39 adds no runtime device, DJI/Pilot 2, control, DRC, credential, or
hardware capability.

Verified with `npm.cmd --prefix web run typecheck`, `npm.cmd --prefix web test`,
`npm.cmd --prefix web run build`, `npm.cmd --prefix web run test:bundle`,
`npm.cmd --prefix web run check:bundle`, `npm.cmd --prefix web run test:e2e`,
`go test ./...`, `go vet ./...`, `go build ./cmd/...`, and `git diff --check`.
