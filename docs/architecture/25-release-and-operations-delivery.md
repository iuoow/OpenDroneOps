# Release and Operations Delivery

Task 25 turns the repository into a release-candidate delivery source without
automatically publishing software to a registry or creating a GitHub Release.

## Deliverables

| Item | Purpose |
|---|---|
| `Dockerfile` | Reproducible, non-root images for `server`, `worker`, and `migrate` targets. |
| `internal/buildinfo` | Version, commit, and UTC build-time provenance embedded with linker flags and logged at startup. |
| `deployment/docker-compose.release.yaml` | Application-only deployment overlay for external PostgreSQL, Redis, and MQTT. |
| `deployment/release.env.example` | No-secret production configuration inventory. Copy it to `release.env` outside source control. |
| `scripts/package-release.ps1` | Local candidate image archives, SHA-256 manifest verification, and no registry push. |
| `.github/workflows/release-candidate.yml` | Manually triggered image-archive workflow; verifies and uploads a checksummed handover bundle only. |

The normal CI `delivery-contract` job builds all three images and validates the
release Compose interpolation. It cannot publish images.

## Candidate procedure

1. Ensure `main` CI is green and the worktree is clean.
2. Choose a SemVer candidate such as `v0.1.0-rc.1`.
3. Run the manual **Release Candidate** workflow, or locally run:

   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts/package-release.ps1 -Version v0.1.0-rc.1
   ```

4. Verify the whole handover directory with
   `scripts/verify-release-package.ps1`. This checks the candidate version,
   source commit, required targets, archive names, and every archive SHA-256.
5. Scan/sign and push the exact archives under immutable registry digests using
   the organisation's approved registry process. Record those digests; a tag is
   not deployment authority.
6. Copy `deployment/release.env.example` to a secret-managed
   `deployment/release.env`; assign a unique `REALTIME_NODE_ID` per instance.
7. On the target host, validate and apply migrations before server/worker
   rollout:

   ```powershell
   docker compose --env-file deployment/release.env -f deployment/docker-compose.release.yaml config --quiet
   docker compose --env-file deployment/release.env -f deployment/docker-compose.release.yaml run --rm migrate
   docker compose --env-file deployment/release.env -f deployment/docker-compose.release.yaml up -d server worker
   ```

`migrate` is deliberately a one-shot service. The standard `up` flow depends
on it completing successfully, but an operator should run it explicitly first
when a migration needs a recorded approval or a rollback decision.

## Rollback

1. Stop or drain workers before reverting a server/worker image digest.
2. Re-deploy the previously recorded immutable server and worker digests.
3. Do **not** run an automatic database down migration. Use the release's
   recorded migration decision, compatible forward fix, or verified restore.
4. Verify `/api/v1/health/live`, authorised readiness checks, MQTT reconnect,
   queue/capacity outcomes, command Outbox recovery, and WebSocket reconnect.
5. Record the UTC interval, digest, configuration revision, evidence links,
   outcome, and follow-up owner.

## Boundaries

This task supplies candidate artifacts and operating instructions. Registry
credentials, signing keys, production secrets, image publication, release tags,
database backup approval, and external change management remain explicitly
human-authorized actions.
