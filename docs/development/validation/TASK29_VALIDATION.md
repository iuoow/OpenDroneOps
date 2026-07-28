# Task 29 Validation

## Scope

Task 29 implements the approved visual foundation and Overview v2 benchmark.
It applies the accepted v2 token values to the active token source, replaces
text-abbreviation navigation with reusable SVG icons, and adds the desktop
attention rail, map workspace treatment, selected-device preview, and
attention-aware map markers. It does not add MapLibre data sources, video,
real DJI capability, DRC, or flight-control actions.

## Automated verification

Run from the repository root:

```powershell
npm.cmd --prefix web run typecheck
npm.cmd --prefix web test
npm.cmd --prefix web run build
go test ./...
go vet ./...
go build ./cmd/...
git diff --check
```

`OperationsMap.test.ts` verifies that selected and attention states are both
rendered, retain an accessible marker label, and emit the selected device.

## Visual verification

The browser preview at `127.0.0.1:4173` was already serving a different/stale
local worktree, so it could not be used as evidence for this change without
stopping a user-owned process. The current source was therefore verified by
typecheck, component tests, production build, and CI. A maintainer can run
`npm.cmd --prefix web run dev` from this repository and inspect
`/app/demo/overview` for the intended visual check.

## Result

Task 29 is complete when the automated commands and CI pass. Task 30 should
apply the same visual system to Alarms and the durable incident workflow.
