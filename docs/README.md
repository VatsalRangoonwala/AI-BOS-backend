# AI-BOS Backend Plan

This directory is the engineering plan for turning the AI-BOS frontend from mock data into a production backend.

`AI-BOS.md` remains the product requirements source of truth. These documents turn those requirements into an implementable, testable, and operable backend design. When the product changes, update the relevant decision and contract documents before changing code.

## Status

| Area | Status |
| --- | --- |
| Product scope | Drafted from `AI-BOS.md` |
| Architecture | Proposed |
| Database model | Proposed |
| REST contract | Skeleton; must be reconciled with frontend types |
| AI assistant | Proposed |
| Operations | Proposed |
| Implementation | Not started |

## Reading order

1. [Product scope and acceptance criteria](01-product-and-scope.md)
2. [Architecture and repository design](02-architecture.md)
3. [Domain model and database design](03-domain-and-data-model.md)
4. [REST API contract](04-api-contract.md)
5. [Security and multi-tenancy](05-security-and-multitenancy.md)
6. [AI assistant design](06-ai-assistant.md)
7. [Events and external integrations](07-events-and-integrations.md)
8. [Testing and quality](08-testing-and-quality.md)
9. [Deployment and operations](09-deployment-and-operations.md)
10. [16-week implementation roadmap](10-implementation-roadmap.md)
11. [Frontend mock-data migration](11-frontend-integration.md)
12. [Open product and technical decisions](12-open-decisions.md)

## Engineering principles

- PostgreSQL is the source of truth; Redis is a cache, coordination mechanism, and job queue, never the system of record.
- Every tenant-owned row is scoped by `business_id` and every request is authorized against the active membership.
- Financial and stock changes are append-only ledgers with explicit state transitions.
- External side effects run through an outbox and worker with retries and idempotency.
- The AI can propose actions, but application services validate and authorize them. Destructive or financial writes require confirmation.
- The API contract is versioned and generated from OpenAPI; frontend mock types must converge on the same schemas.
- Start as a modular monolith plus worker. Split services only when an observed scaling or ownership boundary justifies it.

## Definition of production ready

The backend is ready for a public MVP when all of the following are true:

- Core user journeys work against real PostgreSQL data with no mock fallbacks.
- Tenant isolation, RBAC, authentication, webhook verification, and audit logging have automated tests.
- Invoice, stock, payment, and subscription operations are idempotent and recoverable.
- Background jobs have retries, dead-letter handling, and visible operational status.
- API p95 is below 300 ms for normal CRUD/dashboard requests under the agreed load profile; AI latency and provider failures are observable.
- Backups have been restored in a drill, deployment rollback is documented, and alerts have an owner.

