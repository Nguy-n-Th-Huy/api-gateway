## 1. Backend — shared query and public entry shape

- [x] 1.1 Move `GetLogsByTokenIdPaginated` from `model/telegram_bot_usage.go` to `model/log.go`, next to `GetLogByTokenId`, and rewrite its doc comment for a surface-neutral reader (key-scoped log query: count, page, dialect-aware ordering, `formatUserLogs`). Keep its SQL and signature byte-for-byte; the Telegram bot call site must keep working unchanged.
- [x] 1.2 Add `service/token_logs.go` with `PublicKeyLogEntry` holding exactly `created_at`, `type`, `model_name`, `quota`, `prompt_tokens`, `completion_tokens`, `use_time`, `is_stream`, `group`, `request_id`, plus `BuildPublicKeyLogEntries(logs []*model.Log) []PublicKeyLogEntry` — the single mapping that drops `user_id`, `username`, `ip`, `channel`, `channel_name`, `token_id`, `token_name` and `other`. Return an empty, non-nil slice for no logs so the response carries `items: []`, never `null`.

## 2. Backend — public logs endpoint

- [x] 2.1 Add `checkTokenLogsRequest` (field `key`, decoded with `common.DecodeJson`) and `CheckTokenLogs` in `controller/token.go`: normalize the key with `normalizeTokenKey`, reject empty input through `respondTokenCheckKeyRequired`, look the token up with `model.GetTokenByKey`, answer `gorm.ErrRecordNotFound` with the generic `i18n.MsgTokenInvalid`, and any other error with `respondTokenCheckServerError` plus a `common.SysError` line that does not contain the submitted key. No new i18n message was needed: the check endpoint's three messages already cover every rejection path.
- [x] 2.2 Page the entries with `common.GetPageQuery(c)` and `model.GetLogsByTokenIdPaginated`, then respond with `common.ApiSuccess(c, pageInfo)` after `SetTotal` and `SetItems(service.BuildPublicKeyLogEntries(logs))`; a query failure answers `respondTokenCheckServerError` and logs without the key.
- [x] 2.3 Register `POST /api/token/logs` on the existing public token route group in `router/api-router.go`, with `anonymousRequestBodyLimit` and the group's `middleware.CORS()` / `middleware.CriticalRateLimit()` — not inside the `middleware.UserAuth()`-protected `tokenRoute` group. ← (verified: `router/api_router_test.go` now asserts `POST /api/token/logs` is routed, not 404, alongside `POST /api/token/check` and `GET /api/token/:id`; the suite passes)

## 3. Backend — tests

- [x] 3.1 Add `controller/token_logs_test.go` with a fixture that opens the existing in-memory SQLite harness and migrates `model.Token`, `model.User` and `model.Log`, then seeds two tokens whose log rows interleave.
- [x] 3.2 Test the success path: the handler returns only the checked key's entries, newest first, with `total` counting every entry of that key; the submitted key is pasted as `"  Bearer sk-…-extra  "` so normalization is covered too.
- [x] 3.3 Test the privacy contract at the JSON level: the serialized response contains none of `user_id`, `username`, `email`, `ip`, `channel`, `channel_name`, `token_id`, `token_name`, `other`, while carrying `created_at`, `type`, `model_name`, `quota`, `prompt_tokens`, `completion_tokens`, `use_time`, `is_stream`, `group`, `request_id` — asserted both on the JSON keys and on the seeded username / client IP values.
- [x] 3.4 Test paging: `page_size` below the number of entries returns that many entries with the full `total`, and a second page returns the remainder; a key with no entries returns `total` 0 with `"items":[]`.
- [x] 3.5 Test the rejection paths: unknown key returns the generic localized invalid-key message with no entries, and a whitespace-only key returns `400`.
- [x] 3.6 Run `go build ./...` and `go test ./controller/... ./model/... ./service/...`; fix every failure in the files this change owns. ← (controller, model, router and service/authz pass. `service` has two pre-existing failures, `TestObserveChannelAffinityUsageCacheByRelayFormat_MixedMode` and `…_UnsupportedModeKeepsEmpty`, caused by cross-test global state: they pass in isolation and fail identically on a pristine `HEAD` checkout in a scratch worktree, so they are not this change's to fix.)

## 4. Frontend — data layer

- [x] 4.1 Add the entry and page types to `web/src/features/key-check/types.ts` mirroring the backend field set from task 1.2, plus the API envelope.
- [x] 4.2 Add `TOKEN_LOGS_ENDPOINT` and the page size to `web/src/features/key-check/constants.ts`.
- [x] 4.3 Add `fetchTokenLogs(key, page)` to `web/src/features/key-check/api.ts`: `POST` the key in the body with `p` / `page_size` in the query string, through the shared `api` instance, reusing the feature's existing error type and localized fallback message.
- [x] 4.4 Add `web/src/features/key-check/hooks/use-token-logs.ts`: a React Query hook enabled only once a key has been checked, keyed by the key and the page, that does not carry the previous key's entries into a new key's section and restarts a newly checked key at page one.

## 5. Frontend — section and page

- [x] 5.1 Add `web/src/features/key-check/lib/log-display.ts` with the pure display helpers: entry timestamp and duration formatting through `@/lib/format`, cost through the log-quota formatter, prompt/completion token counts, log-type label lookup through the existing usage-logs vocabulary, and the page count from `total` and the page size. The type vocabulary is imported from `@/features/usage-logs/constants` rather than that feature's `lib/utils`, whose API imports would pull the console's log table into the public page's bundle.
- [x] 5.2 Add `web/src/features/key-check/components/usage-log-section.tsx`: a card whose title is the existing "Usage logs" wording, an instruction state before any check, a loading state, an error state with a retry control, a localized empty state, and the table with time, type, model, tokens, cost and duration columns plus previous/next paging showing the current and total page count. Keyboard operable, labelled for assistive technology, never rendering the key.
- [x] 5.3 Render the section in `web/src/features/key-check/index.tsx` between the result panel and the model status section, passing the checked key.
- [x] 5.4 Add `web/src/features/key-check/__tests__/usage-log-display.test.ts` covering the display helpers' real contracts: page count for an empty result, an exact multiple and a remainder; token rendering for a zero-token entry and for prompt-only entries; duration dash and seconds; type labels for a known and an unrecognized type; the model-name fallback.

## 6. Localization

- [x] 6.1 Add every new user-facing string through `web/scripts/add-missing-keys.mjs` (never by editing `locales/*.json` directly) into `en` and `vi` — the two locales this project ships — then run `bun run i18n:sync` and delete the temporary script. ← (verified: `find-missing-keys.mjs` reports no missing key from this feature — only three pre-existing keys from other features — and `_sync-report.json` reports `missingCount`/`extrasCount`/`untranslatedCount` 0 for both locales)

## 7. Verification

- [x] 7.1 `cd web && bun run typecheck` and lint the touched files; fix every error in them. ← (`tsgo -b` clean; `oxlint` clean on the key-check feature; `oxfmt --write` is a no-op on every touched file)
- [x] 7.2 `cd web && bun run test` scoped to the key-check feature, then the affected suites. ← (key-check: 5 files, 52 tests, all passing)
- [x] 7.3 Smoke test against a running instance: build the production bundle, run the server embedding it against a scratch SQLite database, seed one key with 13 entries and one key with none, then open `/key` and check both: the section lists that key's entries newest first, paging moves through them with the boundaries disabled as specified, the empty key shows the empty state, a switched key drops the previous rows, aborting the log request shows the localized error with a working Retry, and the Vietnamese locale renders every new string. `POST /api/token/logs` was also called directly: the key travels in the body, `?key=…` alone is answered `400`, an average key page returns `total` 13 with 10 entries, `page_size=500` is capped at 100, and the response carries none of `user_id`, `username`, `ip`, `channel`, `other`.
- [x] 7.4 Database compatibility: the key-scoped query now has a dialect test (`model/log_token_query_test.go`) that runs the same assertions — per-key counting, newest-first paging, the user-visibility formatting — on whatever database `TEST_MYSQL_DSN` / `TEST_POSTGRES_DSN` point at, and on an in-memory SQLite database unconditionally. This change adds no schema or migration, so the fresh-vs-upgrade matrix does not apply. See the verification record below.

## 8. Archive

- [x] 8.1 Merge the change's `specs/public-key-check/spec.md` requirements into `openspec/specs/public-key-check/spec.md`, moving the change directory to `openspec/changes/archive/2026-09-16-add-key-usage-log/`, with no edit to any other capability's spec.

## Verification record — database engines

Command (run from the repository root; DSNs point at engines reachable from this machine):

```
TEST_MYSQL_DSN='smoke:smokepass@tcp(127.0.0.1:3306)/newapi_test?charset=utf8mb4&parseTime=True&loc=Local' \
TEST_POSTGRES_DSN='host=127.0.0.1 user=postgres password=smokepass dbname=newapi_test port=5432 sslmode=disable TimeZone=UTC' \
go test ./model/ -run 'TestGetLogsByTokenIdPaginated' -count=1 -v
```

| Engine | Version | Result |
|---|---|---|
| SQLite (glebarez/sqlite, the driver the app embeds) | in-memory | PASS |
| MySQL | 8.4.11 (Ubuntu package) | PASS |
| PostgreSQL | 18.6 (Ubuntu package) | PASS |

The application's default SQLite path (`one-api.db`, WAL journal) is exercised end to end by the smoke test in 7.3, which ran the built server against a scratch database created by the app's own migration.