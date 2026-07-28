# Task 26 Validation

## Scope

Task 26 adds an operator-oriented capacity summary to the loopback management
listener and a PowerShell status view. The summary presents sorted controlled
outcomes, severity, and recommended action while Prometheus remains the source
for rate-based alerting.

## Automated verification

Run from the repository root:

```powershell
go test ./internal/observability ./internal/httpapi
go test ./...
git diff --check
```

The registry tests verify ordered guidance and critical health. HTTP tests
verify `/capacity` returns JSON with `Cache-Control: no-store` on the admin
handler and returns 404 on the public API handler.

## Result

Task 26 is complete when the commands above pass. Runtime use requires a
running process with `ADMIN_ADDR` reachable only through approved management
access; the script does not open or publish that endpoint.
