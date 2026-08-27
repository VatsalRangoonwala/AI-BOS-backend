# Open product and technical decisions

Resolve these decisions before the affected schema/API is considered stable. Each item includes a safe default so implementation can continue in a branch while awaiting confirmation.

| Decision | Why it matters | Recommended default |
| --- | --- | --- |
| When is stock deducted? | Changes order/invoice transaction and cancellation behavior | Reserve on order confirmation; deduct on fulfillment; compensate on cancellation |
| Are issued invoices editable? | Determines auditability and schema transitions | No; drafts editable, issued invoices void/credit-note only |
| Tax/GST fields and calculations? | Affects totals, PDF, reporting, and future compliance | Store tax metadata and line-level amounts; support one configured tax regime initially |
| Credit notes/returns? | Needed for real-world corrections | Add after core issue/payment flow, but preserve extensible invoice states |
| Multiple stores/locations? | Changes every inventory uniqueness/index | Start single location; model nullable `location_id` if expansion is near-term |
| Negative stock allowed? | Affects sale validation and alerts | Reject by default, business setting can explicitly enable |
| Currency scope? | Affects money and subscription pricing | One business currency, INR MVP; no FX conversion |
| Payment collection versus manual recording? | Separates customer receivables from SaaS billing | Manual recording MVP; online customer collection later |
| Invoice sharing channels? | Provider templates/consent/onboarding | Email first; WhatsApp only with opt-in and approved templates |
| WhatsApp provider/onboarding? | Determines integration contract and support | Meta WhatsApp Business Cloud API with sandbox/staging credentials |
| Plan limits? | Required for entitlement enforcement and pricing | Put limits in database-configured plan rows, not code constants |
| Free plan behavior after limit? | UX and billing conversion | Read-only/upgrade prompt; never delete or hide data |
| Staff permissions? | Security and workflow | Owner grants named capabilities; conservative defaults |
| Admin support access? | Privacy and audit | Time-bound, reason-coded, audited impersonation/support session |
| Customer deletion/retention? | Ledger and privacy conflict | Archive operational records; define export/delete policy with legal review |
| AI model/configuration? | Cost, latency, and regression behavior | Provider/model behind config; pin versions per environment and evaluate changes |
| AI write confirmation UX? | Prevents accidental financial/external actions | Human-readable preview, explicit confirm button, short expiry, version check |
| API auth transport? | Frontend implementation and CSRF model | Secure cookie refresh session plus short-lived access token |
| Availability/DR RPO? | Infrastructure cost and recovery promise | Keep one-hour RTO; choose explicit RPO (recommend 15 minutes with PITR) |

## Decision record format

For each resolved item, record date, decision, alternatives considered, consequences, migration impact, and owner. If a decision changes an API or persisted state, add a versioned migration note and update the OpenAPI/ERD documents in the same change.

