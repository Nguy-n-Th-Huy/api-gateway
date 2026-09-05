## 1. Backend — effective status and group resolution

- [x] 1.1 In `controller/token.go`, add a pure function that derives a token's effective status from the token row plus a timestamp, applying disabled → expired → exhausted → enabled in that order (`common.TokenStatus*` constants), returning `enabled` for an unlimited-quota token with zero remaining quota, and performing no database write.
- [x] 1.2 Add table tests with `testify` in `controller/` covering: stored-disabled wins over a past expiry, past expiry wins over zero remaining quota, zero remaining quota with limited quota gives exhausted, zero remaining quota with unlimited quota gives enabled, `expired_time == -1` never expires, and a boundary case where the expiry equals the current timestamp. ← (verify: order of precedence matches spec "Key check reports the effective status without mutating stored state"; no test calls a DB)

## 2. Backend — public check endpoint

- [x] 2.1 Add a request DTO for the check endpoint with a `key` field, decoded through `common.DecodeJson` / `common.Unmarshal` (no direct `encoding/json` marshal or unmarshal calls).
- [x] 2.2 Add the `CheckTokenUsage` handler in `controller/token.go`: normalize the submitted key (trim whitespace, strip `Bearer `/`bearer ` prefix, strip `sk-` prefix, keep only the segment before the first remaining `-`), reject empty input with `400` and `i18n.MsgTokenNotProvided`, look the token up with `model.GetTokenByKey`, answer `gorm.ErrRecordNotFound` with the generic `i18n.MsgTokenInvalid`, and answer any other lookup error with `500`, `i18n.MsgTokenGetInfoFailed`, and a `common.SysError` log line that does not include the submitted key.
- [x] 2.3 In the handler, resolve the effective group: use `token.Group` when non-empty, otherwise `model.GetUserGroup(token.UserId, false)`; on lookup failure return the generic error rather than an invented group.
- [x] 2.4 Build the success response containing exactly `name`, `group`, `status`, `unlimited_quota`, `total_granted` (`RemainQuota + UsedQuota`), `total_used`, `total_available`, `expires_at` (raw `ExpiredTime`, `-1` = never), `created_time`, `accessed_time`, `model_limits_enabled`, `model_limits`, `available_models` (`model.GetGroupEnabledModels` for the effective group, empty list when the group enables none) — and no account id, username, or email. Share the field-building logic with `GetTokenUsage` rather than duplicating the field names, keeping `GetTokenUsage`'s existing response shape byte-for-byte unchanged.
- [x] 2.5 Register `POST /api/token/check` in `router/api-router.go` as a public route with `middleware.CORS()` and `middleware.CriticalRateLimit()` and no auth middleware; confirm it is not nested inside the `middleware.UserAuth()`-protected `tokenRoute` group. ← (verify: route resolves without credentials, is rate limited, and `GET /api/usage/token` still returns its original fields and still rejects disabled keys)
- [x] 2.6 Run `go build ./...` and `go test ./controller/...`; fix every failure in the files this change owns.

## 2b. Backend — setup script endpoint

- [x] 2b.1 Create `controller/setup_script.go` with the supported application registry: for each of `claude-code`, `codex`, `opencode`, `pi`, `oh-my-pi` record its display name, its model slots (Claude Code: opus/sonnet/haiku/subagent; Codex: small/medium/large; OpenCode: model/small_model; Pi and Oh My Pi: none), its documented install command per OS, and its config file path per OS. Use the table in `design.md` — Decisions as the authority; do not invent keys or paths.
- [x] 2b.2 Implement the script templates, one per application, each emitting both a PowerShell variant and a POSIX shell variant that: create the config file's parent directory when missing, parse an existing config file and stop with the file path when it cannot be parsed, merge only the keys this change owns, and write the result back. Templates must not export environment variables — the sole exception is Codex's `env_key` variable, which is persisted with a comment explaining that Codex reads the credential only from the environment.
- [x] 2b.3 Implement the handler: read application, OS, key and per-slot model selections from the request; normalize the key exactly as task 2.2 does; resolve the effective group; reject an unknown key with the same generic `i18n.MsgTokenInvalid` used by the check endpoint; reject an unknown application or OS; reject a missing slot naming the slot; reject any requested model that is not in `model.GetGroupEnabledModels(group)`; on success return the rendered script as `text/plain`. Build the gateway base URL from `system_setting.ServerAddress`. Do not log the query string or the key. ← (verify: every requested model is validated against the key's own group, and no response path leaks whether an unknown key ever existed)
- [x] 2b.4 Register the route in `router/api-router.go` as a public `GET` with `middleware.CORS()` and `middleware.CriticalRateLimit()` and no auth middleware.
- [x] 2b.5 Add table tests with `testify` covering: a model outside the key's group is rejected, an unknown application is rejected, a missing slot is rejected, each application's rendered script contains its documented config path and base URL, the Claude Code / OpenCode / Pi / Oh My Pi scripts persist no environment variable, and the Codex script persists exactly one. ← (verify: rejection cases return no script at all; scripts assert against the documented keys from design.md)

## 3. Frontend — feature scaffolding and API client

- [x] 3.1 Create `web/src/features/key-check/types.ts` with the report response type mirroring the backend fields from task 2.4, plus the API envelope type.
- [x] 3.2 Create `web/src/features/key-check/api.ts` calling `POST /api/token/check` through the shared `api` instance from `@/lib/api`, with the key in the request body.
- [x] 3.3 Create `web/src/features/key-check/lib/` with the Zod schema (key required, trimmed, minimum 8 characters, i18n message keys) and helpers that map a report to display values: effective status → the entry from `API_KEY_STATUSES`, quota figures → `formatQuota`, used share as a percentage, expiry (`-1`/`0` → never) and timestamps via Day.js, model restriction (`model_limits_enabled` false → all models).
- [x] 3.4 Create `web/src/features/key-check/constants.ts` for the i18n message keys and any query keys used by the feature.

## 4. Frontend — page, route and states

- [x] 4.1 Create `web/src/features/key-check/index.tsx` rendering inside `PublicLayout`: a labelled key input with React Hook Form + Zod resolver, a submit control, and the result region. Submit must work with both Enter and the button. Split presentational pieces into `components/` if the file approaches 200 lines.
- [x] 4.2 Implement all four states: idle (guidance text, no result panel), loading (submit disabled with a busy indicator), error (server message inline plus `toast.error`, form stays editable for resubmission, stale result not presented as current), success (result panel).
- [x] 4.3 Implement the result panel: key name, group, status via `StatusBadge` + `API_KEY_STATUSES`, used / remaining / granted quota, used-share bar, expiry, creation time, last-access time, model restriction; unlimited-quota keys state "unlimited" instead of a remaining figure or share; never render the submitted key in full (mask it if shown at all).
- [x] 4.4 Add accessibility: `label` bound to the input, `aria-live="polite"` on the result/error region, focus moved to the result region after a successful lookup, full keyboard operability, WCAG 2.1 AA contrast.
- [x] 4.5 Create `web/src/routes/key/index.tsx` with the project's GPL header, `createFileRoute('/key/')`, no auth `beforeLoad`, rendering the feature component. ← (verify: `/key` loads for an anonymous visitor with no sign-in redirect, and all four states render as specified)

## 4b. Frontend — model status section

- [x] 4b.1 Add the model status data hook: call `GET /api/perf-metrics/summary` through the shared `api` instance, with its own loading, empty and error states so a metrics failure never blocks the key check form.
- [x] 4b.2 Implement the model row: name, state badge (`Operational` at or above 90% success, `Down` below 90% with traffic, `No Data` with no traffic), and the bar strip built from `recent_success_rates`, coloured at the 90% / 50% boundaries with a distinct neutral colour for empty intervals. Keep the thresholds in a single constant.
- [x] 4b.3 Make each bar expose its interval and success rate as text on hover and to assistive technology, so colour is never the only carrier of meaning.
- [x] 4b.4 Narrow the list to the checked key's `available_models` once a key check succeeds, state that the list is scoped to the key's group, list a group model with no metrics entry as `No Data`, and fully replace the list when a different key is checked. ← (verify: switching keys leaves no model from the previous key, and a group model absent from metrics still appears)

## 4c. Frontend — setup section

- [x] 4c.1 Build the application selector (Claude Code, Codex, OpenCode, Pi, Oh My Pi) and the OS tabs (Windows, macOS/Linux), with a default selection so a command is visible without interaction.
- [x] 4c.2 Gate the whole section on a successful key check: before that, show a localized prompt to check a key first and offer no command.
- [x] 4c.3 Populate every model selector exclusively from `available_models`; when the group enables no model, state that and offer no command; when a different key is checked, repopulate and replace any selection the new group does not enable. Show no selector for Pi and Oh My Pi, with a note that the model is chosen inside that CLI.
- [x] 4c.4 Show the documented install command for the selected application and OS as a separate copyable prerequisite step.
- [x] 4c.5 Render the generated command with the key masked, and copy the unmasked working command to the clipboard with localized copy feedback. ← (verify: on-screen command never shows the full key, clipboard content does)

## 5. Localization

- [x] 5.1 Add every new user-facing string as a key in `web/src/i18n/locales/en.json`, appending only — no existing key renamed, reordered destructively, or removed.
- [x] 5.2 Add the matching Vietnamese translations in `web/src/i18n/locales/vi.json`, with correct diacritics, appending only. ← (verify: no page string is hardcoded; en and vi key sets match for this feature)

## 6. Frontend tests and checks

- [x] 6.1 Add `web/src/features/key-check/__tests__/validation.test.ts` covering the schema: empty input, whitespace-only input, input shorter than 8 characters after trimming, and a valid key.
- [x] 6.2 Add `web/src/features/key-check/__tests__/report-display.test.ts` covering the display mapping: limited-quota report → three quota figures and a 25% used share for 500000/1500000, unlimited-quota report → unlimited wording with no share, `expires_at === -1` → never-expires wording, `model_limits_enabled` false → all-models wording and true → the listed models, and each effective status value → its `API_KEY_STATUSES` entry.
- [x] 6.3 Add `web/src/features/key-check/__tests__/model-status.test.ts` covering the health classification and bar colours: 98% → `Operational`, exactly 90% → `Operational`, 20% with traffic → `Down`, no traffic → `No Data`, and one interval of each colour band including the empty-interval neutral colour.
- [x] 6.4 Add `web/src/features/key-check/__tests__/setup-command.test.ts` covering: model selectors offer only `available_models`, an empty `available_models` yields no command, the displayed command masks the key while the copy payload does not, and Pi / Oh My Pi render no model selector.
- [x] 6.5 Run `bun run typecheck` from `web/`, lint the files this change touches, and run the new test files; fix every error in owned files. ← (verify: typecheck and lint clean for owned files, all new test files pass)
