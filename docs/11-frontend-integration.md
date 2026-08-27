# Frontend mock-data migration

The frontend already proves the interaction design with dummy data. The migration should preserve that UX while replacing one data boundary at a time.

## Build a mock-to-API matrix

For each screen and action, record:

| Field | Example |
| --- | --- |
| Screen/component | Dashboard summary |
| Current mock source | `mockDashboard.ts` |
| TypeScript type | `DashboardSummary` |
| Read endpoint | `GET /businesses/{id}/dashboard/summary` |
| Mutations | None / record payment / issue invoice |
| Query state | Date range, cursor, search, filters |
| Empty/loading/error states | Required UI behavior |
| Permission/plan | Owner, Staff, Pro feature |
| Acceptance fixture | Deterministic business/customer/product data |

Do this before implementing handlers; it exposes fields the written product spec does not define.

## Recommended migration order

1. Generated API client, environment-based API URL, request ID/error handling.
2. Auth and active-business context.
3. Customers and products.
4. Inventory and stock alerts.
5. Orders, invoices, PDFs, and payments.
6. Dashboard queries.
7. Notifications and subscription screens.
8. AI chat streaming and confirmation UI.

Keep mock repositories behind the same interface as the API repository during the transition. A feature flag can route a screen between mock and API in development, but production must fail loudly when the API is unavailable rather than silently showing stale dummy data.

## Client behavior requirements

- Generate TypeScript types/client from OpenAPI rather than hand-copying response types.
- Handle cursor pagination, `401` refresh/retry once, `409` version/idempotency conflicts, `422` field errors, `429` retry hints, and `503` degraded states.
- Show mutation pending/success/failure explicitly; do not optimistically display financial or stock changes without reconciliation.
- Send an idempotency key for invoice issue, order confirmation, payment recording, subscription checkout, and assistant confirmation.
- Use SSE reconnect/resume for assistant streams and render `confirmation.required` as a first-class UI state.
- Keep server-generated totals/statuses authoritative.

## Contract fixtures

Use the same deterministic fixtures in backend integration tests, OpenAPI examples, and frontend storybook/tests. Include empty lists, large lists, fractional stock, overdue/partial invoices, notification failures, and permission-denied responses.

## Release strategy

Ship API-backed screens behind a frontend feature flag, compare response shape/metrics with the mock path in staging, then remove the mock path after one stable release. Do not maintain divergent business rules in mock helpers once the real endpoint exists.

