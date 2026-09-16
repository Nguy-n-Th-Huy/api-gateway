## Context

The `/key` page already answers "what state is this key in" (`POST /api/token/check`) and "what can it call" (`available_models`). It cannot answer "what has it been used for". Two other surfaces already do, over the same `logs` table:

- the console's usage-logs page, scoped to the signed-in owner (`model.GetUserLogs`);
- the Telegram bot's key-scoped log (`model.GetLogsByTokenIdPaginated` via `POST /api/bot/v1/keys/logs`).

The public page needs the second shape: entries filtered by `token_id`, paginated, with the key supplied in the request body.

## Goals / Non-Goals

**Goals**

- A key holder sees the key's own recent requests: when, which model, how many tokens, what cost, how long.
- No account identity, no client IP, no upstream channel, no raw log metadata in the response.
- One log query implementation for every key-scoped surface.
- The page's existing sections, endpoint and contract stay untouched.

**Non-Goals**

- Filters (time range, type, model) on the public page — the console has them; the public section shows the key's most recent entries with paging only.
- Prompt/completion content, artifacts, or per-request pricing breakdowns.
- Changing the Telegram bot payload, which deliberately returns the raw entry set for its own authenticated caller.
- Any schema or migration change.

## Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | `POST /api/token/logs`, key in the JSON body | Same rule the check endpoint already follows: the key must never reach access logs or `Referer` headers. `GET` cannot carry a body reliably, so the endpoint is a `POST`. |
| 2 | Paging through the existing query parameters (`p`, `page_size`), parsed with `common.GetPageQuery` | Same clamping (max 100, project default size) every other list endpoint applies; no new paging vocabulary. The key is the only secret, and it is not in the query string. |
| 3 | A whitelisted entry type, produced in `service` next to `TokenCheckReport` | The console and bot surfaces return the raw `model.Log`, which carries `user_id`, `username`, `ip`, `channel`, `channel_name`, `token_id`, `token_name` and `other`. Mapping to an explicit field set in one function makes an accidental leak a code change, not a forgotten filter. |
| 4 | Reuse `GetLogsByTokenIdPaginated`, moved from `model/telegram_bot_usage.go` to `model/log.go` | It is already dialect-aware (SQLite/MySQL/PostgreSQL use `id desc`, ClickHouse uses `created_at desc, request_id desc`) and already applies `formatUserLogs`. Two copies would be a second convention beside the existing one. |
| 5 | Entries are the key's whole log, not `type = consume` only | Matches the console's key-scoped view and the bot endpoint: refunds and errors on the same key stay visible, and the entry carries `type` so the page can label it with the existing log-type vocabulary. |
| 6 | Frontend loads entries with React Query, keyed by `['key-check','token-logs', key, page]` | Follows `use-model-status`; a metrics/logs failure never blocks the key check itself, and paging reuses the cache instead of refetching page one. |
| 7 | Section renders before a check with an instruction, not hidden | Matches the setup and model-status sections: the page's shape is stable, only its content is gated. |

## Risks / Trade-offs

- **More exposure per lookup**: a key holder can now see request history, not just totals. Accepted — the same person can spend the key, and the response excludes identity, IP and channel, so it describes the key's usage without describing the owner or the upstream route.
- **`request_id` is returned**: it is a per-request opaque id, useful for support. It carries no account or channel information.
- **ClickHouse log database**: ordering differs by dialect and is already handled by the moved helper; this change adds no new SQL.
- **Enumeration**: the endpoint is rate-limited by `middleware.CriticalRateLimit()` exactly like the check endpoint, and answers unknown keys with the same generic message.

## Migration

None. No schema change and no data change; the new route is additive.
