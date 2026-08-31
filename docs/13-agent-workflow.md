# Universal backend task workflow

This is the detailed workflow behind the repository-level `AGENTS.md`. It is designed for one primary coding agent to handle any backend task end to end, with optional specialist-agent delegation when available. You should be able to give the agent a request such as “build this API,” “fix this bug,” “secure this endpoint,” or “optimize this query”; the same lifecycle applies automatically.

## What the user provides

Give the agent a short objective and any known context:

```text
Build/fix/improve <requested outcome>.
Relevant frontend route or mock: <path or unknown>.
Expected behavior: <examples if known>.
```

Do not need to specify internal roles, implementation steps, or test commands. The agent discovers those from the repository and this workflow.

## What the primary agent does

The primary agent owns the task from intake to handoff. It may use separate passes or delegate bounded work to contract, domain, implementation, test/security, review, and operations agents, but it must reconcile their outputs and remain responsible for the integrated result.

```text
User request
    |
    v
Discover + classify + plan
    |
    v
Contract/domain/data design
    |
    v
Implement bounded slice
    |
    v
Test + security + operational verification
    |
    v
Review actual diff -> fix findings -> repeat
    |
    v
Handoff with evidence and remaining blockers
```

## Phase 1: Discover

Before editing:

- Read `AGENTS.md`, `AI-BOS.md`, and relevant architecture, API, data, security, testing, integration, operations, and frontend docs.
- Inspect `git status`, the repository tree, build files, configuration, migrations, OpenAPI, tests, and existing implementation.
- Locate the frontend repository or relevant routes, TypeScript models, mock data, API helpers, forms, filters, loading/empty/error states, and permissions. If it is unavailable, state that limitation.
- Identify existing user changes and avoid overwriting them.
- Search for the affected endpoint, entity, status, event, environment variable, and test coverage before making assumptions.

The discovery output should answer: what is the current behavior, what should change, which tenants/users/data are affected, and what could regress?

## Phase 2: Classify and choose depth

Classify one primary task type:

- **Feature/API**: new endpoint, workflow, entity, or provider behavior.
- **Bug fix**: incorrect behavior, error, regression, or data issue.
- **Refactor**: structure/readability/dependency change with behavior preserved.
- **Security/privacy**: auth, tenant isolation, secrets, abuse, PII, webhook, file, or AI safety.
- **Performance**: latency, throughput, query, memory, queue, or cost.
- **Test/quality**: missing coverage, flaky test, contract, tooling, or CI.
- **Documentation/configuration**: docs, examples, environment, or generated artifacts.
- **Operations**: deployment, migration, observability, backup, restore, incident, or rollback.

Add secondary labels when the change crosses boundaries. A payment API is both feature and financial-risk; a query optimization may also be data-correctness work.

## Phase 3: Plan and decision check

Write a concise task plan before code. Include:

- objective and non-goals;
- affected frontend workflow and user roles;
- current behavior and desired behavior;
- affected modules/files;
- API/OpenAPI, database/migration, event, config, and provider changes;
- transaction/state/idempotency/concurrency behavior;
- security, tenant, privacy, and plan-entitlement impact;
- tests, commands, and acceptance criteria;
- assumptions, risks, and external blockers.

Compare the task with `docs/12-open-decisions.md`. Use a safe reversible default for ordinary technical details. Ask instead of guessing when the choice changes financial meaning, inventory timing, tax, access rights, billing, data retention, customer messaging, or irreversible history.

## Phase 4: Contract and domain design

For a public API or behavior change, update the OpenAPI contract before depending on it in code. Define:

- paths, methods, auth, RBAC, and plan entitlements;
- request/response schemas and examples;
- validation, pagination/filtering, stable error codes, and status codes;
- idempotency and optimistic-concurrency behavior;
- SSE/event messages where streaming applies;
- backward compatibility and frontend migration behavior.

For data/state changes, define:

- entities, constraints, indexes, and tenant keys;
- state transitions and forbidden transitions;
- transaction boundaries and row/version locking;
- historical snapshots and append-only ledgers;
- migration rollout, backfill, rollback/data-repair strategy;
- audit and outbox events.

No implementation should quietly invent a public endpoint, persisted status, financial rule, or event contract.

## Phase 5: Implement

Implement the smallest complete vertical slice:

1. migration and typed repository/query;
2. domain/application service with authorization;
3. handler, validation, response/error mapping;
4. audit, idempotency, outbox, and provider interface as applicable;
5. focused tests and fixtures;
6. OpenAPI, generated client/types, docs, and frontend wiring where available.

Keep handlers transport-focused. Keep business rules in services. Keep external calls behind interfaces and asynchronous when they do not need to block the user transaction. Return server-calculated totals/statuses and stable IDs; never leak internal SQL or secrets.

For a bug fix, reproduce the failure first when possible, add a regression test that fails before the fix, make the smallest fix, and verify adjacent behavior. For a refactor, establish behavior tests before changing structure and compare public behavior before/after. For a performance task, capture a baseline and prove improvement without changing correctness.

## Phase 6: Verify

Run checks appropriate to the classification and repository tooling. Prefer repository commands; if the project has not yet created them, run the explicit equivalent and document it.

Minimum mutation checklist:

- happy path;
- malformed and domain-invalid input;
- unauthenticated request;
- unauthorized role/plan;
- another tenant's IDs in path, query, and body;
- duplicate request/idempotency key;
- stale/concurrent update where relevant;
- transaction rollback;
- provider timeout/failure/retry;
- audit/outbox/job behavior.

For API work, validate OpenAPI and generated/client compatibility. For database work, migrate from empty and latest-like schema, inspect indexes/locks, and test rollback/data repair. For auth/security, run negative tests and scan for secret/PII leakage. For provider work, verify signatures, replay/out-of-order events, timeouts, retries, and circuit behavior. For performance, record load, dataset, p50/p95/p99, resource saturation, and comparison baseline.

## Phase 7: Review and repair loop

Review the actual diff and test output as a skeptical maintainer. Prioritize:

1. cross-tenant access or missing authorization;
2. duplicate money, stock, invoice, notification, or subscription effects;
3. broken state transitions or historical rewriting;
4. unsafe/destructive migration;
5. unverified webhook/provider input;
6. secret/PII exposure or unsafe file/AI behavior;
7. API/frontend compatibility regression;
8. missing rollback, observability, or operational handling;
9. missing tests for the changed risk.

Fix blocking and high-severity findings, rerun affected checks, and repeat the review. Classify unresolved lower-severity items as risks rather than silently ignoring them.

## Phase 8: Release readiness

Before calling a change production ready, confirm the applicable items:

- migration is forward-safe and deployment order is documented;
- environment variables/secrets are declared without values in source;
- health/readiness, logs, metrics, traces, and alerts cover new behavior;
- worker retries/dead letters and provider failures are visible;
- backup/restore, rollback, and data-repair impact is understood;
- staging/sandbox smoke test is run when available;
- performance/security/contract gates for the risk class passed;
- frontend migration or compatibility is complete;
- external checks not run are explicitly listed.

“Production ready” means the code and local/CI evidence meet the documented gates. It does not mean the agent can invent credentials, approve a business decision, or claim a live provider/deployment check it could not perform.

## Universal task acceptance criteria

The task is complete when:

- the requested behavior works against real application code/data, not only mocks;
- scope, auth, RBAC, and plan checks are correct and tested;
- API/schema/database/event/config changes are documented;
- retries, idempotency, transactions, state, and failure behavior are safe for the task;
- tests and quality checks pass at the required depth;
- review findings are fixed or explicitly recorded as non-blocking risks;
- the final handoff gives reproducible evidence and names any external work remaining.

## Final response template

```text
Status: complete | blocked | needs-review
Outcome: <what works or what was diagnosed>
Changed: <files/modules and purpose>
Contract: <endpoints, schemas, migrations, events, config>
Verification: <exact commands and results>
Production readiness: <passed gates; external checks still required>
Risks: <remaining limitations>
Next: <smallest sensible follow-up>
```

## Examples

### Build an API

```text
Build customer search and detail APIs and connect the existing frontend customer screen. Follow the repository backend workflow, reconcile the mock types, add tenant/RBAC protection, OpenAPI, migration/indexes if needed, integration tests, and production-readiness verification.
```

### Fix a bug

```text
Fix the duplicate payment issue when the frontend retries the request. Reproduce it, add a failing regression test, implement idempotency transactionally, review related invoice/ledger behavior, and run the appropriate production checks.
```

### Any backend task

```text
<Describe the requested backend outcome in plain language. The repository workflow must discover the design, implement it, test it, review it, fix findings, and report blockers without requiring me to provide internal agent roles.>
```

