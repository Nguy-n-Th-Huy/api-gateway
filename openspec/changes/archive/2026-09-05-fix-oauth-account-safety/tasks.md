## 1. Close the legacy-identifier takeover path

- [x] 1.1 In `findOrCreateOAuthUser` (`controller/oauth.go:308-326`), after a legacy identifier matches an account, require the provider-reported email to equal the email stored on that account, both normalized with the existing `model.NormalizeEmail`.
- [x] 1.2 On mismatch, refuse the sign-in with a dedicated error. Do NOT sign in, do NOT adopt the account, and do NOT fall through to account creation. ← (verify: the matched account is byte-for-byte unchanged afterwards)
- [x] 1.3 On either side having no email, refuse the same way with a message directing the visitor to support.
- [x] 1.4 Keep the stable-identifier path ahead of this branch untouched — an already-migrated user must never reach it. ← (verify)
- [x] 1.5 Add the two refusal messages to `i18n/` for en and zh, following the existing OAuth message convention.

## 2. Make identifier migration auditable

- [x] 2.1 Record an audit entry on a successful legacy-to-stable migration naming the account, the old identifier and the new one.
- [x] 2.2 Keep the existing failure logging on `UpdateGitHubId` and confirm it is recorded as an error, not swallowed silently.

## 3. Guard against stranding an account

- [x] 3.1 In `model.ClearBinding` (`model/user.go:869`), before clearing, determine whether the account would retain any means of signing in: a non-empty password, or any remaining binding column, counting `email`.
- [x] 3.2 Refuse with a dedicated error when nothing would remain, leaving the account untouched. ← (verify: no column is written on the refusal path)
- [x] 3.3 Perform the check inside the same transaction as the clear, so a concurrent clear cannot slip between the check and the write. ← (verify)
- [x] 3.4 Surface the refusal through `AdminClearUserBinding` (`controller/user.go:773`) so the administrator sees why, not a generic failure.
- [x] 3.5 Add the refusal message to `i18n/` for en and zh.
- [x] 3.6 Confirm the guard uses GORM query methods, introduces no raw SQL, and adds no column — this change must trigger no schema migration. ← (verify)

## 4. Google brand mark

- [x] 4.1 Add a Google mark to `web/src/assets/brand-icons/`, matching the file shape, export style and sizing of the existing marks in that directory.
- [x] 4.2 Render it on the Google sign-in action in `web/src/features/auth/components/oauth-providers.tsx` and wherever the sign-in form renders provider marks, replacing the text-only treatment.
- [x] 4.3 Mark the icon `aria-hidden` — it is decorative, the action already carries its own accessible name.

## 5. Tests

- [x] 5.1 Go test: a legacy match with an email equal to the stored one migrates the account and signs in.
- [x] 5.2 Go test: a legacy match with a different email is refused, no account is created, and the matched account is unchanged. This is the takeover regression test.
- [x] 5.3 Go test: a legacy match with no provider email, and one with no stored email, are both refused.
- [x] 5.4 Go test: clearing a binding succeeds while a password remains, succeeds while another binding remains, and is refused when neither does.
- [x] 5.5 Go test: clearing the email is refused when it is the only remaining method.
- [x] 5.6 Use `stretchr/testify` `require`/`assert` with explicit table inputs, per the backend test rules in `AGENTS.md`.

## 6. Verification

- [x] 6.1 `go build ./...` clean. ← (verify)
- [x] 6.2 `go test ./controller/... ./oauth/... ./model/...` pass. ← (verify)
- [x] 6.3 `cd relaykit && GOWORK=off go build ./...` clean. ← (verify)
- [x] 6.4 `cd web && bun run typecheck`, scoped lint on the touched files, and `bun run test` pass. ← (verify)
- [x] 6.5 Confirm no schema change: no new column, no altered column, no `AutoMigrate` entry added. ← (verify)
- [x] 6.6 Confirm no protected identifier (project or organization name) was altered anywhere in the diff. ← (verify)
- [x] 6.7 Confirm no self-service unbind route or button was introduced — removal stays administrator-only. ← (verify)
