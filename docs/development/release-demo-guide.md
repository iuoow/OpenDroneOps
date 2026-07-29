# Release demo and experience acceptance

This guide is the public, repeatable demonstration path for the current
OpenDroneOps release candidate. It intentionally uses deterministic demo data
and Browser Mock; it is not evidence of a real DJI, Pilot 2, Dock, device
control, DRC, or production deployment.

## Run the interactive demo

Prerequisite: Node.js and npm versions supported by `web/package-lock.json`.

```powershell
Set-Location web
npm.cmd ci
npm.cmd run dev -- --host 127.0.0.1 --port 4173
```

Open these local-only routes:

| Experience | Route | Purpose |
|---|---|---|
| Desktop Operations | `http://127.0.0.1:4173/app/demo/overview` | Demonstrate the browser-only Operations Console. |
| Pilot Shell | `http://127.0.0.1:4173/pilot.html` | Demonstrate the isolated, touch-first Browser Mock shell. |

`VITE_DEMO_MODE` defaults to demo mode. Do not set it to `false` unless the
REST and WebSocket backend has been configured separately; the demo must not be
presented as a live fleet connection.

## Suggested demonstration flow

1. In Desktop Operations, begin at **Overview** and select a device to explain
   its current state and evidence time.
2. Open **Alarms** to show severity, acknowledgement state, and the separation
   between triage information and a device action.
3. Open **Commands** to show the persisted command/result evidence. The demo
   contains only bounded, low-risk simulated actions; an MQTT acknowledgement
   is never described as device execution success.
4. Open **Replay** to explain historical evidence, then **Runtime** to show
   the browser realtime state and the boundary around management-plane metrics.
5. Open the separate **Pilot Shell**. Visit Device, Alerts, and More with the
   bottom navigation. In More, confirm that Browser Mock/read-only mode is
   visible and Pilot 2, Command, and DRC are disabled.
6. In Pilot diagnostics, choose **View consent information** and then consent
   only to prepare the mock redacted summary. Confirm that no file path is
   shown and no log is uploaded.

## Screenshots and repeatable UI evidence

Task 39 independently inspected and captured the Desktop Runtime and Pilot
More views using the local demo. The automated equivalent is checked on every
change:

```powershell
Set-Location web
npm.cmd run test:e2e
```

The browser suite covers the desktop investigation path, keyboard skip link,
Pilot 390 px touch navigation, local-only drafts, the consent-first diagnostic
state, and serious/critical axe violations. On a browser-test failure, CI
retains screenshots, video, and first-retry trace as the
`web-e2e-diagnostics` artifact. This is deliberately more reliable than
committing host-specific pixel baselines.

## Local infrastructure verification

The interactive demo does not require Docker. If Docker Desktop has sufficient
space and is running, validate real PostgreSQL, Redis, and MQTT integration
with:

```powershell
pwsh -NoProfile -File scripts/integration-e2e.ps1
```

The script starts pinned local services, applies migrations, runs the tagged
integration tests, and tears services down unless `-KeepServices` is passed.
It is a developer verification path, not a production deployment command.

## Non-negotiable release boundary

- Browser demo data and Browser Mock are synthetic and read-only.
- No real DJI/Pilot 2 credential, app key, license, serial number, device
  control, flight command, DRC path, diagnostic-file access, or log upload is
  included.
- Real Pilot 2 work remains blocked until the external gate documented in
  `docs/development/validation/TASK20_GATE_READINESS.md` is satisfied.
- Platform capacity endpoints and Prometheus metrics are management-plane
  interfaces; the public desktop session does not retrieve or cache them.
