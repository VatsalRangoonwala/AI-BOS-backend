# Security and multi-tenancy

## Threat model

Protect against credential theft, session replay, cross-tenant IDOR, privilege escalation, fraudulent payment/webhook events, malicious file uploads, notification abuse, SQL injection, XSS/CSRF, prompt injection, data exfiltration through AI tools, and provider-secret leakage.

## Authentication

- Hash passwords with Argon2id using parameters benchmarked on the production instance.
- Normalize emails for lookup and uniqueness while preserving display form.
- Require email verification before sensitive business actions.
- Use short-lived access tokens and rotating, hashed refresh sessions.
- Revoke a session on logout, password reset, suspicious reuse, or admin action.
- Rate-limit registration, login, verification, reset, and assistant endpoints per IP and account.
- Do not log passwords, tokens, OTPs, provider secrets, or raw payment credentials.

## Authorization and tenant isolation

The request context must carry authenticated user, active business, membership, role, and entitlements. Every service method accepts an authorization context and business ID. Repository methods never expose an unscoped `GetByID` for tenant data.

| Operation | Owner | Staff (default) | Admin |
| --- | ---: | ---: | ---: |
| Read operational data | Yes | Assigned business | Audited support only |
| Create/update customers | Yes | Yes | Audited support only |
| Adjust stock | Yes | Configurable; default yes | Audited support only |
| Issue/void invoice | Yes | Configurable; default issue yes, void no | Audited support only |
| Record payments | Yes | Yes | Audited support only |
| Manage members/business settings | Yes | No | Platform scope |
| Manage subscription | Yes | No | Platform support/reconciliation |
| Configure AI/provider keys | Yes | No | Platform operations |

Admin support access must be a separate, time-bound, reason-coded, audited capability; an Admin role must not silently bypass tenant checks.

## Web and API controls

- TLS everywhere; redirect or reject plaintext traffic.
- Strict CORS allowlist for deployed frontend origins.
- CSRF tokens/double-submit protection for cookie-authenticated mutations.
- JSON body size and upload limits; timeouts on every handler and provider call.
- Parameterized SQL through `sqlc`; no string-built queries.
- HTML/PDF templates escape untrusted text; apply a restrictive Content Security Policy.
- Validate URLs, MIME types, image/PDF dimensions, and malware-scan uploads before exposing them.
- Security headers: HSTS, frame restrictions, content type sniffing protection, and referrer policy.
- Rate-limit by route, IP, user, and business; use Redis with a safe fallback when unavailable.

## Data protection

Classify email, phone, address, invoices, payment references, and conversation content as sensitive business data. Encrypt disks and managed database backups, keep R2 objects private, issue short-lived signed URLs, and redact PII from logs/traces. Define retention and deletion behavior before launch, including export and account deletion.

## Auditability

Audit authentication events, membership changes, product/stock mutations, order/invoice/payment transitions, subscription/webhook events, AI tool calls, support access, and notification sends. Store actor, business, action, aggregate, request ID, outcome, and redacted before/after metadata.

## AI-specific safeguards

- Treat user text and retrieved records as untrusted content, not instructions.
- Expose only allowlisted, schema-validated tools.
- Re-check tenant, role, plan, and object ownership inside each tool implementation.
- Require explicit confirmation for financial, inventory, messaging, or irreversible actions.
- Never place secrets or unrestricted SQL in model context.
- Bound context size, tool-loop count, spend, and execution time.
- Provide a safe fallback response when the model/provider is unavailable.

