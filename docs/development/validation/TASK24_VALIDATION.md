# Task 24 Validation

## Scope

Task 24 introduces `cmd/capacitycheck` and `scripts/capacity-check.ps1`. The
harness emits a machine-readable report for local fan-out, slow-client fault,
MQTT hot-key recovery, and multi-instance realtime duplicate suppression. It
does not require Docker or external credentials.

## Automated verification

Run from the repository root:

```powershell
go test ./internal/capacitycheck
powershell -ExecutionPolicy Bypass -File scripts/capacity-check.ps1
go test ./...
git diff --check
```

The harness has a deterministic low-volume unit test. CI additionally runs the
default command, including its non-zero failure semantics, before building Go
commands.

## Result

Task 24 is complete when the commands above pass. The resulting report is a
capacity regression signal only. A real deployment still needs the documented
infrastructure load/fault evidence before adopting any production limit or SLO.
