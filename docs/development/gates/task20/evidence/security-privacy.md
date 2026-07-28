# Security and Privacy Review

- Registry item: `security_privacy`
- Status: `draft`
- Evidence owner: `TBD`
- Required approver: `Security and privacy reviewer`
- External record reference: `ADR-0013; TASK19_VALIDATION`
- Decision date: `TBD`
- Review/expiry date: `TBD`

## Repository preparation

The foundation currently enforces an adapter boundary, no direct
`window.djiBridge` access, no filesystem diagnostic read, no upload path, and
consent-first redaction UX. These are preparation facts, not formal approval.

## Required external review

Review bridge origin/CSP, token handling, local storage, log redaction, data
retention, diagnostic consent, auditability, and emergency disablement.
