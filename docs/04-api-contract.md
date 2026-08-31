# REST API contract

The canonical machine-readable contract will live at `api/openapi.yaml`. This document defines conventions and the initial resource surface; endpoint details must be reconciled with the frontend's existing TypeScript models before implementation.

## Base conventions

- Base URL: `/api/v1`.
- JSON request/response bodies use `camelCase`; database fields remain `snake_case`.
- `Content-Type: application/json`; PDF endpoints use `application/pdf` or a signed R2 URL.
- Every response includes `requestId` either in the body envelope or `X-Request-Id` header.
- List endpoints use cursor pagination: `?limit=25&cursor=...`; default 25, maximum 100.
- Dates are UTC ISO-8601 strings. Money uses `{ amountMinor: 125000, currency: "INR" }`.
- Mutating requests support `Idempotency-Key`; conflicting reuse returns `409`.
- Updates support `If-Match`/resource version where concurrent edits could overwrite data.

## Response and error shapes

```json
{
  "data": { "id": "uuid", "name": "Example" },
  "meta": { "requestId": "req_123" }
}
```

```json
{
  "error": {
    "code": "validation_failed",
    "message": "One or more fields are invalid",
    "fieldErrors": { "email": "must be a valid email" },
    "requestId": "req_123"
  }
}
```

Use `400` malformed input, `401` unauthenticated, `403` unauthorized/plan denied, `404` not found in the active tenant, `409` state/idempotency conflict, `413` payload too large, `422` domain validation, `429` rate limit, `500` unexpected error, and `503` dependency unavailable.

## Operational endpoints

The unversioned operational endpoints are defined in `api/openapi.yaml` and require no authentication:

```text
GET /healthz   # process liveness only
GET /readyz    # PostgreSQL and Redis readiness
```

Both return the effective request ID in `X-Request-Id` and the JSON envelope. `/readyz` returns `503 dependency_unavailable` when either dependency fails its bounded health check. The response identifies dependency status without exposing connection errors, hosts, credentials, or other secrets.

## Initial endpoint inventory

### Identity and business

```text
POST   /auth/register
POST   /auth/login
POST   /auth/logout
POST   /auth/refresh
POST   /auth/verify-email
POST   /auth/forgot-password
POST   /auth/reset-password
GET    /me
GET    /businesses
POST   /businesses
GET    /businesses/{businessId}
PATCH  /businesses/{businessId}
GET    /businesses/{businessId}/members
POST   /businesses/{businessId}/invitations
PATCH  /businesses/{businessId}/members/{memberId}
DELETE /businesses/{businessId}/members/{memberId}
```

### Customers

```text
GET    /businesses/{businessId}/customers?q=&cursor=
POST   /businesses/{businessId}/customers
GET    /businesses/{businessId}/customers/{customerId}
PATCH  /businesses/{businessId}/customers/{customerId}
DELETE /businesses/{businessId}/customers/{customerId}   # archive semantics
GET    /businesses/{businessId}/customers/{customerId}/ledger
```

### Catalog and inventory

```text
GET    /businesses/{businessId}/products
POST   /businesses/{businessId}/products
GET    /businesses/{businessId}/products/{productId}
PATCH  /businesses/{businessId}/products/{productId}
DELETE /businesses/{businessId}/products/{productId}     # archive semantics
GET    /businesses/{businessId}/inventory
POST   /businesses/{businessId}/inventory/adjustments
GET    /businesses/{businessId}/products/{productId}/movements
```

### Sales, invoices, and payments

```text
GET    /businesses/{businessId}/orders
POST   /businesses/{businessId}/orders
GET    /businesses/{businessId}/orders/{orderId}
PATCH  /businesses/{businessId}/orders/{orderId}
POST   /businesses/{businessId}/orders/{orderId}/confirm
POST   /businesses/{businessId}/orders/{orderId}/cancel
GET    /businesses/{businessId}/invoices
POST   /businesses/{businessId}/invoices
GET    /businesses/{businessId}/invoices/{invoiceId}
PATCH  /businesses/{businessId}/invoices/{invoiceId}    # drafts only
POST   /businesses/{businessId}/invoices/{invoiceId}/issue
POST   /businesses/{businessId}/invoices/{invoiceId}/void
GET    /businesses/{businessId}/invoices/{invoiceId}/pdf
POST   /businesses/{businessId}/invoices/{invoiceId}/share
GET    /businesses/{businessId}/payments
POST   /businesses/{businessId}/payments
GET    /businesses/{businessId}/payments/{paymentId}
POST   /businesses/{businessId}/payments/{paymentId}/allocations
```

### Dashboard and assistant

```text
GET    /businesses/{businessId}/dashboard/summary?from=&to=
GET    /businesses/{businessId}/dashboard/sales?granularity=day|week|month
GET    /businesses/{businessId}/dashboard/inventory
GET    /businesses/{businessId}/dashboard/customers
GET    /businesses/{businessId}/conversations
POST   /businesses/{businessId}/conversations
GET    /businesses/{businessId}/conversations/{conversationId}
POST   /businesses/{businessId}/conversations/{conversationId}/messages   # SSE
POST   /businesses/{businessId}/assistant/actions/{actionId}/confirm
```

### Plans and integrations

```text
GET    /plans
GET    /businesses/{businessId}/subscription
POST   /businesses/{businessId}/subscription/checkout
POST   /businesses/{businessId}/subscription/cancel
GET    /businesses/{businessId}/notifications
PATCH  /businesses/{businessId}/notification-preferences
POST   /webhooks/razorpay
```

## Authentication contract

Use secure cookies for browser refresh sessions. Access tokens may be short-lived JWTs or a server-side session token; the choice must be consistent with the frontend. Do not put refresh tokens in localStorage. Include the active business ID in a trusted server-side session/context, or require it as a path/header value and authorize it against membership every time.

## SSE assistant contract

`POST /messages` returns an SSE stream with events such as `message.started`, `text.delta`, `tool.preview`, `confirmation.required`, `tool.completed`, `message.completed`, and `error`. Each event has a conversation/message ID and can be replayed or resumed using a request ID. The server closes idle streams and records the final persisted state.

## Contract workflow

1. Extract the frontend mock types and examples.
2. Add schemas and examples to `api/openapi.yaml`.
3. Generate Go server interfaces and TypeScript client types.
4. Run contract tests in CI against handlers and a frontend fixture.
5. Treat breaking changes as a versioned migration, not an incidental handler edit.
