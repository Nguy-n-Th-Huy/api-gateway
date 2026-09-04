## Context

See `proposal.md` — Why, for both defects and their exact locations.

Relevant existing shape: `findOrCreateOAuthUser` (`controller/oauth.go:291`) tries the stable identifier first, then the `legacy_id` carried in `oauth.OAuthUser.Extra`, then falls through to creating an account. Only the GitHub provider populates `legacy_id` today (`oauth/github.go:158`), but the branch is written against the generic `oauth.Provider` interface, so any future provider that sets it inherits the same behavior.

`model.ClearBinding` (`model/user.go:869`) maps a binding type to a users-table column and blanks it inside a transaction. Its only production caller is `AdminClearUserBinding` (`controller/user.go:773`), reached from an administrator-only route. `User.Password` is a plain column (`model/user.go:82`); an account with no password holds the empty string, the same check `model/user.go:591` already makes.

## Goals / Non-Goals

**Goals:**
- Close the takeover path without stranding the legitimate legacy users the path exists to serve.
- Make the lockout impossible at the model layer, where every present and future caller inherits the guard, rather than only in the one controller that calls it today.
- Keep both fixes free of schema change, so no database verification matrix is triggered.

**Non-Goals:**
- No self-service unbind. Adding one is a separate decision affecting all built-in providers.
- No change to how the stable-identifier path matches, which is already correct.
- No attempt to recover accounts already taken over by this path, if any exist. Detecting that reliably after the fact is not possible from the data the system stores; the audit entry this change adds only helps from now on.

## Decisions

**Corroborate a legacy match with the stored email, and refuse rather than fall through when it fails.** The three candidate behaviors on a failed corroboration were: sign in anyway (the status quo, which is the vulnerability), create a new account (silently splits a real user's identity in two and hides the anomaly), or refuse. Refusing is the only one that neither hands an account to a stranger nor quietly manufactures a duplicate. It costs a legitimate user with no stored email one support request; it costs an attacker the entire attack.

**Compare the email exactly, against the address already stored on the account.** Not a fuzzy or domain-level comparison, and not against the email the provider merely asserts is verified — the system's own record is the trustworthy half of the comparison. `model.NormalizeEmail` already exists and is applied on write (`model/user.go:585`), so both sides are compared in normalized form.

**Put the last-sign-in-method guard inside `ClearBinding`, not in the controller.** The controller is today's only caller of *that* function, but the function is exported and the mapping table inside it already enumerates every binding type, so a guard there protects every future caller of it — a bulk admin tool, a migration script. Alternative considered: guarding in `AdminClearUserBinding`, rejected because it protects exactly one call site.

This guard does **not** reach custom OAuth bindings. Those are removed by a different function, `model.DeleteUserOAuthBinding`, reached from the pre-existing self-service route `controller.UnbindCustomOAuth` (`router/api-router.go:120`), and they live in `user_oauth_bindings` rather than in a users-table column. So two gaps remain open after this change: a user whose only way in is a custom OAuth binding can still strand themselves through that route, and `accountHasAnotherSignInMethod` cannot see a custom binding, which makes it over-refuse — it will block clearing a built-in binding for an account whose only *other* method is a custom one. The over-refusal is safe in direction and merely confusing; the self-service gap is a real hole. Both are pre-existing and neither is closed here: fixing them means changing the custom OAuth path, which is outside a change whose stated non-goal is introducing no self-service unbind behavior. They are recorded for a follow-up.

**Count the email address as a sign-in method.** An account with only an email can still get in through the email verification flow, so clearing a provider binding while an email remains is safe. The reverse also holds: clearing the email itself when nothing else remains must be refused, which is why the guard is expressed over the full set rather than over provider bindings alone.

**Compute the guard from the account's persisted state, inside the same transaction as the clear.** Reading the row and deciding outside the transaction would leave a window in which a concurrent clear removes the other binding. The check and the write belong together.

**Ship the Google mark as an SVG asset alongside the existing brand icons.** The provider list already resolves marks from that asset set; the Google button is the only enabled provider rendering without one. This is presentation only — no requirement in `auth/google-oauth` changes, which is why it appears in the tasks and the proposal but produces no spec delta.

## Risks / Trade-offs

- **A legitimate legacy user with no stored email is now refused where they previously signed in** → they see an explanation directing them to support, and an administrator can resolve it by clearing and re-binding. This is the deliberate cost of closing the takeover path; the alternative is leaving every renamed-account user exposed.
- **The guard changes an administrator-facing behavior that previously always succeeded** → the refusal is explicit and explains itself, and an administrator who genuinely wants the account gone still has account deletion, which is the correct tool for that intent.
- **Counting email as a sign-in method assumes the email verification flow stays enabled** → if an operator disables email login entirely, an account holding only an email is already unreachable today; this change neither creates nor worsens that, and the guard still prevents the strictly worse outcome of clearing the last provider binding.
- **No test can prove the takeover is closed in production** → the regression tests assert the refusal on a mismatched email and on a missing email, which are the two conditions the attack needs.
- **The corroboration is only as strong as the provider's email verification, and nothing enforces that** → `oauth.OAuthUser` carries no verified flag. For GitHub the value is the public profile email, which GitHub requires to be verified, so the check holds today. But the legacy branch is written against the generic `oauth.Provider` interface, so a future provider populating `legacy_id` alongside an unverified email would silently reopen the takeover path. Any provider that sets `legacy_id` must supply a verified address.

## Migration Plan

Code-only, no schema change, no data backfill. Rollback is a revert. Accounts already migrated to stable identifiers never enter the changed branch, so the overwhelming majority of users see no difference at all.
