## Why

Two defects in the OAuth account layer can each cost a user their account, and both were found while auditing the GitHub sign-up flow.

The first is an account-takeover path. When a GitHub identity is not recognized by its numeric ID, the sign-in flow falls back to matching the visitor's **current** GitHub login against the stored identifier (`controller/oauth.go:309-311`, fed by `legacy_id` in `oauth/github.go:158`). GitHub does not reserve a login after its owner renames the account — the name becomes available for anyone to register. So a user who linked under the login `alice` and later renamed their GitHub account can have that login claimed by a stranger, who then signs in and lands directly inside the original user's gateway account, with its balance and its API keys.

The second is a lockout path. `model.ClearBinding` (`model/user.go:869`) blanks whichever binding column an administrator names — including `email` — with no check that the account keeps any means of signing in. Clearing the last one leaves a real account permanently unreachable, and nothing warns the administrator before it happens.

The change also completes the Google provider shipped in `add-google-oauth-login`, whose sign-in action currently renders without a brand mark because no Google icon existed in the asset set.

## What Changes

- **BREAKING for one narrow case**: legacy GitHub identifiers are no longer accepted on login-name match alone. A legacy match SHALL additionally require the email GitHub returns to equal the email already stored on the matched account. A match that fails this check is refused with an explanation instead of signing anyone in — it neither adopts the account nor silently creates a second one.
- Refuse to clear a binding when doing so would leave the account with no password and no other binding, and report why. This applies to every binding type, including email.
- Record an audit entry whenever a legacy GitHub identifier is migrated, so an operator can see the identity mapping change after the fact.
- Add a Google brand mark and use it on the Google sign-in action, replacing the current text-only treatment.
- **Out of scope**: no self-service unbind flow is introduced — clearing a binding stays administrator-only, as it is for every built-in provider. No change to any other provider's identity matching. No change to session, JWT, or password handling.

## Capabilities

### New Capabilities
- `auth/account-binding-safety`: the rules protecting an account when an external identity is matched, migrated, or removed.

### Modified Capabilities
<!-- None. The auth/google-oauth spec covers Google's own behavior; nothing in it changes here — the icon is presentation, not behavior. -->

## Impact

- `controller/oauth.go` — the legacy-identifier branch gains the email check and the refusal path.
- `model/user.go` — `ClearBinding` gains the last-sign-in-method guard.
- `controller/user.go` — surfaces the refusal to the administrator.
- `i18n/` (backend) — new messages for the two refusals, en and zh.
- `web/src/assets/brand-icons/` — a Google mark, following the existing icon convention.
- `web/src/features/auth/components/oauth-providers.tsx` and the sign-in form — render the mark.
- `web/src/i18n/locales/*.json` — any new user-facing string.
- **No database change**: no column is added, altered, or dropped, so the three-engine verification matrix does not apply.
- Existing GitHub users who already migrated to a numeric identifier are unaffected — they never reach the legacy branch.
