## Context

See `proposal.md` — Why. The design-relevant facts about the current code:

- Five gateways coexist behind ad-hoc, per-gateway code: `controller/topup*.go`, `controller/subscription_payment_*.go`, `setting/payment_*.go`, `service/epay.go`, `service/waffo_pancake.go`. There is no gateway abstraction to preserve — each has its own controller, settings module, and `model` settlement function.
- Settlement already follows a consistent, safe shape in `model/topup.go` (`RechargeEpay`, `RechargeWaffo`, ...): open a transaction, `lockForUpdate` the order row by `trade_no`, verify `PaymentProvider`, verify status, save the order, then `creditTopUpQuota` which enforces the wallet ceiling with a conditional `UPDATE`. `LockOrder`/`UnlockOrder` is documented as an in-process optimization only. SePay must reuse this shape verbatim.
- `top_ups.trade_no` is already `unique` and indexed. Statuses `pending`/`success`/`failed`/`expired` already exist in `common/constants.go`.
- Quota conversion must go through `common.WalletQuotaFromDecimalStrict` / `common/quota_math.go`; bare casts are prohibited by project rules.
- Every gateway is gated behind `operation_setting.IsPaymentComplianceConfirmed()`.
- `service/system_task.go` already runs a periodic system-task runner that a new expiry sweep can register with.
- SePay's integration model is fundamentally different from all five removed gateways: there is no hosted checkout and no redirect. The user transfers money from their own banking app; SePay observes the bank account and posts a balance-change webhook. Matching is by transfer memo (transfer description) only.

## Goals / Non-Goals

**Goals:**

- One payment provider end to end, with no gateway-selection branching left in backend or frontend.
- Settlement correctness identical in strength to the existing Epay path: DB row lock, provider check, status check, idempotency across instances, wallet ceiling enforced atomically.
- Match an incoming transfer to at most one order, deterministically, without scanning the order table with a wildcard `LIKE`.
- No schema migration that rewrites or drops historical rows.

**Non-Goals:**

- No pluggable payment-gateway abstraction. One provider does not justify an interface; adding a second later is the moment to extract one.
- No automatic refund, partial-payment top-up, or surplus crediting. Underpaid and overpaid transfers are logged for manual handling.
- No multi-currency support. VND is the only settlement currency; USD stays the internal accounting unit.
- No change to the wallet-balance subscription path, redemption codes, or quota accounting.

## Decisions

### D1: Reuse `trade_no` as the transfer memo instead of adding a memo column

The memo must be unique, bank-safe (uppercase ASCII letters and digits only — banks strip punctuation, diacritics, and sometimes lowercase), and cheap to look up. `top_ups.trade_no` is already unique and indexed.

**Decision:** Generate SePay trade numbers in a bank-safe fixed format — prefix `SP`, then a user-derived segment, then random uppercase base32 characters — and use the trade number itself as the transfer memo. Subscription orders use the same generator with a distinct prefix (`SS`) so the two order tables never collide.

**Alternatives considered:** a separate `transfer_memo` column on both tables — rejected because it adds a migration and a second uniqueness constraint to guard for no behavioral gain. Using the numeric order id — rejected because ids are guessable and short, and a mistyped memo could land on a stranger's order.

**Consequence:** the existing `unique` constraint on `trade_no` is what enforces memo uniqueness; the generator retries on the (astronomically unlikely) insert conflict rather than pre-checking.

### D2: Match a webhook to an order by extracting the memo, not by `LIKE '%...%'`

SePay's `content` field is free text: banks prepend and append their own noise around the user's description.

**Decision:** normalize `content` (uppercase, strip everything that is not `A-Z0-9`), then apply a regex for the fixed memo shape to extract candidate memos, then look each candidate up by exact `trade_no` equality on the indexed column. If the extraction yields more than one distinct candidate that resolves to a pending order, settle nothing and log an error naming every candidate and the SePay `referenceCode` — a transfer that could plausibly pay two orders is an operator problem, not something to guess at.

**Alternatives considered:** `WHERE content LIKE '%' || trade_no || '%'` style scanning — rejected: unindexable, and would need a full scan of pending orders per webhook.

### D3: One webhook endpoint serving both order types, resolved by memo prefix

`/api/sepay/webhook` handles both wallet top-up and subscription orders. The memo prefix (`SP` vs `SS`) selects which table to resolve against, so exactly one order type is ever consulted per candidate.

**Alternative considered:** two endpoints with two API keys — rejected because SePay configures the webhook per bank account, not per purpose, and both flows watch the same account.

### D4: Expiry is derived, then materialized by a sweep

An order's expiry instant is `create_time + expiry_window`. The webhook path checks this derived value directly, so a transfer arriving one second after expiry is never credited even if the sweep has not run yet. A periodic sweep registered with the existing system-task runner materializes the `expired` status so the UI and admin views agree with the webhook.

**Consequence:** no `expire_time` column is needed; changing the configured window changes the effective deadline of already-pending orders. This is acceptable and is stated in the spec (only the *amount* is frozen at creation).

**Sweep scope:** the sweep expires only orders whose provider is SePay. Legacy pending orders from removed gateways are deliberately left alone (see `specs/payments/legacy-gateway-removal`), so an operator can still see and manually settle them.

### D5: Amount frozen at creation, in whole Dong

`payable_vnd = round(amount × price × topup_group_ratio × discount)` computed with `decimal` at order creation and stored in the existing `money` column. `price` is reused as-is from the existing general payment settings, reinterpreted as **Dong per USD**, with a default of **1000** — a $10 top-up is payable as 10,000 Dong. No separate exchange-rate setting is introduced; the operator sets one number and it doubles as the price list. The webhook compares the reported `transferAmount` against this stored value; it never recomputes from current settings. All conversion goes through `common.WalletQuotaFromDecimalStrict` and the `common/quota_math.go` helpers, so an absurd requested amount saturates into a rejection at validation rather than wrapping.

Storing whole Dong in the existing `float64 money` column is safe for realistic amounts (VND totals stay far below 2^53), and avoids a column type migration. Comparison against `transferAmount` is done on integers after converting both sides through `decimal`.

### D6: Webhook authentication is an exact `Apikey` header match in constant time

SePay authenticates with `Authorization: Apikey <key>`. Compare with `crypto/subtle.ConstantTimeCompare` after parsing the scheme. Reject with 401 on any mismatch, on a disabled integration, and on an unconfigured key — never fall through to "no key configured means accept". The endpoint sits behind the existing `anonymousRequestBodyLimit` middleware like the webhooks it replaces.

**No IP allowlist by default.** SePay does not publish a stable source range, and a wrong allowlist silently loses payments; the API key is the control. The endpoint is rate-limited by the existing anonymous limiter.

### D7: Response semantics — 200 for "understood, nothing to do"

The webhook returns success for: credited, already credited, underpaid, overpaid-but-credited, outgoing transfer, no matching memo, expired order, ambiguous memo. Each of the non-crediting cases writes a warning or error log carrying the SePay `id` and `referenceCode`. Only authentication failure (401) and an actual internal fault (500) are non-success, so SePay retries only when a retry could help.

### D8: Underpayment and overpayment are logged, not auto-handled

An underpaid order stays pending (and will expire normally); an overpaid order credits exactly what was ordered and logs the surplus. Both are surfaced to the operator through the existing warning-log path rather than through a new reconciliation UI.

**Alternative considered:** crediting proportionally on underpayment — rejected: it silently converts a user error into a partial purchase and complicates the subscription flow, where a plan is indivisible.

### D9: Client learns the outcome by polling, not by redirect

There is no return URL in a bank-transfer flow. The payment panel polls `GET /api/user/sepay/order/:trade_no` on a fixed interval while the panel is open, and stops on `success`, `expired`, or panel close. The endpoint scopes every query to the authenticated user id, so a trade number from another user resolves as not-found.

**Alternative considered:** server-sent events or websocket push — rejected as disproportionate for a flow that lasts minutes and already has a countdown UI.

### D11: The QR is a per-order image URL carrying amount and memo

The panel renders SePay's QR image endpoint with the order's own parameters — destination account, bank code, `amount` set to the payable Dong frozen at creation, and `des` set to the order's memo. There is no static account QR: every order gets its own image, so a scanning banking app pre-fills both the amount and the description and the user types nothing.

The amount the user enters is free-form (bounded by minimum, per-order maximum, and wallet capacity); the preset options in the general settings only fill the input. This matters for matching: because the amount is embedded in the QR rather than typed by the user, the webhook's amount check is normally an exact equality, and the underpaid/overpaid branches exist for the case where the user ignores the QR and transfers manually.

**Alternative considered:** a single static account QR with the user typing amount and memo — rejected because it moves both failure modes (wrong amount, wrong memo) onto the user and would make unmatched transfers the common case rather than the exception.

### D10: Removal is deletion, not feature-flagging

Retired controllers, settings modules, routes, `model` settlement functions, frontend hooks/components, and the four Go SDKs are deleted outright. Retired provider *constants* in `model/topup.go` are kept only where historical rows must still render a label; the constants that exist solely to drive creation paths are removed with their code.

**Rationale:** dead gateway code that still compiles invites accidental reactivation and keeps four unmaintained SDKs in the dependency graph. The spec's "responds 404" contract is what a deleted route naturally produces.

## Risks / Trade-offs

- **A user omits or mistypes the memo when transferring** → The webhook logs the unmatched transfer with its `referenceCode` and amount; the operator settles it through the existing admin manual-completion endpoint. The payment panel states the memo requirement prominently and makes the memo one-tap copyable.
- **Bank rewrites the memo beyond recognition** → Mitigated by the bank-safe uppercase-alphanumeric format (D1) and by normalizing before extraction (D2). Residual cases fall into the unmatched-transfer path above.
- **Two orders' memos both appear in one transfer content** → Settle nothing, log both candidates (D2). Deliberately manual: guessing here could credit the wrong user.
- **Webhook API key leaks** → Anyone could forge credits. Mitigated by masking the key in the UI, never returning it to non-administrators, and constant-time comparison. Key rotation is a settings edit; document it.
- **Legacy pending orders can never settle automatically** → Accepted and specified. The sweep deliberately does not touch them so they stay visible; operators use manual completion.
- **Expiry window change moves the deadline of live orders** (D4) → Bounded by validating the window against a maximum and by the amount staying frozen. Documented for operators.
- **Single point of failure: one gateway** → If SePay is down, no online top-up is possible. This is the explicit product choice in the proposal; redemption codes and admin manual completion remain as fallbacks.
- **Removing four SDKs is a wide diff touching `go.mod`/`go.sum`** → Verify with a clean `go build ./...` plus the full test suite, and confirm `relaykit` still builds independently with `GOWORK=off go build ./...` per project rules.

## Migration Plan

1. **Pre-deploy:** operator drains — stop advertising legacy gateways, let pending legacy orders settle or expire on the old build. Export the list of still-pending legacy orders for manual follow-up.
2. **Deploy:** ship backend and frontend together. No schema migration runs; `AutoMigrate` sees no column change on `top_ups` or `subscription_orders`.
3. **Configure:** administrator confirms payment compliance (if not already), fills SePay settings, and registers `https://<host>/api/sepay/webhook` with the `Apikey` value in the SePay dashboard.
4. **Verify:** perform one real small-value transfer end to end, confirm the wallet credit and the log entry, then replay the same webhook delivery to confirm idempotency.
5. **Rollback:** redeploy the previous build. Because no schema or data changed, SePay orders created in the meantime remain as rows and are settleable through admin manual completion on the old build.

## Open Questions

- Exact bank-code value to store (SePay accepts several bank identifier forms in its QR URL). Resolvable at configuration time by the operator; it does not affect the specs, the approach, or the task breakdown.
