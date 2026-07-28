# Task 27 Validation

## Scope

Task 27 adds a strict release-candidate package verifier, deterministic
positive/negative verifier checks, a consolidated GitHub candidate handover
artifact, and operator documentation. It does not publish images, create a
GitHub Release, expose a registry credential, sign an artifact, or authorize a
production deployment.

## Automated verification

Run from the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/test-release-package-verifier.ps1
go test ./...
go vet ./...
go build ./cmd/...
npm.cmd --prefix web run typecheck
npm.cmd --prefix web test
npm.cmd --prefix web run build
git diff --check
```

The verifier test creates a temporary three-target candidate package, verifies
it successfully, then changes one manifest SHA-256 and verifies that rejection
is enforced. CI executes the same verifier test in its delivery-contract job.
The manually triggered Release Candidate workflow verifies its real archives,
creates a combined manifest, verifies that manifest, and uploads the combined
handover package.

## Result

Task 27 is complete when the local commands and CI pass. Docker image builds
may continue to be validated by GitHub Actions while Docker Desktop lacks local
disk capacity.
