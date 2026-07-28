# Release Artifact Integrity and Handover

Task 27 makes a release-candidate package independently verifiable before an
operator scans, signs, transfers, or publishes it. It adds no registry write,
release creation, secret, signing key, or production deployment authority.

## Package contract

A candidate handover directory contains these files:

| Item | Rule |
|---|---|
| `release-manifest.json` | One manifest for the candidate version and full source commit. |
| `opendroneops-server-<version>.tar[.gz]` | Server image archive. |
| `opendroneops-worker-<version>.tar[.gz]` | Worker image archive. |
| `opendroneops-migrate-<version>.tar[.gz]` | Migration image archive. |
| `*.sha256` | Candidate workflow's archive checksum evidence. |

The manifest must contain exactly the `server`, `worker`, and `migrate`
targets. Each target has an exact image reference, a safe basename-only archive
name, and a lower-case SHA-256 that matches the file. The version must be a
SemVer candidate beginning with `v`; the commit must be a full lower-case Git
SHA. The verifier rejects missing, duplicate, extra, renamed, or modified
artifacts.

## Operator workflow

For a locally created candidate:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/package-release.ps1 -Version v0.1.0-rc.1
powershell -ExecutionPolicy Bypass -File scripts/verify-release-package.ps1 -PackageDirectory dist/release
```

`package-release.ps1` invokes the verifier itself after writing the manifest.
Run the second command again after copying a package to another machine or
before giving it to the approved scan/sign/publish process.

For a GitHub candidate, the manual **Release Candidate** workflow first builds
and checksum-verifies each target archive. Its `manifest` job then downloads
all three, creates `release-manifest.json`, validates it with the same script,
and uploads one `opendroneops-release-bundle-<version>` artifact. The artifact
is a handover package, not a published release.

## Boundaries

Checksums establish file integrity for the package received by an operator; they
do not prove that the source is trusted, replace an SBOM or vulnerability scan,
or authorize a registry push. Those controls remain part of the organisation's
approved release process described in Task 25.
