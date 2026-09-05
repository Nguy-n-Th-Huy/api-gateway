## Why

API key holders currently have no way to inspect a key's remaining quota, group, or status without signing in to the console — and a key can be handed to someone who has no account on the site at all. The only public-ish surface today, `GET /api/usage/token`, sits behind `middleware.TokenAuthReadOnly()`, which rejects disabled keys with `401` and omits the two fields users ask for most: the key's group and its status.

## What Changes

- Add a public page at `/key` (no sign-in required) where a visitor pastes an API key and sees that key's full usage report: key name, group, status, granted / used / remaining quota, unlimited-quota flag, expiry, creation time, last-access time, and model restrictions.
- Add a public backend endpoint `POST /api/token/check` that accepts the key in the JSON body (never in the URL or query string, so keys do not leak into access logs), rate-limited with `middleware.CriticalRateLimit()` to blunt key-enumeration attempts.
- The endpoint reports the key's *effective* status computed at query time (enabled / disabled / expired / exhausted) without writing to the database, so a disabled or exhausted key still returns a readable report instead of an auth error.
- The endpoint resolves the key's *effective* group: the token's own group when set, otherwise the owning user's group.
- The response deliberately excludes all account identity fields (user id, username, email) — the person holding a key is not necessarily the account owner.
- `GET /api/usage/token` keeps its current contract; no field is removed or renamed.
- The same page shows a **model status** section — one row per model with an Operational / Down / No Data badge and a strip of recent-interval bars — sourced from the existing performance metrics summary, and narrowed to the checked key's group once a key has been checked.
- The same page shows a **setup** section that turns the checked key into a one-line command for Claude Code, Codex, OpenCode, Pi or Oh My Pi, on Windows or macOS/Linux. A new endpoint returns the setup script as plain text; the script writes the CLI's own configuration file rather than exporting environment variables.
- Model choices offered in the setup section come only from the models the key's group enables, and the script endpoint re-validates every requested model against that group server-side.

## Capabilities

### New Capabilities
- `public-key-check`: Public, unauthenticated lookup of an API key's own usage and configuration — the `POST /api/token/check` contract plus the `/key` page behavior (form validation, result presentation, loading / error / empty states, accessibility).
- `public-model-status`: The model health section on the public key page — its data source, the Operational / Down / No Data classification, the recent-interval bar strip, and its narrowing to the checked key's group.
- `public-setup-script`: The setup section and the script endpoint behind it — supported applications and operating systems, group-scoped model selection, server-side model validation, and the configuration files each generated script writes.

### Modified Capabilities
<!-- None. GET /api/usage/token keeps its existing contract; no existing spec's requirements change. -->

## Impact

- **Backend**: `controller/token.go` (key check handler + effective-status/group resolution), a new setup-script controller and its per-application script templates, `router/api-router.go` (two new public routes), `i18n/` message catalogs (en, vi) for the new error messages.
- **Frontend**: new route `web/src/routes/key/index.tsx`, new feature `web/src/features/key-check/` covering all three sections, additive keys in `web/src/i18n/locales/en.json` and `vi.json`.
- **Reused, unmodified**: `PublicLayout`, `StatusBadge`, `API_KEY_STATUSES` (`web/src/features/keys/constants.ts`), `formatQuota` (`web/src/lib/format.ts`), the shared `api` axios instance, `GET /api/perf-metrics/summary` (`controller.GetPerfMetricsSummary`), `model.GetGroupEnabledModels` (`model/ability.go:43`), `system_setting.ServerAddress`.
- **Not touched**: database schema (no migration, so the three-database verification matrix does not apply), public navigation/header components and `layout/types.ts` (in-progress work from another change), backend-configured nav modules.
- **Security surface**: one new unauthenticated endpoint. Mitigations: key in request body only, `CriticalRateLimit`, generic "invalid key" message that does not distinguish "not found" from other rejections, and no account identity in the response.
