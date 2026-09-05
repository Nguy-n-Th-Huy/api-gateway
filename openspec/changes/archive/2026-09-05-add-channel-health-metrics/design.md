## Context

See `proposal.md` — Why, for motivation. Requirements live in `specs/channel-health-metrics/spec.md`.

Constraints that shape the approach:

- **Logs may not be co-located with channels.** `LOG_SQL_DSN` (`model/main.go:190`, `InitLogDB` at `model/main.go:230`) puts the `logs` table on a separate handle, `LOG_DB`. That store may be ClickHouse (`migrateClickHouseLogDB`, `model/main.go:391`), which the primary database explicitly rejects (`model/main.go:145`). Any query that joins `logs` to `channels` is therefore invalid.
- **The data already exists.** `Log` (`model/log.go:57`) carries `ChannelId` (indexed), `UseTime`, `Type`, and `CreatedAt` (indexed). `RecordConsumeLog` (`model/log.go:339`) writes `LogTypeConsume = 2`; `RecordErrorLog` (`model/log.go:278`) writes `LogTypeError = 5`. Both populate `ChannelId` and `UseTime`.
- **`use_time` is whole seconds.** Call sites compute it as `time.Now().Unix() - relayInfo.StartTime.Unix()` (`service/quota.go:170`, `service/quota.go:298`). There is no sub-second field.
- **The console already formats latency.** `formatResponseTime(timeMs, t)` and `getResponseTimeConfig(timeMs)` (`web/src/features/channels/lib/channel-utils.ts:369-401`) render a millisecond number into a `StatusBadge` with a severity variant, and are already used by the `response_time` column (`channels-columns.tsx:1141-1159`).
- **Four SQL dialects must accept the aggregation**: SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6, and ClickHouse for the log store.

## Goals / Non-Goals

**Goals:**

- Compute the metrics from data already being written, adding no write path and no schema change.
- Keep the aggregation valid on every supported log store.
- Reuse the console's existing latency formatting and badge components rather than introducing a parallel presentation.
- Bound the cost of the aggregation so console refreshes cannot hammer the log store.

**Non-Goals:**

- No configurable time window, no per-model or per-key breakdown, no historical time series.
- No feeding these metrics back into routing, retry, or auto-disable decisions. The values are advisory only.
- No sub-second latency accuracy. Achieving that would require changing what the relay records, which is a write-path change and out of scope.

## Decisions

### Aggregate on read from `logs` instead of maintaining counters on `channels`

Maintaining rolling counters on the `channels` row would need new columns, a migration validated across three primary databases, and a write in the relay hot path — with a real risk of lock contention on hot channels. Reading from `logs` needs neither. `logs.channel_id` and `logs.created_at` are both indexed, and the result set is one row per active channel, so the payload stays small.

Alternative considered: a materialized summary table refreshed by a background job. Rejected as premature — it reintroduces schema and migration cost to solve a performance problem that has not been observed, and the cache below already bounds query frequency.

### Query `LOG_DB` standalone; merge with channels in Go

The aggregation returns rows keyed by `channel_id`. The console already has the channel list from `GET /api/channel`; the frontend merges the two by id. This keeps the query legal when `logs` lives in another database or in ClickHouse.

Reference pattern: `SumUsedToken` (`model/log.go:676`) uses `LOG_DB.Table("logs")` with no join.

### Express the aggregation with `COUNT` and `SUM(CASE WHEN ...)`, divide in Go

```
SELECT channel_id,
       COUNT(*)                                   AS total_requests,
       SUM(CASE WHEN type = <error> THEN 1 ELSE 0 END) AS failed_requests,
       SUM(use_time)                              AS total_use_time_seconds
FROM logs
WHERE created_at >= <window start>
  AND type IN (<consume>, <error>)
  AND channel_id > 0
GROUP BY channel_id
```

`SUM(CASE WHEN ... THEN 1 ELSE 0 END)` is standard SQL accepted by all four engines. `AVG()` is avoided deliberately: the four engines disagree on the return type and rounding of an average over an integer column, which would make the Go scan target dialect-dependent. Summing integers and dividing in Go keeps the arithmetic in one place, makes the divide-by-zero guard explicit, and makes the conversion unit-testable without a database.

`success_requests` is derived as `total_requests - failed_requests` rather than summed separately, since the `WHERE` clause already restricts rows to the two types.

### Success rate as a ratio in `[0, 1]`, latency in milliseconds

The API returns `success_rate` as a float in `[0, 1]` and `avg_latency_ms` as an integer count of milliseconds (`total_use_time_seconds * 1000 / total_requests`). Milliseconds are chosen so the frontend can pass the value straight into the existing `formatResponseTime` / `getResponseTimeConfig` helpers with no adapter. Percentage formatting is a presentation concern and stays in the frontend.

`total_requests` is guaranteed positive for every returned row because `GROUP BY` only emits groups with at least one row, but the Go conversion still guards against zero so the helper is safe to unit-test and to call from any future caller.

### Short TTL in-memory cache, following the rankings pattern

`service/rankings.go:112-161` already establishes the house pattern: a package-level map guarded by a mutex, entries carrying `expiresAt`, and a TTL constant. The same shape is used here with a shorter TTL (60s) because operators watching a degrading channel expect the numbers to move. A single cache entry suffices — the endpoint takes no parameters.

Alternative considered: Redis. Rejected — this is a per-instance advisory read, not shared state that must be consistent across replicas, and adding a Redis dependency for it would make the feature fail differently depending on deployment topology.

### One `useQuery` in `useChannelsColumns`, not one per cell

`useChannelsColumns` (`channels-columns.tsx:587`) is already a hook, so the health query is called once there and the resulting lookup map is closed over by the two cell renderers. Calling the hook inside each cell would create one subscription per visible row for the same query key; deduplicated at the network layer, but needless subscription churn.

### Distinguish "no traffic" from "0% success"

A channel absent from the aggregation is rendered as `-` in both columns. Rendering `0%` for an idle channel would invert the meaning of the most alarming value in the table. The frontend therefore keys off the absence of a map entry, not off a zero value.

### Disclose the latency precision limit in the UI

Because sub-second requests record `use_time = 0`, the average is a lower bound. The average latency column header carries a tooltip stating that the value is approximate and derived from one-second-granularity data. Presenting the number without that qualifier would be misleading, and the alternative — suppressing the column — would discard genuinely useful signal for the slow channels that matter most.

## Risks / Trade-offs

- **Aggregation cost grows with log volume** → The `WHERE` clause is bounded by the indexed `created_at` and the 60-second cache caps execution to at most once per minute per instance. If a deployment with a very large `logs` table still sees pressure, the follow-up is a summary table, not a wider query.
- **Average latency understates true latency** → Documented in the spec, disclosed in the UI tooltip. Not silently corrected, because any correction would be a guess.
- **Streaming requests inflate the average** → A long streamed completion legitimately occupies the channel for its whole duration, so it is counted as-is. The metric answers "how long do requests on this channel take", not "how fast is the first token". Noted here so a future reader does not mistake it for a bug.
- **Error logs can be recorded for causes outside the channel's control** (client aborts, quota rejections attributed to a channel) → The success rate is advisory and never drives auto-disable, so a skewed rate misleads at worst and breaks nothing. Kept simple deliberately; refining the classification would require reading `Log.Other`, which is a JSON blob and cannot be filtered in SQL portably.
- **ClickHouse cannot be exercised locally in the default developer setup** → The aggregation deliberately uses only portable SQL constructs, and the Go-side arithmetic is unit-tested independently of any engine. If the full multi-engine matrix cannot be run, that must be reported as a gap rather than claimed as verified.

## Migration Plan

None. The change adds no schema, no migration, and no persisted state. Deployment is a normal rollout; rollback is reverting the build. The new endpoint is additive, so an older frontend against a newer backend is unaffected, and a newer frontend against an older backend renders the no-data indicator via the metrics-unavailable path already required by the spec.
