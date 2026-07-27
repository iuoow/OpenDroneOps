# ADR 0013: Pilot Shell Foundation Before Real DJI Integration

- Status: Accepted
- Date: 2026-07-27

## Context

The MVP Operations Console, simulator, and live-infrastructure validation are
complete. The roadmap calls for a DJI Pilot 2-oriented experience, but the
project does not yet have approved DJI credentials, a supported real device
model, field-test authorization, or a production safety SOP.

Treating a browser implementation as a real DJI integration would create an
unsafe and misleading release boundary. Waiting for all external approvals,
however, would delay validation of the separate touch-first Pilot experience
specified by ADR 0008.

## Decision

The next product phase is **Pilot Shell Foundation**. It delivers a separate
Pilot entry point testable in a browser with a Mock Bridge and the existing
read-only Operations API and WebSocket contracts.

The phase may implement a `PilotBridgeAdapter`, browser Mock Bridge, capability
detection, startup UX, a separate Vite entry point, touch-first compact views,
read-only live state, visible reconnection, non-sensitive local drafts, and a
consent-first mock diagnostic state machine.

The phase must not implement real Pilot 2 credentials, direct `window` bridge
calls from business components, DRC, flight controls, device-control commands,
real diagnostic-log access/upload, or claims of real-device support.

## Consequences

- Pilot UX and shared web contracts can be validated before external DJI
  approval, while the Desktop Operations Console remains independent.
- The adapter is the only seam a later approved Pilot 2 integration may use;
  bridge configuration and secrets stay outside business components and logs.
- Alarm acknowledgement, third-party app launch, and all device mutations stay
  out of this release because real identity and field authority are absent.
- Moving to real Pilot 2 requires Task 20's documented gate: supported model,
  DJI credentials, authorized hardware testing, security/privacy review, and a
  production SOP.

## Alternatives

1. Build Pilot screens inside the Desktop shell: rejected by ADR 0008 because
   the touch-first field workflow is distinct from desktop operations.
2. Start with real DJI bridge calls: rejected because approval, device, and
   safety prerequisites are absent.
3. Defer all Pilot work: rejected because Mock-Bridge validation can safely
   reduce later integration risk.
