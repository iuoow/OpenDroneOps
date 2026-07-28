# Task 30 Validation

## Scope

Task 30 implements the alert triage and incident-handling refresh. The new
desktop view provides status filters, severity-first queue cards, an
evidence-first incident detail, explicit handling steps, and the existing
acknowledgement action. It does not add manual resolution, real flight control,
DRC, device mutation, or hardware authority.

## Interaction guarantees

- `OPEN` is labelled **待接手** and exposes `确认接手此告警`.
- Acknowledgement records ownership and changes the state to **已接手**.
- The final handling step remains **等待规则恢复**; it explicitly says that
  acknowledgement does not complete recovery.
- Severity uses text, a status badge, icon, queue border, and timestamp rather
  than colour alone.
- Device telemetry freshness remains visible as incident evidence.

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

`AlarmsView.test.ts` verifies the acknowledgement transition and confirms that
the view does not represent it as automatic rule recovery.

## Result

Task 30 is complete when the commands and CI pass. The next visual-system task
should refresh the Command Center's progress and evidence hierarchy while
retaining its existing low-risk command boundary.
