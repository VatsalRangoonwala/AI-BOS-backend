# Domain model and database design

## Tenancy model

`businesses` is the tenant boundary. Users belong to businesses through `memberships`, allowing one user to work with multiple businesses. Every tenant-owned table includes `business_id`, and repositories require the business ID as an explicit argument.

Use application-level authorization on every query and mutation. PostgreSQL row-level security may be added as defense in depth after the connection/session strategy is proven; it must not be treated as a substitute for service authorization.

## Core entities

| Table | Important fields and constraints |
| --- | --- |
| `users` | `id`, email (case-normalized unique), password hash, name, status, verified_at, timestamps |
| `businesses` | `id`, legal/display name, phone, address, timezone, currency, invoice prefix/sequence, status |
| `memberships` | business/user IDs, role, status, invited/accepted timestamps; unique `(business_id,user_id)` |
| `refresh_sessions` | user ID, token hash, device metadata, expiry, revoked_at, last_seen_at |
| `customers` | business ID, name, normalized phone, email, address, notes, archived_at; indexed search fields |
| `products` | business ID, name, SKU (unique per business), unit, sale price, cost price, tax metadata, reorder level, archived_at |
| `inventory_balances` | business/product IDs, on_hand, reserved, version; unique `(business_id,product_id[,location_id])` |
| `stock_movements` | product/business IDs, signed quantity, movement type, source type/ID, actor, idempotency key, occurred_at |
| `orders` | business/customer IDs, status, totals, currency, source, created_by, version |
| `order_items` | order ID, product ID, description/SKU/price/quantity snapshots, line totals |
| `invoices` | business/customer/order IDs, immutable number, status, issued_at/due_at, totals, currency, PDF file ID, version |
| `invoice_items` | invoice ID, product ID, description/SKU/price/quantity/tax/discount snapshots |
| `payments` | business/customer ID, method, amount, currency, reference, received_at, status, idempotency key |
| `payment_allocations` | payment/invoice IDs, amount; unique payment/invoice pair and check against remaining balances |
| `plans` | stable plan key, limits/entitlements, price, currency, provider identifiers, active flag |
| `subscriptions` | business ID, plan, provider/customer/subscription IDs, status, period, cancel_at_period_end |
| `usage_counters` | business/metric/period, consumed, limit snapshot; unique period key |
| `notifications` | business/user, channel, template, destination, status, provider ID, attempts, error |
| `outbox_events` | aggregate type/ID, event type, payload, available_at, attempts, locked_at, processed_at |
| `files` | business ID, object key, media type, size, checksum, visibility, lifecycle status |
| `conversations` | business/user, title, model metadata, created/updated timestamps |
| `messages` | conversation ID, role, content, response/provider IDs, token/cost metadata |
| `assistant_tool_calls` | message ID, tool name, validated args/result, authorization/confirmation status, timings |
| `webhook_events` | provider, external event ID (unique), raw payload hash, received/processed status, error |
| `audit_logs` | business/user, action, aggregate, redacted metadata, request ID, IP/user agent, timestamp |
| `idempotency_keys` | business/user, key, endpoint, request hash, status, response snapshot, expiry; unique scope/key |

## Invariants and transactions

### Inventory

- `on_hand` is derived from successful stock movements and may be cached for reads.
- A movement records a reason and source document; never update stock silently.
- Confirm/fulfill behavior must be chosen once and enforced consistently.
- Concurrent deductions lock the balance row or use an atomic versioned update.
- Negative stock is rejected unless an explicit business setting permits it.

### Invoices and payments

- Issued invoice number is unique per business and never reused.
- Invoice line descriptions, prices, tax, and discounts are snapshots.
- Invoice totals are calculated server-side from lines; client totals are advisory.
- Payment allocations cannot exceed payment amount or invoice outstanding amount.
- Invoice status is derived from issue/void and allocated amounts, not freely editable.
- Corrections to issued documents use void/credit-note behavior after that decision is finalized.

### Orders

- State transitions are explicit and audited.
- Cancellation after stock deduction creates compensating movements rather than deleting history.
- Order and invoice references are nullable only where the workflow permits drafts.

## Indexing and retention

Index every tenant foreign key, status/date pair used by dashboards, normalized customer phone, product SKU, invoice number, due date, outbox availability, and webhook provider/event ID. Use full-text/trigram search only after measuring query needs.

Partition very large append-only tables by time only after production evidence. Retain audit, ledger, invoice, and payment history according to the published retention policy; expire sessions, idempotency records, and raw webhook payloads according to shorter operational windows.

## Migration rules

- Every schema change is a reviewed, numbered forward migration.
- Prefer expand/ backfill/ contract for zero-downtime changes.
- Never make a large backfill part of the API deploy transaction.
- CI runs migrations from empty and from the latest production-like snapshot.
- Destructive column/table removal requires a separate release and verified backup.

