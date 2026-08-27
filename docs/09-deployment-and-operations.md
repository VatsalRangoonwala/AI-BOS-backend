# Deployment and operations

## Production topology

- Cloudflare DNS/TLS/WAF in front of the API.
- Two or more stateless Go API replicas behind a load balancer.
- One or more Go worker replicas with queue leases and graceful shutdown.
- Managed PostgreSQL with automated backups and point-in-time recovery.
- Managed Redis with persistence/replication appropriate to its cache/queue role.
- Private Cloudflare R2 bucket for generated files.
- Separate staging and production provider credentials and webhook URLs.

The initial VPS deployment can use Docker Compose with a reverse proxy, but database and Redis should be managed services as soon as the product handles real financial records.

## Configuration and secrets

Validate configuration at startup and fail fast on missing required values. Keep secrets in the deployment secret manager, not Git or images. Separate credentials by environment and provider. Rotate JWT signing keys, database passwords, Redis credentials, R2 keys, Razorpay secrets, messaging credentials, and OpenAI keys with a documented procedure.

Typical configuration groups:

```text
APP_ENV, HTTP_ADDR, PUBLIC_API_URL, FRONTEND_ORIGINS
DATABASE_URL, DB_POOL_MAX, DB_POOL_MIN
REDIS_URL
JWT_ISSUER, JWT_KEY_ID, JWT_SIGNING_KEY
R2_ENDPOINT, R2_BUCKET, R2_ACCESS_KEY, R2_SECRET_KEY
OPENAI_API_KEY, OPENAI_MODEL, AI_TIMEOUT, AI_MONTHLY_CAP
RAZORPAY_KEY_ID, RAZORPAY_KEY_SECRET, RAZORPAY_WEBHOOK_SECRET
RESEND_API_KEY, EMAIL_FROM
WHATSAPP_* provider credentials
```

## CI/CD

1. Pull request: format, lint, unit tests, integration containers, migration tests, OpenAPI validation, vulnerability/secret scans, and build.
2. Merge to main: build immutable API/worker images, publish signed artifacts, and deploy to staging.
3. Staging: smoke and contract tests, migration dry run, provider sandbox checks.
4. Production: approval gate, expand/contract migrations, rolling deploy, health checks, smoke test, and monitored rollout.
5. Rollback: redeploy previous image; never roll back a migration that has destroyed data.

## Observability

Every request and job carries a request/trace ID. Emit structured logs for request outcome, actor/business IDs (non-PII), status code, latency, dependency outcome, and error code.

Track dashboards and alerts for:

- API rate, error rate, p50/p95/p99 latency, saturation, and panics.
- Database connections, locks, slow queries, replication/backup status, and storage.
- Redis memory, evictions, latency, and queue age/depth.
- Outbox/job success, retries, dead letters, and oldest pending age.
- PDF, email, WhatsApp, Razorpay, and OpenAI latency/error rates.
- Login failures, suspicious refresh-token reuse, rate-limit blocks, and authorization failures.

Do not put invoice contents, full phone numbers, access tokens, or prompt/customer PII in ordinary logs.

## Reliability targets

- Availability objective: 99.9% for core API endpoints.
- Normal API target: p95 under 300 ms, p99 under 1 second under the agreed load.
- AI target: aim for under five seconds to first useful response, provider-dependent.
- Recovery time objective: within one hour as required by `AI-BOS.md`.
- Set and document an explicit recovery point objective; use point-in-time recovery rather than relying only on daily dumps.

## Backups and disaster recovery

Use automated daily backups plus continuous/WAL point-in-time recovery where available. Encrypt backups and restrict access. Test restoring a staging database from backup at least monthly, record duration, validate row counts/checksums for ledgers, and rehearse API/worker/provider recovery. Keep weekly full backups according to the product requirement.

## Runbooks

Create short runbooks for: database outage, Redis outage, stuck/dead-letter jobs, provider outage, Razorpay webhook backlog, OpenAI quota/error spike, suspected tenant-isolation bug, credential rotation, backup restore, rollback, and data deletion/export request.

