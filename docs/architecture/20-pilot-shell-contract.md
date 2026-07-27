# Pilot Shell Foundation Contract

## Purpose

This contract defines the safe browser-facing boundary used by Tasks 14–19. It
implements ADR 0013 and is intentionally narrower than a future DJI Pilot 2
integration.

## Adapter boundary

`web/src/pilot/bridge.ts` exports `PilotBridgeAdapter`. It is the only
application-facing bridge seam. Foundation components receive state and
capabilities through that adapter; they must not access `window.djiBridge`.

The adapter has only startup configuration methods:

- detect availability;
- verify a license result without exposing license material;
- set the selected workspace;
- configure API and WebSocket endpoints.

It has no method for device control, command submission, DRC, filesystem access,
diagnostic uploads, credentials, or tokens.

## Runtime configuration

The contract accepts a workspace ID, HTTP(S) API endpoint, WebSocket endpoint,
and a non-empty required-module list. The validation function rejects invalid
configuration with the stable `CONFIGURATION_REJECTED` code. It does not retain
or surface the rejected input.

## Startup state model

```text
detecting
  -> verifying_license
  -> configuring
  -> loading_modules
  -> ready

detecting/configuring/loading_modules
  -> failed (retryable)

verifying_license
  -> failed LICENSE_REJECTED (not retryable)
```

Failures carry only a stable code and retryability. Raw bridge exceptions,
diagnostic paths, credentials, and host-specific details are never part of
startup state or UI copy.

## Deferred boundary

Task 15 supplies only a browser Mock Bridge. A real Pilot 2 adapter can be
considered only under Task 20's external gate and must extend this contract via
a separate ADR and security review.
