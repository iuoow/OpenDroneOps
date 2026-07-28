# UI browser quality gate

Task 37 adds a Chromium acceptance layer above component and unit tests. It
protects the operator's primary investigation routes without depending on a
real device, a broker, or private credentials.

## Run locally

From `web/`, run:

```powershell
npm.cmd run test:e2e
```

Playwright starts an isolated Vite server on `127.0.0.1:4174`. Do not point this
test at a production console or an authenticated environment. The suite uses
the public demo route deliberately, so it remains deterministic and safe.

## Covered contract

- An operator can navigate from Overview to Devices, Alarms, Commands, Replay,
  and Runtime in the desktop shell.
- The first keyboard focus exposes the skip-to-content link.
- axe-core must report no `serious` or `critical` violations for the desktop
  demo overview. `landmark-one-main` is excluded because the embedded Pilot
  entry point is independently mounted and is outside this desktop-shell
  assertion.

## Visual diagnostics

The browser configuration retains a screenshot and video when a test fails,
and records a trace on its first retry. CI uploads `web/test-results/` as the
`web-e2e-diagnostics` artifact after a failed web job. This makes a visual
regression actionable without committing host-specific pixel baselines: font
rasterisation and the OS/browser image can otherwise create noisy, non-product
diffs.

When a stable CI rendering image and an approved visual baseline policy are
available, a later task may add Playwright screenshot snapshots. It must retain
this functional, keyboard, and accessibility coverage rather than replacing
it with image-only checks.
