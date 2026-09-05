## Why

Administrators cannot tell which channel is failing or slow under real traffic. The channel table only exposes `response_time` from the manual "Test" button, which measures a single synthetic probe at an arbitrary past moment — it says nothing about the error rate or latency users actually experience. Operators therefore learn about a degraded upstream from user complaints instead of from the console.

The data needed to answer this already exists: every consume log and every error log records `channel_id` and `use_time`. Nothing aggregates it per channel.

## What Changes

- Add a read-only aggregation over the existing `logs` table that computes, per channel, over a fixed trailing 24-hour window: total requests, successful requests, failed requests, success rate, and average latency.
- Add an admin endpoint `GET /api/channel/health` returning those metrics for all channels that had traffic in the window. Gated by the existing `authz.ChannelRead` permission.
- Cache the aggregation in memory with a short TTL so repeated console refreshes do not re-scan the log table.
- Add two columns to the channel management table in the web console: **Success rate** and **Avg latency**. Channels with no traffic in the window render an explicit "no data" state, not `0%`.
- Add the corresponding `en` and `vi` locale strings.

Not in scope, deliberately:

- No database schema change, no migration, no new column on `channels`.
- No effect on routing, retry, auto-disable, billing, or quota. The feature only reads.
- No rework of the channel model into an account-pool architecture.
- No user-selectable time window; the window is fixed at 24 hours.

## Capabilities

### New Capabilities

- `channel-health-metrics`: per-channel success rate and average latency derived from request logs, exposed to administrators through the channel management console.

### Modified Capabilities

<!-- None. No existing capability's requirements change. -->

## Impact

**Backend**

- `model/` — new read-only aggregation query against `LOG_DB` over the `logs` table, grouped by `channel_id`. No writes, no schema change.
- `service/` — in-memory cache wrapper with TTL, following the existing `service/rankings.go` cache pattern.
- `controller/channel.go` — new handler returning the aggregated metrics.
- `router/channel-router.go` — one new entry in `channelPermissionRoutes` with `permission: authz.ChannelRead`.

**Frontend**

- `web/src/features/channels/api.ts` — new fetch function.
- `web/src/features/channels/types.ts` — new zod schema for the response.
- `web/src/features/channels/components/channels-columns.tsx` — two new columns, reusing the existing `formatResponseTime` / `getResponseTimeConfig` helpers and `StatusBadge`.
- `web/src/i18n/locales/en.json`, `web/src/i18n/locales/vi.json` — new keys.

**Data and compatibility**

- The `logs` table may live in a separate database (`LOG_SQL_DSN`) and may be ClickHouse. The aggregation must therefore run as a standalone query on `LOG_DB` and must not join the `channels` table; results are merged with the channel list in Go.
- The aggregation SQL must be valid on SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6, and ClickHouse.

**Known measurement limitation**

- `use_time` is recorded as whole seconds (`time.Now().Unix() - StartTime.Unix()`), so any request faster than one second is stored as `0`. The reported average latency is therefore a lower bound, not an exact mean. This limitation must be surfaced to the operator in the UI rather than hidden.
