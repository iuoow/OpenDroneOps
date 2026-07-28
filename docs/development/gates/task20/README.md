# Task 20 Gate Pack

This directory is the maintainable, no-secret record for the real Pilot 2
integration gate. It does not grant authorization and it does not replace the
external approval system.

## How to maintain it

1. Keep `gate-registry.json` as the machine-readable source of truth.
2. Update only statuses, opaque external record references, owners, and review
   dates after an approval is recorded outside this repository.
3. Never put credentials, app keys, license files, device serial numbers,
   private URLs, log paths, or signed approval documents in Git.
4. Record the external evidence ID in `externalRecordRef` and summarize the
   decision in the matching evidence file.
5. Run `pwsh -NoProfile -File scripts/validate-pilot2-gate.ps1` before
   committing gate changes.
6. Keep `overallStatus` as `blocked` until every item is `approved` and a
   named approver has reviewed the complete package.

Valid item statuses:

- `missing`: required evidence has not been supplied.
- `draft`: repository-side preparation exists, but external approval is absent.
- `submitted`: evidence is awaiting external review.
- `approved`: evidence is approved and has an opaque external record reference.
- `rejected`: evidence was rejected; Task 20 remains blocked.

The current registry intentionally contains only repository-known facts:
security/privacy preparation is `draft`, and all hardware, credentials, model,
SOP, and Command/DRC approvals remain `missing`.

## Evidence files

Each file in `evidence/` is a fillable record with an owner, decision,
external-record reference, expiry/review date, and explicit no-secret reminder.
They are templates until the corresponding external record is supplied.

The hardware-independent conformance evaluator lives in
`web/src/pilot/readiness.ts`. Its tests are a preparation aid only; passing
them never changes `overallStatus` and never authorizes real hardware.

Task 20B wires the evaluator into the browser Mock composition root and makes
the active capability boundary visible in the Pilot More view. A Bridge/readiness
mode mismatch fails closed before the application is mounted.
