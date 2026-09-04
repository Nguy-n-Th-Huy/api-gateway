## 1. Backend provider

- [x] 1.1 Add `oauth/google.go`: `GoogleProvider` implementing every method of `oauth.Provider`, registered as `google` from `init()`. Follow `oauth/github.go` for shape and `oauth/oidc.go` for the `redirect_uri` handling.
- [x] 1.2 Exchange the authorization code at Google's token endpoint with `redirect_uri` = `{ServerAddress}/oauth/google` and scopes `openid email profile`. Return an OAuth error naming Google on failure; never surface the client secret or a raw upstream body. ← (verify: error paths leak nothing)
- [x] 1.3 Fetch the profile from Google's userinfo endpoint and map `sub` → provider user ID, plus name and email. Reject a response with an empty `sub`.
- [x] 1.4 Implement `IsUserIDTaken`, `FillUserByProviderID`, `SetProviderUserID`, `GetProviderPrefix` (`google_`) and `ProviderUserIDColumn` (`google_id`).

## 2. Data model

- [x] 2.1 Add `GoogleId` to `model.User` with `gorm:"column:google_id;index"`, placed alongside the other provider ID fields.
- [x] 2.2 Add `FillUserByGoogleId` and `IsGoogleIdAlreadyTaken` in `model/user.go`, mirroring the GitHub helpers exactly.
- [x] 2.3 Confirm no raw SQL, no dialect-specific type, and no boolean default tag is introduced. ← (verify)

## 3. Configuration

- [x] 3.1 Add `GoogleOAuthEnabled`, `GoogleClientId`, `GoogleClientSecret` to `common/constants.go`.
- [x] 3.2 Wire them into `model/option.go`: default map entries plus the boolean and string dispatch cases.
- [x] 3.3 Publish `google_oauth` on the status payload in `controller/misc.go`. Confirm no credential is published. ← (verify: fetch the payload and grep for the secret)

## 4. Frontend

- [x] 4.1 Add Google to the auth provider list, types and helpers under `web/src/features/auth/` so the action renders only when `google_oauth` is true.
- [x] 4.2 Add the Google action to the sign-in and sign-up pages, matching the existing GitHub treatment.
- [x] 4.3 Add Google to `web/src/features/profile/components/tabs/account-bindings-tab.tsx` with bind and unbind.
- [x] 4.4 Add the Google configuration block to `web/src/features/system-settings/auth/oauth-section.tsx` (enable toggle, client ID, client secret, and the callback URL the operator must register with Google).
- [x] 4.5 Confirm no hardcoded color and no hardcoded user-facing string; every label goes through `t('...')`.

## 5. Internationalization

- [x] 5.1 Add Vietnamese translations for every new key to `web/src/i18n/locales/vi.json`.
- [x] 5.2 Run `cd web && bun run i18n:sync`; confirm all seven locales carry the new keys. ← (verify)
- [x] 5.3 Add any new backend OAuth message to `i18n/` for both en and zh. (No new backend message keys were introduced — Google reuses the existing generic `oauth.*` templated keys with `Provider: "Google"`, already present in `i18n/locales/en.yaml` and `zh-CN.yaml`.)

## 6. Tests

- [x] 6.1 Add Go tests for the provider: profile with no email still yields a valid `OAuthUser`; empty `sub` is rejected; the token-exchange error path returns an error naming Google and carries no secret.
- [x] 6.2 Add a test that `google_id` is the column reported by `ProviderUserIDColumn` and that the taken-check and fill helpers agree on it.
- [x] 6.3 Use `stretchr/testify` `require`/`assert` with explicit table inputs, per the backend test rules.

## 7. Database verification — mandatory, all three engines

- [x] 7.1 Back up any database before running a migration against it. ← (verify; only throwaway containers with no pre-existing data were used, so no backup was required — see report)
- [x] 7.2 Fresh SQLite: start the app against an empty database, confirm the `google_id` column and its index exist. ← (verify)
- [x] 7.3 Fresh MySQL (>= 5.7.8) and fresh PostgreSQL (>= 9.6): same check on each. Use throwaway containers; do not touch any database belonging to another project on this machine. ← (verify)
- [x] 7.4 Upgrade path: on each of the three engines, start from a database created by the previous release, run the migration, and confirm existing rows, indexes and uniqueness guarantees survive. ← (verify)
- [x] 7.5 Idempotency: start twice in a row on each engine; the second start must make no further schema change and must not fail. ← (verify)
- [x] 7.6 Record the exact engine versions, commands, and results in the final report. If any engine cannot be exercised, say so explicitly and do not claim database compatibility. ← (verify)

Recorded per `AGENTS.md`. Method: a temporary Go harness called `model.InitDB()` directly against each DSN; the "previous release" schema came from a `git worktree` at commit `fc4046fd8`. Throwaway containers `napi-dbverify-mysql` / `napi-dbverify-pg` were used and removed afterwards; no shared or third-party database was touched.

| Engine | Version | Fresh (column + index) | Idempotent (2nd start, no DDL) | Upgrade (prior-release DB) |
|---|---|---|---|---|
| SQLite | glebarez/sqlite v1.9.0 (modernc.org/sqlite v1.40.1) | pass | pass | pass — `github_id` retained, `google_id` added empty |
| MySQL | 8.0.46 | pass | pass — no DDL logged on rerun | pass — `CREATE INDEX idx_users_google_id ON users(google_id)` logged once only |
| PostgreSQL | 15.19 | pass | pass | pass — existing rows and indexes preserved |

No backup was required: every migration ran against a throwaway container holding no pre-existing data.

## 8. GitHub flow audit — report only, no fixes

- [x] 8.1 Trace the GitHub sign-up path end to end and report: does a first-time GitHub identity actually create an account, and is that gated on registration being open? ← see final report
- [x] 8.2 Report how a missing or private GitHub email is handled during account creation. ← see final report
- [x] 8.3 Report what happens when the derived username collides with an existing account. ← see final report
- [x] 8.4 Report whether the login-to-numeric-ID migration path (`legacy_id` in `oauth/github.go`) still works for accounts created before it. ← see final report
- [x] 8.5 Report whether unbinding GitHub can leave an account with no way to sign in. ← see final report
- [x] 8.6 Raise every finding for a decision. Do not change GitHub behavior in this change. ← (verify: the diff touches no GitHub logic — confirmed, see report)

## 9. Verification

- [x] 9.1 `go build ./...` clean. ← (verify)
- [x] 9.2 `go test ./oauth/... ./model/...` pass. ← (verify)
- [x] 9.3 `cd relaykit && GOWORK=off go build ./...` clean — required by `AGENTS.md` whenever the module graph could be affected. ← (verify)
- [x] 9.4 `cd web && bun run typecheck`, `bun run lint`, `bun run test` pass. ← (verify; `bun run lint` and `bun run test` have pre-existing, unrelated failures in files this change never touched — see report)
- [x] 9.5 Confirm every new file carries the AGPL copyright header used by its neighbours. ← (verify; `oauth/*.go` neighbours carry no header, so `oauth/google.go`/`oauth/google_test.go` carry none either — consistent. `model/google_id_test.go` matches its package's no-header convention. New frontend files carry the header.)
- [x] 9.6 Confirm no protected identifier (project or organization name) was altered anywhere in the diff. ← (verify)
