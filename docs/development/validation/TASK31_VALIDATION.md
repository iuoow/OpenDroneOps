# Task 31 Validation

## Scope

Task 31 refreshes the desktop Command Center. It adds a status-filtered command
queue, evidence-first command details, an explicit state progression, and a
clear low-risk status-refresh entry point. The implementation preserves the
existing command state machine and does not change MQTT, outbox, or device
execution semantics.

## Interaction guarantees

- The queue separates in-flight and terminal commands without treating a
  transport acknowledgement as a device result.
- Each detail view names the current confirmed fact, shows device telemetry
  freshness, and keeps technical correlation identifiers available on demand.
- `PUBLISHED`, `ACCEPTED`, `EXECUTING`, and `TIMEOUT` explicitly state that the
  device result is not yet confirmed. No automatic resend is presented.
- The only creation action is the existing `sim_status_refresh` command. It is
  labelled as low risk and is disabled when its target device is offline.
- No flight controls, DRC, high-risk command methods, or batch actions are
  introduced.

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

`CommandsView.test.ts` verifies that the success state remains a distinct
device-confirmed outcome and that the UI can create only a `LOW`,
`sim_status_refresh`, `PUBLISH_PENDING` command in demo mode.

## Result

Task 31 is complete when the commands and CI pass. The next visual-system task
should bring the device inventory and device-context workflow to the same
evidence-first hierarchy.
