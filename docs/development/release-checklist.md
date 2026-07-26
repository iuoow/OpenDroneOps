# Release Checklist

Run `powershell -ExecutionPolicy Bypass -File scripts/release-check.ps1` from
the repository root before tagging a release. It validates Go, the Operations
Console, Compose interpolation and whitespace without starting infrastructure.

Before publishing, the release owner must also record:

- immutable image digests for PostgreSQL, Redis, Mosquitto and application
  images; tags alone are insufficient for a production deployment;
- the applied Goose migration version and a rollback decision for each new
  migration;
- backup/restore evidence for PostgreSQL and a Redis cache-rebuild exercise;
- production MQTT TLS, broker ACL, unique credentials and secret delivery;
- a load/fault result for bounded MQTT queues, WebSocket slow clients, Outbox
  retry recovery and one server restart;
- the monitoring location for the loopback-only `/metrics` management endpoint.

The public API process must not expose `/metrics`. `APP_ENV=production` rejects
plaintext `mqtt://` URLs and non-loopback `ADMIN_ADDR` values before startup.
