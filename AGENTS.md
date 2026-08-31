# AI-BOS backend engineering workflow

This file is the default operating contract for every coding agent working in this repository. It applies to feature requests, API work, bug fixes, refactors, tests, security fixes, performance work, documentation, and deployment/operations tasks.

Read this file first, then read `AI-BOS.md` and the relevant documents under `docs/` before changing anything. The detailed lifecycle is in `docs/13-agent-workflow.md`.

## Mission

Deliver the user's requested backend change completely and safely. Do not stop at a proposal when implementation is possible. Inspect the current repository and frontend contract, make the smallest coherent change, verify it, review the result, fix findings, and report exactly what is complete and what is externally blocked.

## Universal task lifecycle

Apply these phases to every task. Scale the depth to risk; a typo does not need a load test, but a payment or tenant-isolation change does.

1. **Discover**: inspect the repository, git status, relevant frontend/mock data, existing APIs, migrations, tests, configuration, and user changes. Read the relevant `docs/` files. Do not assume a greenfield project.
2. **Classify**: label the request as feature/API, bug fix, refactor, security, performance, test/quality, documentation, or operations. Identify affected users, tenants, state, external providers, and data.
3. **Plan**: state objective, non-goals, assumptions, files, contract changes, migrations, risks, and verification commands. Use a tracked plan for work with more than one meaningful step.
4. **Contract first**: for public behavior, reconcile the frontend and `AI-BOS.md`; update or create OpenAPI schemas, request/response examples, errors, auth/RBAC, pagination, idempotency, and compatibility notes before implementation.
5. **Implement**: keep HTTP handlers thin and rules in application/domain services. Use scoped repositories, typed queries, transactions, explicit state transitions, and provider interfaces. Keep the change bounded.
6. **Verify**: run the narrowest useful checks while iterating, then all checks required by the risk class. Add regression tests for bugs. Test authorization, tenant isolation, retries/idempotency, validation, conflicts, rollback, and dependency failure when relevant.
7. **Review**: inspect the actual diff for correctness, security, data integrity, migration safety, API compatibility, observability, and documentation drift. Fix blocking/high-severity findings and rerun checks.
8. **Handoff**: summarize outcome, changed files, API/schema/migration/events, exact commands/results, limitations, and next action. Never claim a check passed unless it ran.

## Risk-based verification

| Task | Required minimum verification |
| --- | --- |
| Documentation/config-only | Markdown/config validation, link check where available, diff review |
| Read-only API/query | Unit tests, handler/contract tests, auth/RBAC, cross-tenant test, query/error behavior |
| Write API/domain change | Above plus integration transaction tests, idempotency, rollback, state/concurrency tests, audit/outbox checks |
| Auth/tenant/security | Above plus negative authorization matrix, session/CSRF/rate-limit/secret checks, regression tests |
| Payments/invoices/inventory/subscriptions | Full write checks plus duplicate/replay, financial/stock invariants, webhook/provider failure, migration review |
| AI/tool/provider integration | Contract tests, timeout/retry/circuit behavior, secret/PII review, tenant/RBAC/tool-confirmation tests, cost/latency telemetry |
| Performance/operations | Baseline measurements, load or failure test, resource/alert review, rollback/backup impact |

## Non-negotiable safety rules

- Scope every tenant-owned read and write by `business_id`; verify active membership, role, and plan entitlement on every path.
- PostgreSQL is the source of truth. Redis may cache, rate-limit, or coordinate jobs; it must not silently become authoritative.
- Store money as integer minor units plus currency. Store quantities with explicit decimal semantics. Calculate totals and statuses server-side.
- Use transactions, row/version locking, append-only stock/payment/audit history, and idempotency for retryable mutations.
- Use an outbox and worker for PDFs, email, WhatsApp, analytics, and external provider side effects.
- Verify provider webhook signatures over the raw body, deduplicate external event IDs, and handle retries/out-of-order delivery.
- Do not let AI tools bypass normal services, tenant checks, RBAC, entitlements, or confirmation requirements for consequential actions.
- Never log secrets, tokens, passwords, OTPs, full payment credentials, or unnecessary customer/invoice/prompt content.
- Do not rewrite or delete migrations, revert user changes, weaken security, remove tests, or perform destructive actions without explicit authority.
- Do not expose an unscoped tenant `GetByID` repository method or trust client-supplied totals, roles, statuses, prices, or stock balances.

## Decision boundaries

Make reasonable reversible technical decisions. Stop and ask the user only when the answer materially changes product behavior or needs authority that cannot be discovered locally: money, stock timing, tax, permissions, billing, retention/deletion, external messaging consent, destructive migrations, production secrets/access, or conflicting frontend/product requirements.

When blocked, complete all safe local work first and record the exact decision, evidence, options, and unblock request in `docs/12-open-decisions.md` or the task handoff. Do not hide a blocker behind a guessed implementation.

## Change and collaboration rules

- Preserve unrelated user changes and inspect overlapping edits before touching a file.
- Prefer small vertical slices and existing repository patterns over broad rewrites or premature microservices.
- Public API, database, event, environment, and behavior changes require documentation updates in the same change.
- If specialist agents are available, delegate only bounded non-overlapping work and keep one primary owner for integration and final review. Otherwise perform the same design, implementation, test, and review passes yourself.
- Never claim “production ready” merely because unit tests pass. Include migration, security, observability, deployment, backup/restore, and external-provider status appropriate to the task.

## Handoff format

Use this exact structure in the final response:

```text
Status: complete | blocked | needs-review
Outcome: what now works or what was diagnosed
Changed: important files/modules
Contract: endpoints, schemas, migrations, events, config
Verification: exact commands and results
Production readiness: passed gates; external checks still required
Risks: remaining limitations
Next: smallest sensible follow-up
```

