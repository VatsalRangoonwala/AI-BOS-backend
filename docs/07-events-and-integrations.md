# Events, workers, and external integrations

## Outbox pattern

When a domain transaction changes business state, it writes an `outbox_events` row in the same PostgreSQL transaction. A worker claims available events with a lease, publishes/executes the side effect, records attempts and provider IDs, and retries safely. This avoids the dual-write failure where an invoice commits but its PDF or notification is lost.

Example events:

```text
business.created
member.invited
order.confirmed
order.cancelled
invoice.issued
invoice.voided
payment.recorded
inventory.low
subscription.changed
```

Handlers must be idempotent by event ID and aggregate/version. Keep a dead-letter state with a human-visible failure reason and a replay operation restricted to Owner/Admin operations staff.

## Worker jobs

| Job | Trigger | Retry/failure policy |
| --- | --- | --- |
| Invoice PDF | `invoice.issued` | Exponential retry; dead-letter after bounded attempts; signed R2 URL when complete |
| Email | notification event | Provider retry/backoff; suppress invalid destinations |
| WhatsApp | reminder/share event | Respect opt-in and provider template rules; retry transient errors |
| Low-stock alert | stock movement crossing threshold | Deduplicate per product/window |
| Payment reminder | schedule/due-date scan | Idempotency by invoice/channel/date |
| Analytics rollup | periodic or event-driven | Rebuildable summaries; never lose source ledger |
| Razorpay reconciliation | scheduled/webhook | Verify provider state and record discrepancy |
| Cleanup | schedule | Expire sessions, idempotency keys, temporary files, and stale previews |

## Cloudflare R2

Keep buckets private. Generate object keys using tenant and resource UUIDs, store checksum/media type/size in `files`, upload with server-side validation, and issue short-lived signed download URLs. Do not accept arbitrary client-provided object keys. Define lifecycle rules for temporary PDFs and abandoned uploads.

## Invoice PDF generation

Render from a versioned server template and persisted invoice snapshots, not live product rows. Include invoice number, business profile, customer, line items, totals, currency, payment status, and issue/due dates. Record template version and generation status so an old invoice can be reproduced.

## Resend email

Centralize templates, sender domains, unsubscribe/consent rules, provider IDs, and delivery webhooks. Store only the minimum rendered data needed for retry/audit. Never let a request wait for email delivery.

## WhatsApp Business API

Start with approved templates for invoice sharing and payment reminders. Store recipient consent, template/version, provider message ID, delivery state, and opt-out events. Treat provider callbacks as untrusted input and verify signatures where supported.

## Razorpay subscriptions

Create checkout/subscription records before redirecting the user. Verify webhook signatures over the raw body, deduplicate external event IDs, handle out-of-order events, and make webhook processing transactional. Reconcile periodically because a webhook can be delayed or lost. Never mark a subscription active from an unverified browser callback.

## OpenAI

Keep provider calls behind an interface with timeout, retry, model/config version, request ID, usage, and error metrics. API keys live only in server secrets. See [AI assistant design](06-ai-assistant.md) for tool authorization and confirmation.

## Notification preferences

Business and customer preferences must be checked immediately before enqueueing and again before sending. Record channel, template, destination hash/masked destination, consent source, and outcome for support without exposing unnecessary PII.

