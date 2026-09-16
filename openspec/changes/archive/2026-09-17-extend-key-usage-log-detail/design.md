## Context

`/key`'s usage-log table is fed by `POST /api/token/logs`, whose entries are mapped by `service.BuildPublicKeyLogEntries` from `model.Log` rows. The rows already contain everything a fuller table needs (see `service/log_info_generate.go`, which writes `request_path`, `cache_tokens` and `cache_creation_tokens` into the user-visible metadata), so this change is a whitelist extension plus a table redesign — no new query, no schema, no writer change.

## Goals / Non-Goals

**Goals**

- Each row answers: which endpoint, did it work, how many input / cached / output tokens, what did it cost, how long did it take.
- The extra fields come from data the owning account already sees on its own log page.
- The privacy contract (no account identity, no IP, no channel, no raw metadata) and the single-producer rule stay intact.

**Non-Goals**

- First-response time, ratios, billing path, or the rest of the console's details dialog — the key page stays a summary, not a debugging surface.
- A details/expand row, filters, or export.
- Changing the Telegram bot payload.

## Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | Lift `request_path`, `cache_tokens`, `cache_creation_tokens` out of `other` into typed fields; keep `other` itself excluded | The consumer needs three values, not a blob. Extracting them server-side keeps the whitelist explicit and means a new key added by a log writer can never reach the public surface by accident. |
| 2 | Cache counts are clamped to a non-negative 32-bit range while parsing | Token counts come from upstream usage reports; the table is display-only, and the project's rule is that an unbounded upstream number must never reach arithmetic unclamped. |
| 3 | A malformed `other` yields no extra fields rather than failing the request | One corrupt row must not break the whole page; the entry still carries time, model, quota and tokens. |
| 4 | Request kind is derived on the client from `request_path` | The mapping is presentation (and localized); the server exposes the fact (the path) rather than a vocabulary. Finer-grained paths map by longest prefix, so `/v1/chat/completions` never falls into `/v1/completions`. |
| 5 | Status derives from the log type: `consume` → Success, `error` → Failed, anything else → no badge | This mirrors what the row means. Rendering a refund as "Success" or an error as "Success" would both be wrong, so entries that are neither get an empty status. |
| 6 | Cost stays as a column | The page exists to answer what the key spent; the reference layout does not show cost, but dropping it would lose the one figure the page was built for. |

## Risks / Trade-offs

- **More columns on a public page**: nine columns on a narrow screen scroll horizontally inside the card (the table is already wrapped in an overflow container); the mobile layout is unchanged otherwise.
- **Fallback wording**: an entry with no endpoint shows the log-type label (Refund / Consume / Error / Unknown) in the Type column, so the column mixes two vocabularies. Accepted: the alternative is an empty column for refund entries, which reads worse.
- **Cache wording**: the cell shows `↓` for cache-read and `↑` for cache-write tokens, the same convention the console's own tokens column uses.

## Migration

None — no schema change, no data change, additive response fields.
