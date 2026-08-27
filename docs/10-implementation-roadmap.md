# 16-week implementation roadmap

The schedule follows the timeline in `AI-BOS.md`, but each phase ends with a deployable vertical slice and evidence. Dates are estimates; dependencies and acceptance criteria matter more than calendar week numbers.

## Weeks 1-2: contract and foundation

Deliver:

- Mock-data inventory for every frontend screen.
- OpenAPI skeleton, common envelopes, error codes, pagination, auth/session decision.
- ERD, state machines, RBAC/entitlement matrix, threat model, and unresolved decision log.
- Go API/worker skeleton, config validation, Docker Compose, CI, migrations, health endpoints, structured logging, tracing, and local seed fixtures.

Exit criteria: frontend and backend agree on the first auth/customer/product schemas; a clean environment boots API, worker, PostgreSQL, and Redis.

## Weeks 3-4: identity and tenancy

Deliver:

- Registration, verification, login/logout, refresh rotation, reset, `/me`.
- Businesses, memberships, invitations, roles, active-business selection.
- Audit logs, rate limits, CORS/CSRF, session revocation, and tenant-isolation tests.

Exit criteria: two seeded businesses remain isolated under CRUD, search, direct-ID, and unauthorized-role tests.

## Weeks 5-6: customers, catalog, inventory

Deliver:

- Customer CRUD/search/ledger foundation.
- Product CRUD/SKU search/archiving.
- Inventory balances, append-only movements, adjustments, low-stock events, and concurrency tests.

Exit criteria: the frontend customer/inventory screens run against real data; a concurrent stock deduction cannot oversell under the selected policy.

## Weeks 7-9: orders, invoices, payments

Deliver:

- Order state machine and confirmation/cancellation.
- Draft/issue/void invoice flow with numbering, snapshots, totals, and PDF job.
- R2 signed URLs.
- Manual payments, allocations, customer ledger, overdue calculation, and share/reminder command.

Exit criteria: an end-to-end sale is atomic and retry-safe; issued invoice history is reproducible; partial payment updates balances correctly.

## Weeks 10-11: dashboards and notifications

Deliver:

- Summary, trend, inventory, customer, and recent-order queries.
- SQL indexes/rollups and Redis cache where measured useful.
- Outbox worker, Resend integration, WhatsApp template integration, preferences/consent, retries, dead letters, and admin visibility.

Exit criteria: dashboards meet latency target on a production-like dataset and failed provider jobs are recoverable without duplicate sends.

## Weeks 12-13: AI assistant

Deliver:

- Conversation/message persistence and SSE endpoint.
- Read tool catalog, strict argument validation, tenant/RBAC checks.
- Write previews, expiring confirmation actions, domain-service execution, quotas, cost/latency telemetry, fallback behavior, and evaluation dataset.

Exit criteria: representative assistant tasks select correct tools; no test can bypass authorization or confirmation; provider failure leaves a resumable conversation.

## Week 14: plans and Razorpay

Deliver:

- Plan/entitlement configuration and usage counters.
- Checkout/subscription lifecycle, verified idempotent webhooks, reconciliation, upgrade/cancel behavior, and billing UI contract.

Exit criteria: sandbox checkout and every webhook state can be replayed safely; limits are enforced by backend services.

## Week 15: verification and hardening

Deliver:

- Full unit/integration/contract/E2E suite.
- Security testing and scans.
- k6 load test at 1,000 concurrent users, query tuning, connection-pool tuning, and queue burst test.
- Backup restore drill, observability dashboards, alerts, runbooks, and release checklist.

Exit criteria: release gates pass, critical/high findings are closed or explicitly accepted, and rollback/restore are timed and documented.

## Week 16: production release

Deliver:

- Production infrastructure and secrets.
- Expand/contract migrations, rolling API/worker deployment, smoke tests, Cloudflare rules, provider production credentials, and monitored launch.

Exit criteria: core journeys work with real providers, on-call ownership is assigned, and the first 24-hour review has no unresolved critical alert.

## Definition of done for every feature

- OpenAPI schema and frontend example updated.
- Domain service and authorization policy implemented.
- Migration, indexes, audit event, and idempotency behavior defined.
- Unit, integration, contract, and relevant E2E tests added.
- Metrics/logs/traces and failure behavior documented.
- Rollback/data-repair consideration reviewed.

