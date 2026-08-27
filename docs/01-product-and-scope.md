# Product scope and acceptance criteria

## Product goal

AI-BOS is a multi-tenant SaaS business operating system for small and medium-sized local businesses. It combines day-to-day operations (customers, products, stock, orders, invoices, and payments) with dashboards, notifications, subscriptions, and a conversational assistant.

The backend must serve the existing Next.js PWA on mobile and desktop browsers and must remain usable when a business has unreliable connectivity or an external provider is temporarily unavailable.

## In scope for MVP

- User registration, email verification, login, logout, refresh sessions, and password reset.
- A business/workspace with Owner, Staff, and platform Admin roles.
- Customer profiles, search, and customer ledger.
- Product catalog with SKU, price, quantity, and low-stock threshold.
- Stock movements and current inventory.
- Orders with status tracking.
- Draft and issued invoices, PDF receipts, invoice sharing links, and immutable invoice line snapshots.
- Manual customer payments, partial payments, outstanding dues, and reminders.
- Daily/weekly/monthly sales summaries and inventory/customer insights.
- AI chat with safe read tools and confirmed write actions.
- Free/Pro/Premium plans, Razorpay subscription checkout, and verified webhooks.
- Email and WhatsApp notifications where the business has configured and consented to those channels.
- Docker-based deployment, CI/CD, backups, logs, metrics, traces, and audit trails.

## Explicitly out of scope initially

- ERP manufacturing, bills of material, and production planning.
- Payroll and HR.
- Full accounting/tax compliance automation or multi-country tax rules.
- Offline conflict-free replication.
- Native mobile applications.
- Arbitrary custom AI agents or unrestricted database access.

## Actors and permissions

| Actor | Primary capabilities |
| --- | --- |
| Owner | Business settings, team, catalog, inventory, orders, invoices, payments, analytics, assistant, billing |
| Staff | Assigned operational work; permissions are configurable but default to customers, orders, and inventory reads/updates; no billing or member administration |
| Admin | Platform-level support, user/business moderation, plans, webhook/provider health, and system monitoring; no implicit access to tenant business data without an audited support action |

## Core user journeys

### Onboarding

1. User registers and verifies email.
2. User creates a business and becomes Owner.
3. User selects a plan (Free by default), configures business profile, currency, invoice numbering, and notification preferences.
4. User optionally invites staff.

### Sell and collect

1. Staff searches or creates a customer.
2. Staff adds products to an order; the service validates available stock.
3. Owner/staff confirms the order, creating stock movements according to the agreed fulfillment policy.
4. User creates and issues an invoice; the invoice receives a unique number and a PDF job.
5. User records a full or partial payment; allocations update the customer ledger and invoice status.
6. User sends a reminder or shares the invoice through an approved channel.

### Ask the assistant

1. User asks a natural-language question.
2. Backend sends tenant-scoped context and a small tool set to the model.
3. Read tools execute immediately after normal authorization.
4. A write tool returns a preview and requires explicit confirmation.
5. The confirmed command is executed by a normal domain service and is shown in the conversation audit trail.

## Acceptance criteria for the first real release

- A new business can complete onboarding without a support/admin action.
- Two businesses cannot read, mutate, or infer one another's records through IDs, filters, exports, AI tools, or files.
- Creating an invoice and its stock/payment effects is atomic, retry-safe, and auditable.
- Issued invoices cannot be silently edited; corrections use void/credit-note behavior once that product decision is finalized.
- A duplicate Razorpay webhook or client retry does not duplicate a subscription, payment, notification, or stock movement.
- A failed email/WhatsApp/PDF job is visible, retried, and eventually dead-lettered without blocking the API request.
- The frontend can replace each mock repository with the API without changing business meaning of its TypeScript models.

## Assumptions to validate with the product owner

These are intentionally tracked in [open decisions](12-open-decisions.md): stock deduction timing, tax/GST scope, store/location support, invoice edit rules, payment collection versus manual recording, plan limits, WhatsApp onboarding, and data retention.

