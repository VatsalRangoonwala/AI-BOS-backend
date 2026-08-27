# Testing and quality plan

## Test pyramid

### Unit tests

Cover money/quantity arithmetic, state machines, permission policies, plan entitlements, invoice numbering, payment allocation, stock reservation/deduction, idempotency, prompt/tool argument validation, and notification selection.

### Integration tests

Run against real PostgreSQL and Redis containers. Cover migrations, repository queries, transaction rollback, row locking, outbox claiming, session rotation, signed URL metadata, and worker retries.

### Contract tests

Validate handlers against `api/openapi.yaml` and generated frontend types. Every endpoint has success, validation, unauthorized, cross-tenant, conflict, and dependency-failure examples.

### End-to-end tests

Exercise browser-facing journeys: register/verify, create business, invite staff, create customer/product, confirm order, issue invoice, generate/download PDF, record partial payment, view dashboard, ask assistant, confirm a write, and subscribe in Razorpay sandbox.

## Security tests

- Automated tenant-isolation matrix using IDs from another business.
- RBAC tests for each role and resource/state transition.
- CSRF, CORS, session replay, refresh-token reuse, and rate-limit tests.
- SQL injection, XSS in customer/product/invoice fields, malicious file, and oversized payload tests.
- Razorpay signature, duplicate, out-of-order, and replay tests.
- Prompt-injection, data-exfiltration, unauthorized-tool, and confirmation-bypass tests.
- Dependency/container scanning, secret scanning, and `govulncheck`/static analysis in CI.

## Performance tests

Use a production-like dataset and k6 (or equivalent) to test:

- 1,000 concurrent users for login, list/search, invoice creation, and dashboard reads.
- p95 normal API latency below 300 ms and p99 below 1 second for the agreed profile.
- Queue throughput for PDF/notification bursts without starving API database connections.
- AI streaming latency, provider timeout behavior, and quota enforcement.

Measure database pool saturation, slow queries, cache hit rate, Redis latency, queue age, provider latency, and error budgets—not just HTTP response time.

## Quality gates

A pull request must have formatted/linted code, unit and integration tests, migration checks, OpenAPI validation, security scans, and review. A release additionally requires staging smoke tests, rollback verification, and a database-backup restore check on the scheduled cadence.

## Data fixtures

Keep deterministic fixtures for multiple businesses, roles, currencies, fractional quantities, partial payments, overdue invoices, low stock, failed notifications, and subscription states. Never use production PII in tests.

