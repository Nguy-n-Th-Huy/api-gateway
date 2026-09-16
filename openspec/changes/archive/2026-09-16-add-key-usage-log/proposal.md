## Why

The public `/key` page reports a key's quota, status and model limits, but not *what the key has been spending that quota on*. A key holder — often someone without an account on the site — has to take the totals on faith: they cannot see which model was called, when, how many tokens it consumed, or what it cost. Every other surface that reports on a key already answers this: the console's usage-logs page for the owner, and the Telegram bot integration for a bot-linked account. The public page is the only key-facing surface that stops at the aggregate.

## What Changes

- Add a key-scoped usage-log section to the `/key` page: a table of that key's log entries (time, type, model, tokens, cost, duration) with paging, shown after a successful key check.
- Add a public, unauthenticated endpoint `POST /api/token/logs` returning the entries recorded against the submitted key, key in the JSON body only (never the URL or query string), rate-limited with `middleware.CriticalRateLimit()`.
- Return a whitelisted public entry shape — no account identity (user id, username), no client IP, no upstream channel, no raw `other` metadata — so the endpoint exposes usage without exposing the owner or the site's infrastructure.
- Reuse the existing key-scoped log query (`model.GetLogsByTokenIdPaginated`), moving it from the Telegram-specific model file next to `GetLogByTokenId` in `model/log.go`, since it now serves two surfaces. Its SQL and behavior are unchanged.
- Keep the Telegram bot's `/api/bot/v1/keys/logs` contract as it is; only the shared query helper moves.

## Capabilities

### Modified Capabilities

- `public-key-check`: the page gains a usage-log section and the endpoint behind it. The existing key-check report, page states, setup section and model-status section are unchanged.

## Impact

- **Backend**: `controller/token.go` (new `CheckTokenLogs` handler, reusing `normalizeTokenKey` and the check endpoint's error responses), `service/token_logs.go` (new public entry type and its single mapping function), `model/log.go` (receives `GetLogsByTokenIdPaginated`), `model/telegram_bot_usage.go` (function removed), `router/api-router.go` (one new public route).
- **Frontend**: `web/src/features/key-check/` gains the log API call, a React Query hook, a display lib and a section component; `index.tsx` renders the section; new keys in `web/src/i18n/locales/{en,vi}.json`.
- **Not touched**: database schema (no migration, so the three-database verification matrix covers query compatibility, not migration), `GET /api/usage/token`, the Telegram bot payload, the model-status and setup sections.
- **Security surface**: one more unauthenticated endpoint answering key lookups. Mitigations mirror `POST /api/token/check`: key in the body only, `CriticalRateLimit`, generic invalid-key message, no identity or infrastructure fields, no key in logs.
