# Architecture and repository design

## Proposed topology

```text
Browser / PWA
    |
    | HTTPS REST / SSE
    v
Cloudflare (TLS, WAF, rate-limit edge rules)
    |
    +--> Go API replicas (stateless)
    |       |-- PostgreSQL: transactions and source of truth
    |       |-- Redis: cache, rate limits, queue coordination
    |       |-- R2: private PDFs and uploaded files
    |       '-- OpenAI: assistant orchestration
    |
    '-- Go worker replicas
            |-- outbox/event jobs
            |-- invoice PDF generation
            |-- Resend email
            |-- WhatsApp Business API
            '-- Razorpay reconciliation
```

Deploy one API binary and one worker binary from the same repository. The domain modules are isolated in code and can later become services without first introducing distributed transactions.

## Technology decisions

| Concern | Choice | Reason |
| --- | --- | --- |
| Language | Go | Small operational footprint, strong concurrency, easy static binaries |
| HTTP | `chi` (or equivalent small router) | Explicit middleware and idiomatic Go |
| PostgreSQL | `pgx` + `sqlc` | Typed queries, transaction control, no runtime ORM surprises |
| Migrations | Versioned SQL with `goose`/`atlas`-style tool | Reviewable schema history and repeatable deploys |
| Cache/queue | Redis + durable jobs/outbox in PostgreSQL | Fast coordination without making Redis authoritative |
| API contract | OpenAPI 3.1, generated Go/frontend types | Prevent mock/API drift |
| Storage | Cloudflare R2, private bucket | PDFs/files without bloating PostgreSQL |
| AI | OpenAI Responses API with function tools | Structured tool loop and streamed responses |
| Observability | OpenTelemetry, Prometheus-compatible metrics, structured logs | Correlate request, job, provider, and business events |
| Packaging | Docker; Docker Compose for local development | Reproducible environments |

## Repository layout

```text
.
├── cmd/
│   ├── api/main.go
│   └── worker/main.go
├── internal/
│   ├── platform/          # config, db, redis, http, auth, observability
│   ├── identity/
│   ├── businesses/
│   ├── customers/
│   ├── catalog/
│   ├── inventory/
│   ├── sales/              # orders, invoices, payments, ledger
│   ├── analytics/
│   ├── notifications/
│   ├── subscriptions/
│   ├── assistant/
│   └── files/
├── migrations/
├── api/openapi.yaml
├── test/                    # integration, contract, fixtures
├── deploy/                  # Docker, reverse proxy, IaC/runbooks
├── docs/
└── Makefile
```

Each module should expose application use cases and domain types, keep SQL in repository packages, and depend on interfaces for providers. HTTP handlers translate transport DTOs to use cases; they must not contain business rules.

## Request lifecycle

1. Cloudflare terminates TLS and applies coarse edge controls.
2. API middleware assigns a request ID, enforces body/timeout limits, authenticates the session, resolves the active business, and checks rate limits.
3. Handler validates the request against the OpenAPI schema.
4. Application service checks RBAC and plan entitlements, then runs a transaction if state changes.
5. The transaction writes business rows, audit events, and outbox entries together.
6. The response returns a stable envelope with resource data and metadata.
7. Workers deliver external side effects asynchronously.

## Synchronous versus asynchronous work

Keep synchronous: authentication, CRUD, stock validation, order confirmation, invoice issue, manual payment recording, and read-only dashboards.

Queue asynchronously: PDF rendering, email/WhatsApp, reminders, low-stock notifications, analytics rollups, AI follow-up work, provider reconciliation, and cleanup jobs.

## Cross-cutting conventions

- UTC timestamps in storage and ISO-8601 in JSON.
- UUIDs for public identifiers; never expose sequential database IDs.
- Money as integer minor units plus `currency`; never floating-point JSON numbers.
- Decimal quantities for stock; define scale/rounding per product/business.
- Soft archive where history matters; hard delete only for legally safe ephemeral records.
- Structured errors with a stable `code`, human-safe `message`, optional `field_errors`, and `request_id`.
- All mutations accept an idempotency key where a client retry could duplicate state.
- Forward-compatible enums: clients must tolerate an unknown status.

## Environments

- Local: Docker Compose PostgreSQL/Redis, fake providers, seeded fixtures.
- CI: ephemeral PostgreSQL/Redis, provider contract mocks, race detector, migrations from empty database.
- Staging: production-like managed services and sandbox provider accounts.
- Production: isolated credentials, private storage, managed database backups, at least two API replicas, worker autoscaling when queue depth requires it.

