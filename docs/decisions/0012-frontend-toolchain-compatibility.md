# ADR 0012: Frontend TypeScript Toolchain Compatibility

- Status: Accepted
- Date: 2026-07-26

## Context

The frontend dependency baseline originally listed TypeScript 7.0.2. During
Task 10 validation, the pinned `vue-tsc` 3.2.4 toolchain could not resolve
TypeScript 7's package exports and type checking failed before application
sources were evaluated.

## Decision

Use TypeScript 5.9.3 for the Vue/Vite workspace until the Vue language tooling
publishes a compatible TypeScript 7 integration. The version remains exact in
`web/package.json` and `web/package-lock.json`.

## Consequences

- `vue-tsc --noEmit`, Vitest, and Vite production builds run on the locked
  workspace without compiler shims.
- A future TypeScript 7 upgrade requires rerunning the frontend quality gate and
  updating this ADR and the dependency register.
