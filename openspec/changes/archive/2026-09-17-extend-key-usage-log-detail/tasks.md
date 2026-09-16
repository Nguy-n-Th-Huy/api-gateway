## 1. Spec

- [x] 1.1 Write the change (proposal, design, tasks) and the `public-key-check` delta that replaces the entry field set and the page's column list.

## 2. Backend

- [x] 2.1 Extend `service.PublicKeyLogEntry` with `cache_tokens`, `cache_creation_tokens` and `request_path`, and parse them out of the log's `other` metadata with a malformed blob yielding empty/zero values instead of an error.
- [x] 2.2 Clamp the parsed cache counts into a non-negative 32-bit range before they reach the response.
- [x] 2.3 Keep the whitelist closed: no `other`, `user_id`, `username`, `ip`, `channel`, `channel_name`, `token_id` or `token_name` in the response. ← (verified: the handler test asserts the JSON carries none of those keys and does not even contain the metadata key `model_ratio`, even though three of its values are lifted out)
- [x] 2.4 Extend `controller/token_logs_test.go`: a consume row carrying cache and path metadata surfaces all three values; a row with unreadable metadata still returns with zero/empty extras; an oversized count is clamped instead of wrapping; the privacy assertions still hold.

## 3. Frontend

- [x] 3.1 Add the three fields to `KeyUsageLogEntry`.
- [x] 3.2 In `lib/log-display.ts`: map `request_path` to a request-kind label (most specific path first, unknown paths fall back to the log-type wording), derive the outcome from the log type (consume → success, error → failed, otherwise none), and split the token cell into input / cache (read ↓, write ↑) / output with digit grouping.
- [x] 3.3 Rebuild the table: Time, Model, Type (kind badge), Status (outcome badge), Input, Cache, Output, Cost, Duration — no IP column, no key, paging and states unchanged. Widen the page container to `max-w-7xl` (1280px, up from `max-w-3xl`'s 768px) so nine columns get real room without horizontal scrolling on a desktop viewport.
- [x] 3.4 Extend `__tests__/usage-log-display.test.ts` over the new mappings: kind for specific paths (including the `/v1/chat/completions` vs `/v1/completions` boundary), nested paths, unknown path fallback, outcome for consume/error/refund, cache cell with read only, write only, both, and neither.
- [x] 3.5 Add the new labels (`Responses`, `Messages`, `Completions`, `Moderation`, `Realtime`, `Midjourney`) through `web/scripts/add-missing-keys.mjs`, then `bun run i18n:sync`, and delete the temporary script. ← (verified: sync report 0 missing / 0 extras / 0 untranslated for both locales, no `t()` key of this feature missing from `en.json`)

## 4. Verification

- [x] 4.1 `go test ./controller/ ./model/` and `bun run typecheck`, `oxlint` on the touched files, `bun run test src/features/key-check`. ← (backend ok; typecheck clean; oxlint clean; 64 frontend tests pass; `oxfmt --write` is a no-op on every touched file)
- [x] 4.2 Rebuild the production bundle, run the server against a seeded database whose rows carry cache and path metadata, and check in a browser that the columns render, the status badge distinguishes a failed row, the empty and error states still work, and neither the IP nor any account field appears. ← (verified on the dev server and again on the production bundle: Chat/Responses/Image/Refund kind badges, green Success and red Failed, `86,272 | ↓86,272 ↑1,024 | 71`, refund row blank status, error state with a working Retry, empty state, Vietnamese labels, no `IP` header and no owner/IP string anywhere on the page)
- [x] 4.3 `bun run build` clean, then report how to publish (push + branch workflow to Docker Hub).

## 5. Archive

- [x] 5.1 Merge the delta into `openspec/specs/public-key-check/spec.md` and move the change to `openspec/changes/archive/2026-09-17-extend-key-usage-log-detail/`.
