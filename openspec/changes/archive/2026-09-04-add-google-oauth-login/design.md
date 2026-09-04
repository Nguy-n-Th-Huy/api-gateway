## Context

See `proposal.md` — Why. The OAuth layer is already a registry of providers behind one interface: `oauth.Provider` (`oauth/provider.go`) with eleven methods, providers self-registering from `init()` (`oauth/registry.go`), and a single generic route `GET /api/oauth/:provider` dispatching to `controller.HandleOAuth` (`router/api-router.go:56`). GitHub (`oauth/github.go`, 183 lines), Discord and LinuxDO are the reference implementations; OIDC (`oauth/oidc.go`) is the reference for a provider that needs a `redirect_uri` on the token exchange.

Provider identifiers are stored as indexed columns on the users table — `github_id`, `discord_id`, `oidc_id`, `wechat_id`, `telegram_id`, `linux_do_id` (`model/user.go:87-105`) — and reached through per-provider `FillUserBy…Id` / `Is…IdAlreadyTaken` helpers. Custom OAuth providers instead persist to `user_oauth_bindings`; that path is not used here, because the ask is for a first-class provider.

Operator settings follow a fixed shape: a `common.*` package variable, an entry in `model/option.go` (default map, boolean/string dispatch), and a public flag on `controller.GetStatus`.

## Goals / Non-Goals

**Goals:**
- Add Google by following the existing provider shape exactly, so it behaves identically to GitHub everywhere the two are listed side by side.
- Keep the change additive: no existing provider, route, or stored binding changes behavior.
- Give the three-database migration requirement first-class treatment in the task list, because a new indexed column is exactly the kind of change `AGENTS.md` requires be proven on real engines.

**Non-Goals:**
- No change to the generic OIDC provider or the custom OAuth system, even though either could approximate Google today.
- No Google API access beyond identity — no Drive, Gmail, or Workspace directory scopes.
- No account-linking-by-email. Matching a Google identity to an existing account by shared email address is a known account-takeover vector when the upstream email is unverified, and no existing provider does it.

## Decisions

**Identify the account by Google's `sub` claim, never by email.** Google guarantees `sub` is stable and unique per account; email addresses on Workspace domains can be reassigned to a different person after an employee leaves. This mirrors the GitHub provider, which deliberately keys on the numeric ID and keeps `login` only as a legacy migration hint (`oauth/github.go:153-159`). Alternative considered: keying on email, rejected as a takeover vector.

**Fetch the profile from Google's userinfo endpoint rather than decoding the ID token locally.** Decoding an ID token correctly means fetching and caching Google's JWKS, verifying the signature, issuer, audience and expiry — real work that the codebase has no existing helper for. The userinfo endpoint returns the same `sub`, `email`, `name`, `picture` over a channel already authenticated by the access token we just obtained. Alternative considered: adding a JWT verification dependency, rejected as unnecessary surface for the benefit gained.

**Request the minimum scopes: `openid email profile`.** Enough for a stable identifier, a display name, and an email when the user permits it. No offline access and no refresh token: the gateway authenticates the visitor once and issues its own session, so a long-lived Google credential would be a stored secret with no use.

**Derive the `redirect_uri` from the configured server address, following the OIDC provider.** `oauth/oidc.go` builds `{ServerAddress}/oauth/oidc`; Google becomes `{ServerAddress}/oauth/google`. Google validates `redirect_uri` strictly against the value registered in the Cloud console, so this must be a single deterministic string, not something derived per-request from the incoming `Host` header — a `Host`-derived value would break the moment the app sits behind a proxy.

**Store the identifier in a new `google_id` column on users, not in `user_oauth_bindings`.** Every first-class provider uses a column; `ProviderUserIDColumn()` exists on the interface precisely so bind flows can update one column instead of writing a full user snapshot. Using the bindings table instead would make Google the only first-class provider that behaves like a custom one.

**Report the GitHub audit rather than fixing it in this change.** The user asked for a review of that flow. Mixing opportunistic fixes to a working, shipped authentication path into a change whose purpose is to add a provider makes both harder to review and harder to roll back. Findings are reported for a decision.

## Risks / Trade-offs

- **A new indexed column touches all three supported databases** → the task list carries explicit verification on real SQLite, MySQL and PostgreSQL, fresh and upgraded, with the migration run twice for idempotency, and the exact engine versions recorded. `AGENTS.md` does not accept a build, unit tests, or a single-dialect check as a substitute.
- **Google returns no email when the user withholds the scope, or for some Workspace configurations** → account creation must not depend on the email field; the spec makes that an explicit scenario.
- **Username derived from a Google display name can collide, and display names are not unique at all** → creation uses the existing collision-avoidance path with the `google_` prefix, the same as every other provider.
- **An operator who already configured Google through the OIDC slot or a custom provider will now see two Google paths** → existing bindings are untouched and keep working; the two are separate identities from the system's point of view, and a user bound through the old path stays bound there. This is a documentation matter, called out in the tasks, not a migration.
- **A misconfigured `ServerAddress` produces a `redirect_uri` Google rejects** → the failure is loud and immediate at the Google consent screen rather than silent, and the same constraint already governs the OIDC provider.

## Migration Plan

One additive column via `AutoMigrate`, no data backfill, no contract change. Rollback is a revert of the code; the `google_id` column left behind on a rolled-back instance is inert — no code reads it and no constraint depends on it. Operators who do not configure Google see no change at all: the flag defaults off and the sign-in page renders exactly as before.
