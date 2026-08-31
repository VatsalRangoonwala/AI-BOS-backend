# AI-BOS Backend Plan

This directory is the engineering plan for turning the AI-BOS frontend from mock data into a production backend.

`AI-BOS.md` remains the product requirements source of truth. These documents turn those requirements into an implementable, testable, and operable backend design. When the product changes, update the relevant decision and contract documents before changing code.

## Status

| Area | Status |
| --- | --- |
| Product scope | Drafted from `AI-BOS.md` |
| Architecture | Proposed |
| Database model | Proposed |
| Backend foundation | Implemented |
| REST contract | Health/readiness foundation implemented; domain APIs remain proposed |
| AI assistant | Proposed |
| Operations | Proposed |
| Domain implementation | Not started |

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
13. [Universal backend task workflow](13-agent-workflow.md)

## Local setup

Prerequisites:

- Go 1.27.0 or a newer compatible Go 1.27 patch release.
- Docker with the Compose plugin.
- Node.js 24 for local OpenAPI linting.

From a clean checkout:

```bash
cp .env.example .env
make docker-up
make migrate-up
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz
```

`make docker-up` builds and starts separate API and worker containers plus PostgreSQL and Redis. PostgreSQL is the source of truth. Redis is used only for cache and coordination concerns and is never authoritative.

To run the Go processes on the host while dependencies run in Docker:

```bash
docker compose up -d postgres redis
set -a
. ./.env
set +a
go run ./cmd/api
```

Run `go run ./cmd/worker` in a second terminal with the same exported environment. The worker currently verifies its dependencies and waits for shutdown; no business jobs are implemented yet.

## Environment variables

The committed `.env.example` contains local-only values. The application does not load `.env` files itself; export them in the shell or let Docker Compose supply the container environment. Use deployment-managed secrets outside local development.

| Variable | Purpose | Local default |
| --- | --- | --- |
| `APP_ENV` | Required: `local`, `test`, `staging`, or `production` | None |
| `LOG_LEVEL` | Structured JSON log level | `info` |
| `HTTP_ADDR` | API listen address | `:8080` |
| `FRONTEND_ORIGINS` | Comma-separated exact CORS origins | `http://localhost:3000` |
| `HTTP_READ_HEADER_TIMEOUT` | Header read deadline | `5s` |
| `HTTP_READ_TIMEOUT` | Request read deadline | `15s` |
| `HTTP_WRITE_TIMEOUT` | Response write deadline | `20s` |
| `HTTP_IDLE_TIMEOUT` | Keep-alive idle deadline | `60s` |
| `HTTP_REQUEST_TIMEOUT` | Per-request context deadline | `15s` |
| `HTTP_SHUTDOWN_TIMEOUT` | Graceful HTTP drain limit | `15s` |
| `HTTP_MAX_BODY_BYTES` | Maximum request body size | `1048576` |
| `DATABASE_URL` | pgx PostgreSQL connection URL | Local PostgreSQL URL only |
| `DB_POOL_MAX`, `DB_POOL_MIN` | pgx pool bounds | `20`, `2` |
| `DB_CONNECT_TIMEOUT` | PostgreSQL connection deadline | `5s` |
| `DB_HEALTH_TIMEOUT` | PostgreSQL readiness deadline | `2s` |
| `DB_MAX_CONN_LIFETIME` | Maximum pooled connection lifetime | `30m` |
| `DB_MAX_CONN_IDLE_TIME` | Maximum pooled idle time | `5m` |
| `REDIS_URL` | Redis connection URL | `redis://localhost:6379/0` |
| `REDIS_DIAL_TIMEOUT` | Redis connection deadline | `3s` |
| `REDIS_READ_TIMEOUT` | Redis read deadline | `2s` |
| `REDIS_WRITE_TIMEOUT` | Redis write deadline | `2s` |
| `REDIS_HEALTH_TIMEOUT` | Redis readiness deadline | `2s` |
| `WORKER_HEALTH_INTERVAL` | Worker dependency-check interval | `30s` |

`APP_ENV` is always required. `DATABASE_URL`, `REDIS_URL`, and `FRONTEND_ORIGINS` receive defaults only when it is explicitly `local`; they are required in every other environment. Production also requires PostgreSQL TLS, a `rediss://` Redis URL, and HTTPS frontend origins. Wildcard CORS origins are rejected. Connection URLs may contain credentials and must never be printed or committed.

## Development commands

```bash
make fmt              # format Go code
make lint             # go vet and golangci-lint
make test             # unit tests with the race detector
make test-integration # real PostgreSQL and Redis checks; start dependencies first
make build            # build bin/api and bin/worker
make openapi-lint     # validate api/openapi.yaml
make security         # module verification and govulncheck
make migrate-up       # apply versioned SQL migrations
make docker-down      # stop local containers
```

The initial migration is a schema-neutral foundation marker. It creates no customer, product, inventory, order, invoice, payment, AI, notification, or subscription table.

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
