## Context

See `proposal.md` — Why. Requirements live in the four spec deltas under `specs/`.

Constraints that shape the approach, all verified in the current tree:

- **The bot is not built here.** It ships from a separate repository on its own cycle. This repository owns only the contract.
- **SePay is the sole payment provider**, fixed by `payments/legacy-gateway-removal` and `payments/gateway-administration`. Bot top-ups must ride the existing rails rather than introduce a second one.
- **Existing credentials do not fit a bot.** Session tokens (`service.ParseDashboardAccessToken`) belong to one browser session; personal access tokens (`model.ValidateAccessToken`) belong to exactly one user; relay keys (`middleware.TokenAuth`) cannot reach wallet or account data. A bot acting for many users has no usable credential today.
- **Rate limiting is per client IP** (`middleware.CriticalRateLimit`, `SearchRateLimit`). A bot presents one IP for its whole population, so IP-scoped limits would let one abuser deny service to everyone. `middleware.userRedisRateLimiter` already accepts a pre-built key, which is the seam a per-Telegram-user limiter needs.
- **Telegram binding already exists** through the Login Widget (`controller/telegram.go`), backed by `user.telegram_id` and the `external_identity_claims` table with its unique constraint. The new flow must join that model, not shadow it.
- **`model/auth_flow.go`** already provides single-use, expiring, purpose-scoped flows with transactional consumption (`ConsumeAuthFlowWithAction`). It stores only a SHA-256 of its token.
- **Three databases** must stay supported, so schema work is limited to additive column creation.

## Goals / Non-Goals

**Goals:**

- One credential for the bot, with a blast radius small enough that its compromise is an embarrassment rather than an incident.
- Bot-originated top-ups that are byte-for-byte the same kind of order as console top-ups, sharing one code path so the two cannot drift.
- One producer for the key report and one producer for order creation, so a change in either propagates everywhere automatically.
- Abuse isolation between Telegram users despite a single source IP.
- Outbound events that are strictly secondary to the money path.

**Non-Goals:**

- Telegram Mini App, Telegram Stars, or any second payment provider.
- A general-purpose machine-to-machine authorization framework. This is one credential for one known first-party consumer; generalizing it now would be speculative.
- Bot-side behaviour (command parsing, message rendering, message deletion). The contract states obligations; the other repository fulfils them.
- Registering the bot's webhook with Telegram, or any call from this gateway to the Telegram Bot API.

## Decisions

### A single bot service key, with power removed rather than power split

**Chosen:** one administrator-configured key presented as `Authorization: Bot <key>`, compared with `crypto/subtle`, stored write-only in options exactly as `SePayWebhookApiKey` is (saving an empty value keeps the stored key — see `controller/option.go`). Every user-scoped request additionally carries `telegram_user_id`, which the gateway resolves to an account.

**Rejected — service key exchanged for short-lived per-user tokens:** it narrows what a leaked key can read, but the bot must then cache, refresh, and invalidate tokens for every user, and the gateway must issue a token type that has no other consumer. The added machinery buys little here because the same key still mints those tokens on demand.

**Rejected — mTLS or an IP allowlist:** the strongest option, but it fixes the bot's deployment topology to a static address before that topology is known.

The blast radius is instead bounded by **subtraction**, which is what makes the simple credential acceptable:

- There is no unlink endpoint on the bot surface. Unbinding stays on the website.
- The bot can *issue* a link code but cannot *redeem* one. Redemption requires a browser session. A stolen key therefore links nothing.
- No endpoint mutates quota, balance, keys, or user records.
- What remains reachable with a stolen key: reading balances and usage, and creating pending orders that nobody has paid for. Both are audited and rate limited.

### Link codes reuse `auth_flow` with a caller-supplied short token

A person must be able to type the code, so the 32-byte random token `CreateAuthFlow` generates is unusable. Rather than add a second single-use-token table with its own expiry, consumption, and cleanup semantics, `AuthFlowCreate` gains an optional caller-supplied token. The column still stores only `authFlowTokenHash(token)`, so the code is never persisted in clear, the unique index still prevents collisions, and `ConsumeAuthFlowWithAction` still gives transactional redemption.

The code is 8 characters from a case-insensitive alphabet with visually ambiguous characters removed, normalized to upper case before hashing. `Provider` is set to the Telegram provider constant and the Telegram account identifier goes in `Payload`; `UserId` stays empty because the account is unknown until redemption.

Entropy is roughly 2^40 over a five-minute window. That is not enough on its own, so redemption is rate limited per session and per IP, and the three failure modes — unknown, expired, already consumed — are reported identically so codes cannot be probed for existence.

**Rejected — a dedicated `telegram_link_codes` table:** it would duplicate expiry, consumption, idempotency, and cleanup logic that `auth_flow` already implements and already has tests for.

### Rate limiting keyed by Telegram user

A limiter variant builds its Redis key from the Telegram account identifier and delegates to the existing `userRedisRateLimiter`, inheriting the fixed-window implementation and its in-memory fallback. Endpoints that take no Telegram user (health) keep the ordinary global limit.

### Order creation and the key report each get exactly one producer

`controller.SePayRequestTopUp` currently interleaves session lookup, validation, order insertion, and response building. The parts after "which user is this" move into a service function taking a user id and an amount; the existing handler and the bot handler both call it. The same treatment applies to the report body inside `controller.CheckTokenUsage`.

This is the mechanism behind two spec requirements — that a bot order is indistinguishable from a console order, and that the key report field set cannot drift between surfaces. Copying either body would satisfy the tests on the day it was written and silently diverge later.

### Outbound events never touch the settlement transaction

Events are emitted after the credit or state transition has committed, handed to a bounded worker, and delivered with a timeout. Signature scheme follows the one already in `service/webhook.go`: HMAC-SHA256 over the exact response body, hex-encoded, carried in a header alongside the event identifier. Emission points are `model.RechargeSePay` (credited), the sweep in `service/sepay_expiry_task.go` (expired), and the link redemption handler (linked).

A delivery failure is logged and retried with increasing delay up to a bound, then abandoned with an operator-visible entry. It never returns an error into the payment path, because the SePay webhook must keep acknowledging normally regardless of whether some external bot is reachable.

**Rejected — a persisted outbound event queue with durable retry:** correct for an at-least-once guarantee across restarts, but it adds a table, a sweeper, and a backlog to operate. The events here are conveniences; the authoritative state is always readable through the order-status endpoint, which the bot can fall back to. This is a conscious trade-off, and its limit is explicit: an event produced while the process is stopping may never be delivered, and the customer then learns of their credit by asking rather than by being told.

### `telegram_username` is a display field, never a key

Added as `varchar(32)`, nullable-by-default, no unique index, created through `AutoMigrate` so all three engines take it as an additive column. It carries a self-declared value until a link completes, after which it is overwritten with the handle Telegram reports, or cleared if Telegram reports none.

No index is added because nothing may look accounts up by it — a Telegram handle can be changed, released, or sold, so treating it as an identifier is precisely the account-takeover vector that `auth/account-binding-safety` exists to close. `telegram_id` remains the only identity key.

The registration requirement is an option defaulting to true. Following the project rule about boolean defaults across engines, the default lives in option initialization rather than in a GORM `default:` tag, which MySQL and PostgreSQL normalize differently and which makes `AutoMigrate` re-issue `ALTER TABLE` on every restart.

### The bot route group is absent, not merely rejecting, when unconfigured

The group's guard returns 404 rather than 401 or 403 while the integration is disabled or unkeyed, so a deployment that never enables the feature presents no discoverable surface. This matches how the retired payment gateways were removed.

## Risks / Trade-offs

- **The bot service key can read any linked user's balance and usage.** → Bounded by subtraction: no unlink, no redemption, no mutation. Every user-scoped call is audited with the Telegram identifier and outcome, the key is write-only in settings and rotatable, and the surface disappears at 404 when disabled.
- **An 8-character link code is brute-forcible in isolation.** → Five-minute TTL, single use, per-session and per-IP limits on redemption, identical responses for unknown, expired, and consumed. The reward for a successful guess is also small: it binds the attacker's Telegram account to their own account, granting no access to anyone else's.
- **Someone declares another person's Telegram handle at registration.** → The handle is inert. It grants nothing, matches nothing, and blocks nothing; the real holder can still link their own account. This is why the field is unindexed and non-unique rather than treated as a claim.
- **API keys pasted into a Telegram chat persist in Telegram's history.** → The contract obliges the bot to delete those messages, and the key-listing endpoint exists so a customer can inspect their keys without any key value crossing a chat. The gateway cannot enforce the deletion; that limit is stated in the spec rather than hidden.
- **Extracting shared logic touches the live top-up path.** → The extraction is a move, not a rewrite: the existing SePay order tests must pass unchanged, before any bot handler is written.
- **Best-effort events can be lost on shutdown.** → Accepted, stated above. The order-status endpoint remains authoritative and the bot can poll on demand.
- **A new column ships to three engines.** → Additive `ADD COLUMN` only, exercised on a fresh database and on an upgraded one, with startup run twice to prove idempotency, per the repository's database rules.

## Migration Plan

1. Ship the column, the options, and the shared-logic extraction. Nothing is reachable yet: the bot integration defaults to disabled, and the registration requirement defaults to on but has no field to enforce until the frontend ships in the same release.
2. Verify migration on SQLite, MySQL, and PostgreSQL, on a fresh database and on one created by the previous release, running startup twice.
3. An operator generates a bot service key and enables the integration. Optionally sets a callback address and signing secret for events.
4. The separate bot repository points at the deployment and begins calling the surface.

Rollback: disable the bot integration option, which returns the whole surface to 404 without touching data. Turning off the registration requirement makes the handle optional again. The column is additive and can stay; existing rows carry an empty handle and every path already treats an empty handle as "not declared".

## Open Questions

- Retry bound and backoff schedule for outbound events. Any bounded schedule satisfies the spec; the exact numbers can be tuned from production behaviour without changing the contract, the approach, or the task breakdown.
