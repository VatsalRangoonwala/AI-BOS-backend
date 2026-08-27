# AI assistant design

The assistant is an orchestration layer over normal AI-BOS application services. It is not a second business-logic implementation and it never receives direct database access.

The implementation follows the official OpenAI function-calling flow: send a model request with JSON-schema tools, receive one or more tool calls, execute each call in application code, send tool outputs back using the call identifier, and then produce the final response. See [OpenAI Docs: Function calling](https://developers.openai.com/api/docs/guides/function-calling).

## Components

- `ConversationService`: persists conversations, messages, model metadata, and usage.
- `PromptBuilder`: creates a small, tenant-safe context from business settings and recent messages.
- `ToolRegistry`: allowlisted tool definitions with strict input schemas and descriptions.
- `ToolExecutor`: validates arguments, checks authorization/entitlements, executes application use cases, and redacts output.
- `ConfirmationService`: stores short-lived action previews and binds confirmation to user, business, tool, arguments hash, and expiry.
- `StreamBroker`: converts model deltas and lifecycle events to SSE.
- `AssistantPolicy`: quotas, max loops, timeouts, model fallback, and safety policy.

## Tool catalog

Start small and add tools only when a user journey needs them.

### Read tools (no confirmation)

- `search_customers`
- `get_customer_ledger`
- `find_products`
- `get_inventory_status`
- `get_sales_summary`
- `get_outstanding_dues`
- `get_recent_orders`

### Write tools (preview then confirmation)

- `prepare_invoice`
- `record_payment`
- `create_order`
- `adjust_stock`
- `send_payment_reminder`

The model may prepare an action, but the server owns final validation, pricing, stock availability, invoice numbering, payment allocation, and notification consent.

## Tool execution flow

1. Authenticate the request and resolve business membership.
2. Load conversation state and apply plan quota.
3. Build tools appropriate to the role and current business.
4. Call the OpenAI Responses API with bounded context and timeout.
5. For each function call, parse strict JSON arguments and reject unknown fields.
6. Execute read tools or create a write preview; persist the tool call and result.
7. For a write preview, emit `confirmation.required` with a human-readable summary and expiring action ID.
8. On confirmation, recompute/validate the action and invoke the same domain service as REST.
9. Send sanitized tool output back to the model and continue until final text or loop limit.
10. Persist final text, provider IDs, latency, token usage, and outcome.

## Confirmation requirements

Confirmation is mandatory for issuing/voiding an invoice, changing stock, recording or allocating a payment, creating/cancelling an order, sending a reminder, or any action with external side effects. Confirmation must expire quickly, be single-use, and fail if the underlying data version changed.

## Streaming and failure behavior

Use SSE events: `message.started`, `text.delta`, `tool.started`, `tool.preview`, `confirmation.required`, `tool.completed`, `message.completed`, and `error`. Persist enough state to recover after a browser disconnect. If OpenAI times out or fails, show a retry-safe error and keep the conversation usable; do not duplicate a confirmed write on retry.

## Context and privacy

Send only the minimum tenant-scoped records needed for the current request. Summarize long conversations, cap retrieved rows, and never include passwords, refresh tokens, provider credentials, or unnecessary customer PII. Store provider request IDs for support, but redact prompt content from ordinary logs.

## Quotas and cost controls

Track calls, input/output tokens, estimated cost, latency, tool count, and errors per business and plan period. Enforce Free/Pro/Premium limits server-side, with a clear `assistant_quota_exceeded` error and dashboard usage visibility. Add circuit breaking when provider errors or spend exceed configured thresholds.

## Evaluation and release gate

Maintain a versioned dataset of representative questions and expected tool/action outcomes: sales summaries, ambiguous customer names, insufficient stock, partial payments, unauthorized staff actions, prompt injection, and provider failures. A prompt/tool/model change requires regression evaluation before production rollout. Measure factuality, correct tool selection, argument validity, authorization failures, confirmation behavior, latency, and cost.

