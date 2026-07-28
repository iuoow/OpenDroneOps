# Operating and Rollback SOP

- Registry item: `operating_sop`
- Status: `draft`
- Evidence owner: `TBD`
- Required approver: `Operations owner`
- External record reference: `ADR-0013; release-checklist`
- Decision date: `TBD`
- Review/expiry date: `TBD`

## Repository-supported draft

The current release process already requires immutable image digests,
migration/rollback decisions, PostgreSQL backup/restore evidence, Redis cache
rebuild evidence, production MQTT TLS and ACLs, bounded-queue fault results,
and monitoring for the loopback-only metrics endpoint. ADR 0013 additionally
keeps real Pilot 2, device control, DRC, and diagnostic upload disabled until
the external gate is approved.

These controls are a repository-side draft only and do not authorize field
operations.

## Required external decision

Record operator authority, preflight checks, audit requirements, rollback,
disablement, emergency escalation, maintenance windows, and release sign-off.

The SOP must explicitly state that commands and DRC are disabled unless their
separate approval is active.
