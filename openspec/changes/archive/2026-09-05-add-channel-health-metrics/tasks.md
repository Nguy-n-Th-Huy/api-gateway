## 1. Backend aggregation

- [x] 1.1 Add `model/channel_health.go` with a `ChannelHealthStat` result type (channel id, total requests, success requests, failed requests, success rate, average latency in ms) and a raw aggregation row type holding the summed integers.
- [x] 1.2 Implement the read-only aggregation against `LOG_DB.Table("logs")`: `COUNT(*)`, `SUM(CASE WHEN type = ? THEN 1 ELSE 0 END)` for failures, `SUM(use_time)`, filtered by `created_at >= ?`, `type IN (consume, error)`, `channel_id > 0`, grouped by `channel_id`. No join to `channels`, no `AVG()`.
- [x] 1.3 Implement the pure conversion from a raw aggregation row to `ChannelHealthStat`: success = total − failed, success rate as a `[0, 1]` ratio, average latency as `total_use_time_seconds * 1000 / total_requests`, with an explicit guard for a zero total. ← (verify: divide-by-zero guarded; seconds→ms conversion correct; success rate never exceeds 1 nor goes negative)
- [x] 1.4 Add `model/channel_health_test.go` covering the conversion with `testify`: zero total, all failures, all successes, mixed, and sub-second durations recorded as zero. Table-driven, deterministic, no database. ← (verify: tests assert exact expected values and actually fail if the conversion is broken)

## 2. Caching and endpoint

- [x] 2.1 Add `service/channel_health.go` with a 60-second TTL in-memory cache guarded by a mutex, following the shape of `rankingCacheItem` / `rankingCacheMu` / `rankingCacheTTL` in `service/rankings.go:112-161`. Single cache entry; the endpoint takes no parameters.
- [x] 2.2 Add a `GetChannelHealth` handler in `controller/channel.go` that returns the cached stats via the existing success-response helper used by neighbouring channel handlers. Response shape per `design.md` — Success rate as a ratio in `[0, 1]`, `avg_latency_ms` as an integer.
- [x] 2.3 Register `{method: http.MethodGet, path: "/health", permission: authz.ChannelRead, handler: controller.GetChannelHealth}` in `channelPermissionRoutes` in `router/channel-router.go`. ← (verify: route resolves without colliding with `/:id`; permission gate is `ChannelRead`; an unauthorised caller is rejected)
- [x] 2.4 Run `go build ./...` and `go vet ./...` for the root module, and `cd relaykit && GOWORK=off go build ./...` to confirm the independent module still builds. ← (verify: all three commands succeed; report any failure rather than working around it)

## 3. Frontend data layer

- [x] 3.1 Add the response zod schema and inferred types for channel health to `web/src/features/channels/types.ts`, following the existing zod usage in that file.
- [x] 3.2 Add the fetch function for `GET /api/channel/health` to `web/src/features/channels/api.ts`, following the existing `api.get` usage in that file.
- [x] 3.3 Add a query key and a `useChannelHealth` hook returning a lookup keyed by channel id, so a missing entry is distinguishable from a zero value. Place it with the feature's other hooks and query keys.

## 4. Frontend presentation

- [x] 4.1 Call `useChannelHealth()` once inside `useChannelsColumns` (`web/src/features/channels/components/channels-columns.tsx:587`) and close the lookup over the new cell renderers — not once per cell.
- [x] 4.2 Add the success rate column: percentage for channels present in the lookup, the `-` no-data indicator for channels absent from it or when the query failed. Never render `0%` for a channel with no traffic.
- [x] 4.3 Add the average latency column reusing `formatResponseTime` and `getResponseTimeConfig` from `web/src/features/channels/lib/channel-utils.ts:369-401` with `StatusBadge`, matching the existing `response_time` column at `channels-columns.tsx:1141-1159`. Same `-` no-data behaviour.
- [x] 4.4 Add a tooltip on the average latency column header disclosing that the value is approximate because durations are recorded at one-second granularity. ← (verify: the disclosure is actually reachable in the UI, not only present in the source)
- [x] 4.5 Add every new user-facing string to both `web/src/i18n/locales/en.json` and `web/src/i18n/locales/vi.json`, keyed by the English source string, and use `t()` for all of them. No hardcoded UI text. ← (verify: no literal UI string left in the new code; `en` and `vi` both contain each new key)

## 5. Frontend tests and checks

- [x] 5.1 Add a test under `web/src/features/channels/lib/__tests__/` for the health lookup and its formatting decisions: channel present, channel absent, query failed, zero success rate rendered as `0%` only when traffic exists. ← (verify: the "absent channel" case asserts the no-data indicator and would fail if it rendered `0%`)
- [x] 5.2 Run the frontend lint, typecheck, and test commands defined in `web/package.json` and report the results without suppressing failures. ← (verify: commands actually ran; any failure is reported, not silenced)
  - `bun run typecheck`: passed, no errors.
  - `bun run lint`: full-repo run has pre-existing errors in unrelated files (`src/features/home/...`, `src/features/setup/...`, `src/assets/...`, `scripts/sync-i18n.mjs`, etc.) not touched by this change; a scoped `oxlint` run limited to every file this change added or edited passed with zero errors/warnings.
  - `bun run test`: full-repo run has one pre-existing failing test unrelated to this change (`src/features/users/components/dialogs/__tests__/user-binding-dialog.test.tsx`, a jsdom "navigation to another Document" timeout that passes when run in isolation — flaky under the full suite, not caused by this change); a scoped run of `src/features/channels` passed 25/25 tests, including the new `channel-health.test.ts`.

## 6. Multi-engine verification

- [x] 6.1 Exercise the aggregation query against SQLite, MySQL, and PostgreSQL. If any engine cannot be run in this environment, state which ones were exercised and which were not — do not claim multi-database compatibility that was not tested. ← (verify: the report names the engines actually exercised, with commands and results)
  - SQLite: exercised via `go test ./model/... -run TestGetChannelHealthStatsAggregatesOverSQLite` (in-memory SQLite, `model/channel_health_sqlite_test.go`) — passed.
  - PostgreSQL: exercised via `docker exec -i new-api-dev-pg psql -U root -d new-api` against the running PostgreSQL 15.19 dev container, running the exact aggregation SQL shape in a rolled-back transaction on a temp table — results matched expected values exactly, no permanent state left behind.
  - MySQL: NOT exercised. No MySQL instance was available in this environment; a MariaDB 11.5 container was running but belongs to an unrelated project on this machine, so it was not used for this repo's verification. This is a reported gap, not a claimed pass.
  - ClickHouse (log store only): NOT exercised, consistent with design.md's already-documented gap.

## 7. Post-verification fixes (independent verification MAJOR findings)

- [x] 7.1 Stop reusing `formatResponseTime`/`getResponseTimeConfig` (task 4.3) for the average latency column: those treat `0` as "manual test never run", but `avg_latency_ms === 0` here means every request in the window completed in under one second, per the "Sub-second requests reported as zero" scenario. Added `formatAvgLatency` / `getAvgLatencyConfig` in `web/src/features/channels/lib/channel-utils.ts`, rendering `0` as `< 1s` with the healthy badge variant, and using the existing formatting/coloring for any positive value. `formatResponseTime` / `getResponseTimeConfig` and the `response_time` column are unchanged. ← (verify: a channel with `avg_latency_ms: 0` renders `< 1s`, not `Not tested`; the `< 1s` string is a translated key in both `en.json` and `vi.json`)
- [x] 7.2 Add a regression test on the rendered table cells, not only the `getChannelHealthCellView` view-model. Extracted the two cell renderers into `web/src/features/channels/components/channel-health-cells.tsx` (`SuccessRateCell`, `AvgLatencyCell`) so they're testable without importing `channels-columns.tsx`'s much larger dependency graph, and added `web/src/features/channels/components/__tests__/channel-health-cells.test.tsx`. ← (verify: temporarily removing the `!hasData` branch in either cell makes the new test fail; restoring it makes the test pass again)
