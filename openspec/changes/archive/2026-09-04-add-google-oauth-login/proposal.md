## Why

Google is the single most common consumer identity provider, and the gateway currently offers no way to sign up or sign in with it as a named provider. An operator can approximate it today only by pointing the generic OIDC slot or a custom OAuth entry at Google — which consumes the one OIDC slot, shows a generic label, and stores the binding outside the users table like every other custom provider. Google deserves the same first-class treatment GitHub, Discord, LinuxDO, WeChat and Telegram already have.

A companion audit of the existing GitHub sign-up flow is included: GitHub support is already implemented end to end, so the work there is to confirm the flow is sound rather than to build it.

## What Changes

- Add a Google OAuth provider registered under the slug `google`, implementing the existing `oauth.Provider` interface. The generic `GET /api/oauth/:provider` route already dispatches through the registry, so no new route is introduced.
- Add a `google_id` column to the users table, indexed like the other provider ID columns, so a Google identity binds to an account the same way a GitHub identity does.
- Add the operator settings that turn Google sign-in on and hold its client credentials, alongside the existing GitHub and Discord settings.
- Publish a `google_oauth` flag on the public status payload so the sign-in page can show the Google action only when an operator has enabled it.
- Add the Google action to the sign-in and sign-up pages, the account-bindings tab, and the admin authentication settings section, matching the existing provider treatments.
- Audit the existing GitHub sign-up flow and report findings: whether a first-time GitHub identity actually creates an account, how a missing or private email is handled, what happens on username collision, and whether the numeric-ID migration path still holds. Report only — any defect found is raised for a decision, not silently fixed.
- **Out of scope**: no change to the generic OIDC provider, the custom OAuth system, or any other existing provider. No change to session, JWT, or password handling.

## Capabilities

### New Capabilities
- `auth/google-oauth`: signing up and signing in with a Google account, binding a Google identity to an existing account, and the operator configuration that governs it.

### Modified Capabilities
<!-- None. No existing spec covers OAuth provider behavior in openspec/specs/. -->

## Impact

- `oauth/google.go` (new) — provider implementation and registry registration.
- `model/user.go` — `GoogleId` field, lookup and taken-check helpers mirroring the GitHub ones.
- **Database**: a new `google_id` column reaches all three supported engines through `AutoMigrate`. Per `AGENTS.md` this requires verification against real SQLite, MySQL and PostgreSQL instances, on a fresh database and on one upgraded from the previous release, with migration run twice to prove idempotency.
- `common/constants.go`, `model/option.go` — the enable flag and client credential options.
- `controller/misc.go` — the `google_oauth` status flag.
- `web/src/features/auth/` — provider list, types, sign-in and sign-up actions.
- `web/src/features/profile/components/tabs/account-bindings-tab.tsx` — bind entry and bound-state display.
- `web/src/features/system-settings/auth/oauth-section.tsx` — operator configuration.
- `web/src/i18n/locales/*.json` — new keys, Vietnamese translated, other locales synced.
- `i18n/` (backend) — any new OAuth error message.
