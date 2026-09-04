## 1. SePay settings and configuration

- [ ] 1.1 Add `setting/payment_sepay.go` holding enabled flag, bank account number, bank code, account holder name, webhook API key, minimum top-up, and expiry window in minutes, wired into the existing option loading/persistence path
- [ ] 1.2 Add a completeness check (`IsSePayConfigured`) that requires enabled plus every credential field non-empty, and validation rejecting an out-of-range expiry window
- [ ] 1.3 Set the general price setting's default to 1000 Dong per USD and relabel the field as Dong per USD in the settings UI
- [ ] 1.4 Ensure the webhook API key is never emitted to non-admin responses, and that saving an empty key leaves the stored key unchanged ← (verify: no endpoint reachable by a non-admin returns the key; empty-key save preserves the previous value)

## 2. SePay order model and settlement

- [ ] 2.1 Add the bank-safe trade-number/memo generator (uppercase `A-Z0-9`, `SP` prefix for top-up, `SS` for subscription) with retry on unique-constraint conflict
- [ ] 2.2 Add `model` top-up settlement `RechargeSePay(tradeNo, transferAmountVND, callerIp)` following the existing `RechargeEpay` shape: transaction, `lockForUpdate` by `trade_no`, provider check, status check, amount-sufficiency check, save, `creditTopUpQuota`, top-up log
- [ ] 2.3 Add subscription settlement for SePay reusing `CompleteSubscriptionOrder` with `expectedPaymentProvider` set to SePay
- [ ] 2.4 Add expiry helpers that derive the deadline from `create_time` plus the configured window, and mark SePay-only pending orders `expired` in bulk
- [ ] 2.5 Add `PaymentMethodSePay` / `PaymentProviderSePay` constants and keep the retired provider constants only where historical rows must still render a label ← (verify: settlement is idempotent under duplicate and concurrent calls, wallet ceiling is enforced inside the same transaction, and no bare int cast is used in quota conversion)

## 3. SePay order creation endpoints

- [ ] 3.1 Add `POST /api/user/sepay/pay`: compliance gate, configuration gate, minimum-amount check, `ValidateTopUpQuotaCapacity`, decimal VND conversion via `common/quota_math.go` helpers, pending order insert, and a response carrying memo, payable VND, bank details, VietQR image URL, trade number, expiry timestamp
- [ ] 3.2 Add `POST /api/subscription/sepay/pay` creating a pending subscription order with the same response shape
- [ ] 3.3 Add `GET /api/user/sepay/order/:trade_no` scoped to the authenticated user, returning status, payable amount, memo, and expiry
- [ ] 3.4 Apply `middleware.CriticalRateLimit()` to both creation endpoints, matching the removed gateways ← (verify: an amount below minimum, above the per-order bound, or exceeding wallet capacity creates no order; a trade number owned by another user returns not-found)

## 4. SePay webhook

- [ ] 4.1 Add `POST /api/sepay/webhook` (anonymous, body-limited) authenticating `Authorization: Apikey <key>` with `crypto/subtle.ConstantTimeCompare`, rejecting with 401 when the key is missing, wrong, or the integration is disabled
- [ ] 4.2 Implement memo extraction: normalize `content` to uppercase `A-Z0-9`, regex-extract candidate memos, resolve each by exact `trade_no` lookup, and route by prefix to the top-up or subscription table
- [ ] 4.3 Implement the outcome branches — credited, already credited, underpaid, overpaid, outgoing transfer, unmatched, expired, ambiguous — each returning HTTP 200 with a warning or error log carrying the SePay `id` and `referenceCode`
- [ ] 4.4 Register the SePay expiry sweep with the existing system-task runner, scoped to SePay orders only ← (verify: replaying an identical delivery credits exactly once; a transfer for an order past its derived deadline credits nothing even before the sweep runs; a content matching two pending memos settles nothing)

## 5. Remove retired backend gateways

- [ ] 5.1 Delete `controller/topup_stripe.go`, `controller/topup_creem.go`, `controller/topup_waffo.go`, `controller/topup_waffo_pancake.go`, `controller/topup_waffo_pancake_test.go`
- [ ] 5.2 Delete `controller/subscription_payment_epay.go`, `controller/subscription_payment_stripe.go`, `controller/subscription_payment_creem.go`, `controller/subscription_payment_waffo_pancake.go`
- [ ] 5.3 Delete `service/epay.go`, `service/waffo_pancake.go`, `setting/payment_stripe.go`, `setting/payment_creem.go`, `setting/payment_waffo.go`, `setting/payment_waffo_pancake.go`
- [ ] 5.4 Remove `RequestEpay`, `EpayNotify`, `GetEpayClient` from `controller/topup.go` and rewrite `GetTopUpInfo` to report only SePay availability plus the shared pricing settings
- [ ] 5.5 Remove `RechargeEpay`, `Recharge`, `RechargeCreem`, `RechargeWaffo`, `RechargeWaffoPancake` from `model/topup.go`, keeping `ManualCompleteTopUp` working for legacy pending orders
- [ ] 5.6 Remove Epay `PayMethods`/`PayAddress`/`EpayId`/`EpayKey` from `setting/operation_setting/payment_setting_old.go` and the payment-method list from the general settings contract
- [ ] 5.7 Delete every retired route from `router/api-router.go` and update `controller/payment_webhook_availability.go` to report only SePay
- [ ] 5.8 Remove `go-epay`, `stripe-go/v81`, `waffo-go`, `waffo-pancake-sdk-go` from `go.mod`, run `go mod tidy`, and confirm `go build ./...` plus `cd relaykit && GOWORK=off go build ./...` ← (verify: no retired identifier remains via repo-wide grep; retired routes return 404; `GetTopUpInfo` payload contains no retired flag, product list, or payment-method entry)

## 6. Frontend SePay flow

- [ ] 6.1 Add SePay API bindings in `web/src/features/wallet/api.ts` and types for the order-creation and order-status payloads
- [ ] 6.2 Add the SePay payment panel component: per-order VietQR image (amount and memo encoded), payable VND amount, bank account number, bank code, account holder, copyable memo, copyable account number, memo-required instruction, and expiry countdown
- [ ] 6.3 Make the amount input free-form within the configured bounds, with preset options acting as editable shortcuts and a live preview of the payable Dong amount
- [ ] 6.4 Add the order-status polling hook that stops on success, expiry, or unmount, and refreshes the displayed wallet balance on success
- [ ] 6.5 Rewrite `hooks/use-payment.ts`, `lib/payment.ts`, `lib/ui.tsx`, `constants.ts`, `types.ts` so a single provider path remains
- [ ] 6.6 Wire the SePay path into the subscription purchase dialog alongside the unchanged wallet-balance option ← (verify: the panel renders every required field, polling stops on both terminal states, and the balance-payment path is untouched)

## 7. Remove retired frontend surfaces

- [ ] 7.1 Delete `hooks/use-creem-payment.ts`, `hooks/use-waffo-pancake-payment.ts`, `components/creem-products-section.tsx`, `components/dialogs/creem-confirm-dialog.tsx` and their exports in `hooks/index.ts`
- [ ] 7.2 Delete `integrations/waffo-pancake-settings-section.tsx`, `integrations/waffo-pancake-api.ts`, `integrations/creem-products-visual-editor.tsx`, `integrations/creem-product-dialog.tsx`, `integrations/payment-methods-visual-editor.tsx`, `integrations/payment-method-dialog.tsx`
- [ ] 7.3 Rewrite `integrations/payment-settings-section.tsx` and `billing/section-registry.tsx` to a single SePay tab plus general settings, with the API key masked once stored
- [ ] 7.4 Delete `web/src/assets/brand-icons/icon-stripe.tsx` and remove retired gateway branches from the subscriptions feature
- [ ] 7.5 Remove retired i18n keys from `web/src/i18n/locales/*.json` and `web/src/i18n/static-keys.ts`, add SePay keys for every new string in all seven locales, and run `bun run i18n:sync` ← (verify: no retired gateway string, icon, or settings panel is reachable in the UI; no untranslated raw key renders; legacy provider labels in history still render readably)

## 8. Tests

- [ ] 8.1 Backend tests for memo generation and extraction: bank-safe format, normalization of noisy bank content, single-candidate resolution, ambiguous-content rejection
- [ ] 8.2 Backend tests for webhook authentication: valid key, missing key, wrong key, malformed scheme, disabled integration
- [ ] 8.3 Backend settlement tests: successful credit, duplicate delivery, underpayment, overpayment, outgoing transfer, unmatched memo, expired order, wallet-ceiling rejection
- [ ] 8.4 Backend validation tests for order creation bounds — below minimum, above per-order maximum, wallet capacity exceeded, zero-Dong conversion, non-preset arbitrary amount accepted, non-integer amount rejected — asserting no order is created and no saturated value is stored when rejected
- [ ] 8.5 Backend test that legacy top-up and subscription rows with retired provider values remain listable and searchable, and that `ManualCompleteTopUp` still credits them exactly once
- [ ] 8.6 Frontend tests for the payment panel: required fields rendered, copy actions, countdown reaching expiry, polling stop on success and on expiry ← (verify: full `go test ./...` passes, `cd web && bun run test`, `bun run typecheck`, and lint on every touched file pass)

## 9. Database compatibility verification

- [ ] 9.1 Run startup and `AutoMigrate` twice against fresh SQLite, MySQL, and PostgreSQL instances, confirming no schema change is emitted for `top_ups` and `subscription_orders`
- [ ] 9.2 Upgrade a representative database created by the latest released version and confirm legacy top-up and subscription rows keep their provider values, indexes, and uniqueness guarantees
- [ ] 9.3 Exercise the full SePay create → webhook → credit flow on each of the three engines, including a concurrent duplicate webhook, and record exact engine versions, commands, and results in the handoff ← (verify: all three engines produce identical settlement outcomes and exactly one credit under concurrency)

## 10. Documentation

- [ ] 10.1 Update deployment and configuration docs plus `.env` examples to describe SePay configuration and remove the retired gateways
- [ ] 10.2 Document the operator runbook: registering the webhook URL and API key with SePay, rotating the key, handling unmatched or underpaid transfers, and settling legacy pending orders through admin manual completion
