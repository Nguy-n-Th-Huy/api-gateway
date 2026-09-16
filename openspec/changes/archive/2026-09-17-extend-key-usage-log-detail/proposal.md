## Why

The `/key` page's usage log answers "what has this key spent" but not "what did each request look like". Its table collapses prompt and completion into one `Tokens` cell, has no cached-token figure even though the log column records one, and marks nothing about the request's outcome — so a key holder cannot tell a failed request from a successful one, cannot see how much of the bill was cache reads, and cannot tell a chat call from an image generation.

The stored log row already carries all of it: `prompt_tokens` / `completion_tokens`, the log type (`consume` / `refund` / `error`), and the user-visible `other` metadata with `cache_tokens`, `cache_creation_tokens` and `request_path`. Only the public endpoint's field whitelist and the table's columns stand in the way.

## What Changes

- The public entry gains three fields, all of which the owning account already sees on its own log page: `request_path` (the relay endpoint), `cache_tokens` (cache-read tokens) and `cache_creation_tokens` (cache-write tokens). They are lifted out of the log's `other` blob server-side, which stays excluded as a whole.
- The usage-log table splits the token figure the way the console does — **Input**, **Cache**, **Output** — shows the request **Type** as the endpoint's kind (Chat, Responses, Messages, Image, Embeddings, Audio, Rerank, …) falling back to the log-type wording (Refund, Consume, Error) for an entry that has no endpoint, and shows the request's outcome as a **Status** badge: `Success` for a consume entry, `Failed` for an error entry, nothing for entries that are neither.
- IP stays out, as do account identity, upstream channel and the raw metadata blob: the entry remains an explicit whitelist with exactly one producer.
- No change to the table's paging, states, ordering, or to the rest of the page.

## Capabilities

### Modified Capabilities

- `public-key-check`: the entry field set and the usage-log table's columns change. Every other requirement of the capability — the report, the page states, the setup section, the model-status section, the privacy rule, rate limiting — is unchanged.

## Impact

- **Backend**: `service/token_logs.go` (three more fields plus the metadata parse), `controller/token_logs_test.go` (fixtures and assertions).
- **Frontend**: `web/src/features/key-check/types.ts`, `lib/log-display.ts` (request-kind mapping, status mapping, split token cells), `components/usage-log-section.tsx`, `__tests__/usage-log-display.test.ts`, and the new labels in `web/src/i18n/locales/{en,vi}.json`.
- **Not touched**: `model.Log`, the log writers, the endpoint's route, paging, privacy fields, the Telegram bot payload.
