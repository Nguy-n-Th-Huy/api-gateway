## Why

The gateway currently ships five payment integrations (Epay, Stripe, Creem, Waffo, Waffo Pancake) that none of the operators of this deployment use. They are card/global-checkout oriented, carry four third-party SDK dependencies, and produce a payment settings UI with five tabs of dead configuration. This deployment sells to Vietnamese users, who pay by domestic bank transfer, so the only gateway that matters is SePay (VietQR bank transfer with balance-change webhooks).

## What Changes

- **BREAKING** Remove every existing top-up gateway: Epay, Stripe, Creem, Waffo, Waffo Pancake. Their checkout endpoints, webhook/notify endpoints, settings, admin configuration UI, and wallet UI are deleted, not merely hidden.
- **BREAKING** Remove the same gateways from the subscription purchase flow. Subscriptions keep the existing wallet-balance payment path and gain SePay as the only external gateway.
- Add SePay as the single external payment provider for both wallet top-up and subscription purchase, using its bank-transfer model: the backend creates a pending order with a unique transfer memo, the frontend renders a VietQR payment image plus copyable bank details, and SePay's webhook credits the order when the matching transfer lands.
- Add SePay administration to system settings: enabled toggle, bank account number, bank code, account holder, webhook API key, VND-per-USD price (default 1000 — one thousand Dong per USD of balance), minimum top-up, and order expiry window. The five removed gateway tabs collapse into one SePay tab plus the shared general settings.
- Preserve historical data: existing `top_ups` and `subscription_orders` rows with removed provider values stay readable in user and admin history; only the creation and settlement paths for those providers are removed.
- Pending orders belonging to removed providers can no longer be settled by webhook. The existing admin manual-completion endpoint stays the operator's escape hatch for them.
- Drop the four Go SDK dependencies: `github.com/Calcium-Ion/go-epay`, `github.com/stripe/stripe-go/v81`, `github.com/waffo-com/waffo-go`, `github.com/waffo-com/waffo-pancake-sdk-go`.

## Capabilities

### New Capabilities

- `payments/sepay-topup`: Wallet top-up through SePay — order creation with a unique transfer memo, VietQR rendering data, USD↔VND conversion, order expiry, webhook verification, idempotent crediting, and the user-facing payment status flow.
- `payments/sepay-subscription`: Subscription plan purchase through SePay, reusing the same order/memo/webhook mechanics while keeping the existing wallet-balance purchase path intact.
- `payments/gateway-administration`: Admin configuration surface for payments after the consolidation — SePay credentials and limits, shared general settings (price, minimum top-up, amount options, discounts), payment compliance confirmation, and the guarantee that removed gateways are absent from both configuration and the public top-up info payload.
- `payments/legacy-gateway-removal`: The removal contract itself — which endpoints stop existing, what the API returns for them, how historical records of removed providers remain visible, and how pending legacy orders are handled.

### Modified Capabilities

<!-- None. openspec/specs/ contains no published capability specs yet, so all payment behavior is captured as new capabilities above. -->

## Impact

**Backend — removed**

- `controller/topup.go`: `RequestEpay`, `EpayNotify`, `GetEpayClient` and Epay branches of `GetTopUpInfo`
- `controller/topup_stripe.go`, `controller/topup_creem.go`, `controller/topup_waffo.go`, `controller/topup_waffo_pancake.go`, `controller/topup_waffo_pancake_test.go`
- `controller/subscription_payment_epay.go`, `controller/subscription_payment_stripe.go`, `controller/subscription_payment_creem.go`, `controller/subscription_payment_waffo_pancake.go`
- `controller/payment_webhook_availability.go` entries for removed gateways
- `service/epay.go`, `service/waffo_pancake.go`
- `setting/payment_stripe.go`, `setting/payment_creem.go`, `setting/payment_waffo.go`, `setting/payment_waffo_pancake.go`
- `model/topup.go`: `RechargeEpay`, `Recharge` (Stripe), `RechargeCreem`, `RechargeWaffo`, `RechargeWaffoPancake` and the corresponding `PaymentMethod*`/`PaymentProvider*` constants for creation paths
- `setting/operation_setting/payment_setting_old.go`: Epay `PayMethods` / `PayAddress` / `EpayId` / `EpayKey` configuration
- `router/api-router.go`: `/api/stripe/webhook`, `/api/creem/webhook`, `/api/waffo/webhook`, `/api/waffo-pancake/webhook/:env`, `/api/user/epay/notify` (GET+POST), `/api/user/pay`, `/api/user/stripe/*`, `/api/user/creem/pay`, `/api/user/waffo/*`, `/api/user/waffo-pancake/*`, `/api/subscription/{epay,stripe,creem,waffo-pancake}/pay`, `/api/subscription/epay/{notify,return}`, `/api/option/waffo-pancake/*`

**Backend — added**

- SePay controller (top-up order creation, subscription order creation, webhook receiver, order status query), SePay settings module, SePay order-expiry sweep in the existing system task runner, and `model` settlement functions mirroring the existing row-lock + idempotency pattern used by `RechargeEpay`
- New routes under `/api/user/sepay/*`, `/api/subscription/sepay/pay`, and the anonymous webhook `/api/sepay/webhook`

**Frontend — removed**

- `web/src/features/wallet/`: `hooks/use-creem-payment.ts`, `hooks/use-waffo-pancake-payment.ts`, `components/creem-products-section.tsx`, `components/dialogs/creem-confirm-dialog.tsx`, and the multi-gateway branches in `hooks/use-payment.ts`, `lib/payment.ts`, `lib/ui.tsx`, `constants.ts`, `types.ts`, `api.ts`
- `web/src/features/system-settings/integrations/`: `waffo-pancake-settings-section.tsx`, `waffo-pancake-api.ts`, `creem-products-visual-editor.tsx`, `creem-product-dialog.tsx`, `payment-methods-visual-editor.tsx`, `payment-method-dialog.tsx`, plus the Epay/Stripe/Creem/Waffo tabs in `payment-settings-section.tsx` and `billing/section-registry.tsx`
- `web/src/assets/brand-icons/icon-stripe.tsx`
- `web/src/features/subscriptions/`: gateway selection branches in `components/dialogs/subscription-purchase-dialog.tsx`, `api.ts`, `types.ts`
- Removed gateway keys across `web/src/i18n/locales/*.json` and `web/src/i18n/static-keys.ts`

**Frontend — added**

- SePay top-up flow (amount entry → order creation → VietQR image, bank details, copy-to-clipboard memo, countdown to expiry, automatic status polling until credited or expired)
- SePay subscription purchase flow reusing the same payment panel
- Single SePay tab in payment system settings

**Dependencies**

- `go.mod` / `go.sum`: remove `go-epay`, `stripe-go/v81`, `waffo-go`, `waffo-pancake-sdk-go`. SePay needs no SDK — VietQR image URL plus an authenticated inbound webhook.

**Data**

- No destructive migration. `top_ups` and `subscription_orders` keep their `payment_method` / `payment_provider` columns and all historical values.

**Documentation**

- Deployment/configuration docs and `.env` examples that mention the removed gateways must be updated to describe SePay configuration instead.
