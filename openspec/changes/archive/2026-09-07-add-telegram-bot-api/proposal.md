## Why

Customers reach this platform only through the web console today. A large part of the Vietnamese customer base lives in Telegram, and they want to check their balance, inspect an API key, read their usage history, and start a top-up without leaving the chat app.

The bot that serves them will be built and deployed as a separate project, on its own release cycle and in its own language. This repository must therefore expose a stable, narrowly scoped integration surface for that bot to call, rather than absorbing a Telegram bot runtime into the gateway binary.

A second, related gap: nothing today records a user's Telegram handle at sign-up, so an operator has no way to reconcile a person in a Telegram support chat with an account in the console.

## What Changes

- **New service-to-service API** under `/api/bot/v1`, authenticated by a single bot service key (`Authorization: Bot <key>`), rate limited per Telegram user rather than per IP. Thirteen endpoints covering health, identity resolution, link-code issuance, account summary, top-up configuration and orders, key inspection, key listing, and usage logs.
- **New account-link flow** that does not depend on a browser widget: the bot requests a short-lived one-time code, and the user redeems it while signed in on the website through a new endpoint `POST /api/user/telegram/link/confirm`. The existing Telegram Login Widget binding stays untouched and remains a parallel path.
- **New `telegram_username` field on users**, captured at registration and required by default through a new administrator option. The value is self-declared and explicitly untrusted until an actual Telegram account is linked, at which point it is overwritten with the verified handle.
- **New outbound event delivery** from the gateway to the bot: `topup.succeeded`, `topup.expired`, and `account.linked`, signed with HMAC so the bot can notify the customer instead of polling.
- **New administrator settings** for the bot service key, the outbound callback URL and its signing secret, the bot API enablement flag, and the registration-username requirement.
- No payment provider is added or changed. Bot-originated top-ups create ordinary SePay orders and settle through the existing SePay webhook, so the "SePay is the only external payment provider" contract is preserved.
- No Telegram Mini App is introduced. The bot renders everything in chat.

## Capabilities

### New Capabilities

- `telegram/bot-api`: The `/api/bot/v1` service-to-service surface — its authentication, per-Telegram-user rate limiting, audit logging, the not-linked error contract, and the behaviour of each of the thirteen endpoints, including the deliberate absence of an unlink endpoint.
- `telegram/account-link`: The self-declared `telegram_username` captured at registration, its normalization and validation rules, the administrator requirement toggle, the one-time link code lifecycle, the website redemption endpoint, and the transition of the handle from claimed to verified.
- `telegram/bot-events`: Outbound signed event delivery to the bot — the event set, payload shape, signature scheme, idempotency identifier, retry policy, and the rule that delivery failure never disturbs a payment or a binding.

### Modified Capabilities

- `payments/sepay-topup`: Top-up orders may now also originate from the bot integration surface. The requirement set must state that such orders are subject to identical amount validation, wallet-ceiling checks, memo generation, expiry, ownership scoping, and webhook settlement as orders created from the web console.
- `public-key-check`: The key report field set becomes a shared contract consumed by both the public unauthenticated endpoint and the bot endpoint, and the rule that a submitted key never appears in a URL or query string must extend to the bot surface.
- `auth/account-binding-safety`: The existing binding safety guarantees must explicitly cover the new link-code path, so a code cannot inherit an account, cannot bind an already-bound Telegram account, and cannot be used to remove the last remaining way into an account.

## Impact

**Backend**

- New: bot API controller, service-key authentication middleware, per-Telegram-user rate limiter, outbound event sender.
- Modified: `controller/user.go` (registration), `controller/option.go` and `model/option.go` and `setting/` (new options), `router/api-router.go` (new route group and one user route), `model/user.go` (new column), `model/auth_flow.go` (new purpose), `model/topup.go` and `service/sepay_expiry_task.go` (event emission points).
- Database: one new column on the users table, added on all three supported engines.

**Frontend**

- Modified: sign-up form and its schema, profile account-bindings tab (link-code entry), Telegram area of the authentication settings screen, `/api/status` consumer for the new flags, English and Vietnamese locale files.

**External**

- A separate repository hosts the bot itself. This change ships the contract it consumes, not the bot.
- Operators must generate a bot service key and, if outbound events are enabled, a callback URL and signing secret.

**Security surface**

- One new high-value credential (the bot service key). Blast radius is deliberately bounded: the bot API cannot unlink an account, cannot complete a link on its own, and cannot mutate quota. Compromise exposes balance reads and the ability to create unpaid pending orders.
