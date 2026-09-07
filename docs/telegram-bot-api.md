# Telegram Bot Integration — Gateway Contract

This is the gateway-side reference for the service-to-service surface at `/api/bot/v1` that an externally hosted Telegram bot calls on behalf of a Telegram user, and for the events the gateway pushes back to that bot. The bot itself ships from a separate repository (`api-gateway-telegram-bot`) on its own release cycle; this repository owns only the contract.

The authoritative behavior specs live in `openspec/changes/2026-09-07-add-telegram-bot-api/specs/telegram/{bot-api,account-link,bot-events}/spec.md` and its `design.md`. This document restates that contract concretely — exact routes, request/response shapes, and error codes — so a client can be implemented without reading gateway source. If this document and the specs ever disagree, the specs win.

Implementation: `router/bot-api-router.go`, `controller/telegram_bot.go`, `controller/telegram_link.go`, `middleware/telegram_bot_auth.go`, `service/telegram_bot_events.go`, `service/telegram_bot_identity.go`, `model/telegram_link.go`, `model/telegram_handle.go`.

## Authentication

Every `/api/bot/v1` request is authenticated with a single administrator-configured bot service key:

```
Authorization: Bot <service key>
```

- Comparison is constant-time (`crypto/subtle.ConstantTimeCompare`).
- A missing `Authorization` header, a header without the `Bot ` scheme, or a non-matching key all return **HTTP 401** with an identical body — none of these cases can be told apart by the caller.
- A user session token or a personal access token presented here is also rejected with 401; only the bot service key is accepted on this surface.
- While the integration is disabled, or enabled with no service key configured, **every** `/api/bot/v1` request returns **HTTP 404** instead of 401 — an unconfigured deployment exposes no discoverable surface at all.

## Response envelope

Every endpoint below responds with one of these shapes:

- Success: `{"success": true, "data": <endpoint-specific payload>}`, HTTP 200.
- Business-logic refusal (not-linked, already-linked, key/order not found, top-up unavailable, invalid request): `{"success": false, "error_code": "<CODE>", "message": "<human-readable text>"}`, **HTTP 200** — these are ordinary JSON bodies, not transport errors, so the bot must branch on `error_code`, not on HTTP status, to tell them apart.
- Transport-level failure: HTTP 401 (bad credential), 404 (integration unconfigured), 429 (rate limited), or 500 (`error_code: "TELEGRAM_BOT_INTERNAL_ERROR"`, HTTP 500).

Stable `error_code` values:

| Code | Meaning |
|---|---|
| `TELEGRAM_BOT_INVALID_REQUEST` | Missing/malformed required field (e.g. `telegram_user_id`), HTTP 400 |
| `TELEGRAM_BOT_NOT_LINKED` | The supplied `telegram_user_id` is bound to no account. Distinct from every other outcome so the bot can route the person into the linking flow. |
| `TELEGRAM_BOT_ACCOUNT_UNUSABLE` | The bound account is disabled or deleted. Distinct from not-linked. |
| `TELEGRAM_BOT_ALREADY_LINKED` | Link-code issuance requested for a `telegram_user_id` already bound to an account. |
| `TELEGRAM_BOT_KEY_NOT_FOUND` | The submitted API key matches no token. |
| `TELEGRAM_BOT_ORDER_NOT_FOUND` | The trade number does not exist, or exists but is owned by a different account (the two cases are indistinguishable by design). |
| `TELEGRAM_BOT_TOPUP_UNAVAILABLE` | Top-up is currently unavailable (SePay not configured, or payment compliance not confirmed). |
| `TELEGRAM_BOT_INTERNAL_ERROR` | Unexpected server error. |

## Rate limiting

User-scoped endpoints are rate-limited **per `telegram_user_id`**, not per caller IP, because the bot presents one IP for its entire population. Exceeding the limit for one Telegram user returns **HTTP 429** for that identifier only; other Telegram users are unaffected. `GET /health` takes no Telegram user and uses the ordinary global limit instead.

## Auditing

Every account-scoped request is recorded with the Telegram user identifier, the resolved account (when one exists), the endpoint, and the outcome. No API key, key fragment, service key, or session token ever reaches an audit entry or log line — a key-inspection audit entry records only that an inspection occurred and its outcome.

## API keys are body-only, never in a URL or query string

On every endpoint that accepts an API key (key inspection, key-scoped usage log), the key **must** be supplied in the JSON request body. A key supplied only as a path segment or query parameter is rejected as missing — the request never uses it for lookup. This keeps key material out of access logs, proxy logs, and referrer headers.

## Withheld powers

- **No unlink endpoint exists on this surface.** Unbinding a Telegram account stays on the website, behind an authenticated session.
- **No redemption endpoint exists on this surface.** The bot can *issue* a link code (below) but can never *redeem* one — redemption is `POST /api/user/telegram/link/confirm`, reachable only to a signed-in browser session. A stolen bot service key therefore links nothing.
- **No endpoint mutates quota, balance, keys, or user records.** Everything reachable here is a read, or the creation of a pending, unpaid SePay top-up order.

## The thirteen `/api/bot/v1` endpoints

Pagination query parameters (`p`, `page_size`, default page size from `common.ItemsPerPage`, capped at 100) apply to every paginated endpoint below, including the two that accept their other parameters in the JSON body. A paginated `data` payload is always `{"page": n, "page_size": n, "total": n, "items": [...]}`.

### 1. Health report — no Telegram user required

`GET /api/bot/v1/health`

No request body. Use this at startup and periodically to decide whether to show top-up commands.

```json
// data
{
  "version": "v0.0.0",
  "topup_available": true,
  "topup_min_amount": 1,
  "topup_max_amount": 1000,
  "telegram_handle_required": true
}
```

### 2. Identity resolution

`POST /api/bot/v1/identity/resolve`

The Telegram identifier is read only from the JSON body — never a query parameter — for the same reason keys are body-only.

Request:

```json
{ "telegram_user_id": "123456789" }
```

Response when bound (never includes balance, key, order, or log data):

```json
{ "linked": true, "account_id": 42, "username": "alice", "status": 1 }
```

Response when unbound — this is a **normal success response**, not `TELEGRAM_BOT_NOT_LINKED`, because reporting bound-vs-unbound is this endpoint's entire purpose:

```json
{ "linked": false }
```

`status` follows the console's user status codes: `1` enabled, `2` disabled.

### 3. Link-code issuance

`POST /api/bot/v1/identity/link/start`

Request:

```json
{ "telegram_user_id": "123456789", "telegram_username": "alice_tg" }
```

`telegram_username` is optional and is the bot's **self-reported, unverified** handle for that Telegram account — see "Handle provenance" below. Success response:

```json
{ "code": "AB3D9F2K", "expires_at": 1700000300, "redemption_address": "/api/user/telegram/link/confirm" }
```

`expires_at` is a Unix second timestamp; the code is single-use and expires roughly five minutes after issuance. If `telegram_user_id` is already bound to an account, the response is `error_code: "TELEGRAM_BOT_ALREADY_LINKED"` and no code is issued.

Redemption is **not reachable from `/api/bot/v1` at all**. It happens only at `POST /api/user/telegram/link/confirm` (body: `{"code": "AB3D9F2K"}`), which requires an authenticated website session and returns identical refusals for an unknown, expired, or already-consumed code so codes cannot be probed for existence.

### 4. Account summary

`GET /api/bot/v1/account?telegram_user_id=123456789`

```json
// data
{ "quota": 500000, "used_quota": 120000, "group": "default", "status": 1, "telegram_username": "alice_tg" }
```

`quota`/`used_quota` are in the same integer unit the console uses. Errors: `TELEGRAM_BOT_NOT_LINKED`, `TELEGRAM_BOT_ACCOUNT_UNUSABLE`.

### 5. Top-up configuration

`GET /api/bot/v1/topup/config?telegram_user_id=123456789` (the identifier is optional here and only scopes the rate limit)

```json
// data
{
  "available": true,
  "min_amount": 1,
  "max_amount": 1000,
  "presets": [{ "amount": 10, "discount": 0.05 }, { "amount": 50 }],
  "price": 26000,
  "order_lifetime_minutes": 30
}
```

`price` is Dong per USD; `discount` is omitted for a preset with none.

### 6. Top-up order creation

`POST /api/bot/v1/topup/orders`

Request:

```json
{ "telegram_user_id": "123456789", "amount": 10 }
```

`amount` is in USD. This is an ordinary SePay order — identical validation, wallet-ceiling check, memo generation, expiry, and webhook settlement as a console-created order, and it settles through the same `POST /api/sepay/webhook`. Success response:

```json
{
  "trade_no": "SP1700000000ABCDEF",
  "memo": "SP1700000000ABCDEF",
  "payable_vnd": 260000,
  "bank_account": "1234567890",
  "bank_code": "VCB",
  "account_holder": "Example Company",
  "vietqr_url": "https://img.vietqr.io/image/...",
  "create_time": 1700000000,
  "expire_time": 1700001800,
  "status": "pending",
  "money": 10
}
```

Errors: `TELEGRAM_BOT_NOT_LINKED`, `TELEGRAM_BOT_ACCOUNT_UNUSABLE`, `TELEGRAM_BOT_TOPUP_UNAVAILABLE` (top-up currently unavailable), or `TELEGRAM_BOT_INVALID_REQUEST` carrying the validation message (below minimum, above the per-order maximum, or would exceed the wallet ceiling) — in every rejection case, no order is created.

### 7. Order lookup

`GET /api/bot/v1/topup/orders/:trade_no?telegram_user_id=123456789`

Same response shape as order creation's `data`. A trade number owned by a different account and one that does not exist both return `error_code: "TELEGRAM_BOT_ORDER_NOT_FOUND"` — the two cases are indistinguishable so existence is never disclosed to a non-owner.

### 8. Top-up history

`GET /api/bot/v1/topup/orders?telegram_user_id=123456789&p=1&page_size=20`

Paginated `items` are the same per-order shape as endpoint 6/7's `data`, restricted to the resolved account's own SePay orders.

### 9. Key inspection

`POST /api/bot/v1/keys/check`

Request:

```json
{ "telegram_user_id": "123456789", "key": "sk-..." }
```

The key is normalized the same way the public key-check endpoint (`docs`/`specs/public-key-check`) normalizes it. Success response reuses that endpoint's report verbatim, plus an ownership flag:

```json
{
  "report": {
    "name": "my-key",
    "group": "default",
    "status": 1,
    "unlimited_quota": false,
    "total_granted": 100000,
    "total_used": 25000,
    "total_available": 75000,
    "expires_at": -1,
    "created_time": 1700000000,
    "accessed_time": 1700003600,
    "model_limits_enabled": false,
    "model_limits": {},
    "available_models": ["gpt-4o", "claude-3-5-sonnet"]
  },
  "is_owner": true
}
```

Because `report` is produced by the same shared function the public key-check page uses, any field the public report gains or loses appears here identically and automatically. An unknown key returns `error_code: "TELEGRAM_BOT_KEY_NOT_FOUND"` with no `report`.

**Consumer obligation:** see "Delete chat messages carrying a key" below before wiring this endpoint into a chat flow.

### 10. Key listing

`GET /api/bot/v1/keys?telegram_user_id=123456789&p=1&page_size=20`

```json
// data.items[]
{
  "id": 7,
  "name": "my-key",
  "group": "default",
  "status": 1,
  "remain_quota": 75000,
  "used_quota": 25000,
  "unlimited_quota": false,
  "expired_time": -1
}
```

No entry ever carries a key value or any fragment of one. This is the endpoint to steer a person toward instead of asking them to paste a key into the chat.

### 11. Usage log listing

`GET /api/bot/v1/logs?telegram_user_id=123456789&type=2&start_timestamp=0&end_timestamp=0&model_name=&key_name=&p=1&page_size=20`

All query parameters except `telegram_user_id` are optional filters. `type` follows the log type codes: `1` top-up, `2` consume, `3` manage, `4` system, `5` error, `6` refund, `7` login. `items` are the console's ordinary log-entry shape (`id`, `created_at`, `type`, `content`, `model_name`, `token_name`, `quota`, `prompt_tokens`, `completion_tokens`, `use_time`, `is_stream`, `channel_name`, `group`, `ip`, ...), restricted to the resolved account's own entries, most recent first.

### 12. Usage statistics

`GET /api/bot/v1/logs/stat?telegram_user_id=123456789&start_timestamp=0&end_timestamp=0`

```json
// data
{ "quota": 250000, "request_count": 340, "prompt_tokens": 120000, "completion_tokens": 45000 }
```

A `start_timestamp`/`end_timestamp` of `0` is unbounded on that side. Figures match what the console reports for the same account and range.

### 13. Key-scoped usage log

`POST /api/bot/v1/keys/logs`

Request:

```json
{ "telegram_user_id": "123456789", "key": "sk-..." }
```

Pagination (`p`, `page_size`) is still supplied as query parameters even though this is a POST. Response `items` are the same log-entry shape as endpoint 11, restricted to the entries recorded against that key. Unknown key: `error_code: "TELEGRAM_BOT_KEY_NOT_FOUND"`.

**Consumer obligation:** see below.

## Delete chat messages carrying a key

The key-inspection (9) and key-scoped-log (13) endpoints exist because a person may need to check a key they already have. **A consumer that accepts a key through a Telegram message MUST delete that chat message immediately after handling it** — a key pasted into a chat otherwise persists indefinitely in Telegram's own history and on Telegram's servers, entirely outside the gateway's control. The gateway cannot enforce this; it is a hard requirement on the bot's implementation.

Prefer the key-listing endpoint (10) whenever the goal is only "let a person see their keys" — it never requires a key to cross the chat at all, so there is nothing to delete.

## Handle provenance — display-only, never verified with Telegram

`telegram_username`, wherever it appears in this contract (endpoint 4's account summary, the optional field on endpoint 3's link-code issuance), is a **display value the bot itself reports**, at the moment it issues a link code — not a value the gateway independently verifies with the Telegram Bot API. **The gateway never calls the Telegram Bot API.** At redemption, the gateway writes exactly what the bot supplied at issuance time into the account's stored handle, unmodified.

This is bounded by construction, per `specs/telegram/account-link/spec.md` ("A stored handle is untrusted until the account is linked" / "`telegram_username` is a display field, never a key" in `design.md`):

- The handle is **display-only**. It is never a lookup key for authentication, binding, or authorization.
- The column is **not indexed** and **not unique** — no query anywhere resolves an account by handle.
- The only identity key is the verified Telegram account identifier (`telegram_user_id` / `telegram_id`), established through the existing Telegram Login Widget binding or link-code redemption — never the handle.

A bot that reports a wrong or stale handle can only make a display label wrong; it cannot bind, redirect, or expose anything belonging to a different account.

## Outbound events

The gateway pushes signed events to a bot-configured callback address. **Delivery is opt-in and best-effort**: it happens only once an administrator has set a callback URL, a signing secret, and enabled the integration. While any of those is missing, no events are sent and the gateway continues operating normally — this is not an error condition, it simply means the bot must poll the query endpoints above instead. A slow, failing, unreachable, or hostile callback address never delays, rolls back, duplicates, or otherwise alters a wallet credit, an order state transition, or a binding; an event that is never delivered (e.g. the process was shutting down) is an accepted, permanent loss, and the corresponding query endpoint remains the authoritative source of truth. No event is delivered for an operation on an account with no Telegram binding.

### Delivery

```
POST <configured callback URL>
Content-Type: application/json
X-Telegram-Bot-Signature: <hex-encoded HMAC-SHA256>
```

```json
{
  "event_id": "<stable identifier>",
  "type": "topup.succeeded",
  "timestamp": 1700000000,
  "data": { "...": "event-specific fields, see below" }
}
```

### Signature

`X-Telegram-Bot-Signature` is `hex(HMAC-SHA256(callback secret, exact request body bytes))` — the same scheme the existing SePay webhook signer uses. The bot must recompute the signature over the exact bytes received (before any re-serialization) and compare it to the presented one before acting; a byte-for-byte mismatch means either tampering or a forgery from a third party, and the delivery must be rejected. The secret itself never appears in the payload.

### Idempotency

`event_id` is stable across every retry of one delivery and distinct between different events. A bot that has already acted on a given `event_id` must not notify the customer a second time.

### Retry and delivery bound

Failed deliveries are retried with increasing delay up to a bounded number of attempts (4, with a 5-second per-attempt timeout and doubling backoff starting at 1 second, as currently configured — treat the exact numbers as tunable, not contractual). An operator-visible log entry is recorded if an event is ultimately abandoned after exhausting the bound.

### `topup.succeeded`

Delivered after a top-up order belonging to a linked account has committed its wallet credit.

```json
{ "telegram_user_id": "123456789", "trade_no": "SP1700000000ABCDEF", "credited_amount": 10, "balance": 510000 }
```

### `topup.expired`

Delivered from the expiry sweep when a pending top-up order belonging to a linked account is marked expired.

```json
{ "telegram_user_id": "123456789", "trade_no": "SP1700000000ABCDEF" }
```

### `account.linked`

Delivered after a link-code redemption commits a new binding.

```json
{ "telegram_user_id": "123456789", "username": "alice" }
```

### What no event ever contains

No event payload contains an API key, a key fragment, the bot service key, the signing secret, or a session token.

## Operator setup

### Generating and rotating the service key

1. In the admin panel, go to **System Settings → Authentication → OAuth Integrations → Telegram** and open the **Bot Integration** section.
2. Set **Bot Service Key** to a long random value (this is the value the bot presents as `Authorization: Bot <key>`) and enable **Enable Bot Integration**.
3. Saving with the key field left blank keeps the currently stored key unchanged — exactly like the SePay webhook key — so an operator can edit an unrelated field on the same form without retyping the key. This field never renders a previously stored value back into the input; it always starts blank.
4. To rotate: enter a new value and save, then update the bot's own configuration with the same new key. There is a short window where the old key stops working before the bot is updated; there is no dual-key grace period.
5. **Require Telegram Handle at Registration** (`TelegramHandleRequired`, defaults on) is independent of the above — it controls whether the sign-up form requires a handle, not whether the bot surface is reachable.

### Configuring the outbound callback

1. In the same Bot Integration section, set **Event Callback URL** to the bot's public HTTPS endpoint and **Event Callback Secret** to a shared secret. Both must be set together — while either is empty, no events are sent (the bot surface itself keeps serving requests regardless).
2. The secret field has the same write-only behavior as the service key: leave it blank to keep the stored value.
3. The callback address must be reachable from the gateway's network and should respond quickly — the gateway applies a short per-attempt timeout and does not wait indefinitely.

### Disabling the integration (rollback)

Turn off **Enable Bot Integration** and save. Every `/api/bot/v1` request then returns HTTP 404, identical to an unconfigured deployment — the whole surface disappears without touching any data. No column or binding is affected: `user.telegram_id` (verified) and `user.telegram_username` (display-only) are untouched, existing SePay orders and their settlement continue exactly as before, and the website's Telegram Login Widget binding and link-code redemption endpoints are unaffected because they do not live behind this flag. Turning **Require Telegram Handle at Registration** off separately just makes the sign-up field optional again; it does not touch existing accounts.
