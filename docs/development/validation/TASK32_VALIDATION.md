# Task 32 Validation

## Scope

Task 32 refreshes the desktop Device Management experience. The previous table
is replaced by a filterable, keyboard-operable device queue and an
evidence-first device context pane. The selected device remains addressable in
the URL and the detail intentionally links to existing investigation routes.

## Interaction guarantees

- Operators can distinguish online device presence, stale telemetry, and
  offline or not-yet-online devices without relying on colour alone.
- The device detail names the current confirmed fact, shows the latest
  telemetry evidence and timestamp, and surfaces related active alarms and
  recent commands.
- Search includes serial number, model, and device type; queue filters include
  all, online, stale, and offline evidence states plus device type.
- The view offers investigation links only: realtime context and, for aircraft,
  trajectory replay. It adds no flight, Dock, batch, or device-control action.
- Empty inventory and filter-empty states are distinct and explain the next
  safe step.

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

`DevicesView.test.ts` verifies stale filtering, URL-addressable selection, the
stale-evidence explanation, and the absence of device-control affordances.

## Result

Task 32 is complete when the commands and CI pass. The next visual-system task
should refresh trajectory replay for investigation-oriented evidence comparison.
