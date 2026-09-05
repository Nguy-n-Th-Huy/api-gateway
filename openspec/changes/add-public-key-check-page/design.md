## Context

See proposal.md — Why.

Constraints that shape the approach:

- `controller/token.go:226` already has `GetTokenUsage`, which parses `Authorization` itself and looks the token up by key. It is mounted at `GET /api/usage/token` behind `middleware.TokenAuthReadOnly()` (`router/api-router.go:246-253`). That middleware rejects `common.TokenStatusDisabled` with `401`, which is exactly the case the new page must be able to display.
- `middleware.TokenAuthReadOnly` is also used by `GET /api/log/token` (`router/api-router.go:301`), so its rejection rules cannot be relaxed without changing that endpoint's security posture.
- `model.Token` (`model/token.go:15-33`) carries `Group`, `Status`, `CreatedTime`, `AccessedTime`; an empty `Group` means "inherit the owner's group". `model.GetUserGroup(id, fromDB)` (`model/user.go:1260`) resolves the owner's group with a Redis-first path.
- `model.ValidateUserToken` recomputes expired/exhausted status but *writes it back* to the database when Redis is off. The check endpoint must not do that.
- AGENTS.md forbids `encoding/json` marshal/unmarshal in business code (use `common.*`), requires localized API messages via `i18n/`, and requires the three-database verification matrix only for schema/DB-behavior changes — this change has none.
- The frontend is TanStack Router file-based (`web/src/routes/`), with public pages composed from `PublicLayout` (`web/src/features/about/index.tsx` is the closest precedent). `API_KEY_STATUSES` + `StatusBadge` and `formatQuota` already exist and encode the project's status vocabulary and quota display rules.
- `web/src/components/layout/components/public-navigation.tsx`, `public-header.tsx` and `layout/types.ts` currently carry uncommitted work from a separate nav-grouping change; touching them would collide.

## Goals / Non-Goals

**Goals:**

- One new public endpoint whose failure modes are all representable in the UI, including disabled keys.
- Status and group resolution that is read-only and identical to what the relay would apply, so the page never contradicts the console.
- A page that is complete on its own: validation, four UI states, full report, localization, keyboard and screen-reader support.

**Non-Goals:**

- Any change to `GET /api/usage/token`'s observable contract, its middleware, or `TokenAuthReadOnly` itself.
- Discoverability work (public nav entry, backend nav module) — the page is reached by its URL for now.
- Showing per-request logs or spend history for the key; this is a snapshot of the key's own record.
- Any database schema, migration, or GORM tag change.

## Decisions

### Separate `POST /api/token/check` instead of loosening `GET /api/usage/token`

Chosen because the disabled-key case is a first-class result for this feature but an auth failure for every existing consumer of `TokenAuthReadOnly`.

Alternatives considered:

- *Relax `TokenAuthReadOnly` to allow disabled tokens.* Rejected: it would also open `GET /api/log/token` to disabled keys — a security-posture change nobody asked for.
- *Add a second route pointing at `GetTokenUsage` without the middleware.* Rejected: `GetTokenUsage`'s response shape and its `Authorization`-header contract are consumed by existing clients; overloading it with new semantics (report a disabled key rather than reject it) makes one handler answer to two contracts.

`GetTokenUsage` stays exactly as it is. The two handlers share the report-building logic rather than duplicating field names.

### Key travels in the JSON body, over `POST`

A `GET /api/token/check?key=…` would put a live credential into access logs, proxy logs, browser history and `Referer` headers. `POST` with the key in the body keeps it out of all of those. The endpoint is idempotent and read-only despite being a `POST`; that is a deliberate trade of REST purity for credential hygiene, and it matches how `POST /api/token/:id/key` already handles key material in this codebase.

### Effective status is computed, never persisted

The handler derives status in a pure function of the token row plus the current timestamp, in the fixed order disabled → expired → exhausted → enabled, and returns it. Unlike `ValidateUserToken`, it performs no `SelectUpdate`. Reasons: an unauthenticated endpoint must not be able to trigger writes (a write path reachable without credentials is an amplification target), and the reported value must not depend on whether Redis happens to be enabled.

The pure function is the natural unit for the required backend test — it encodes a real business rule (which state wins when several apply) rather than an implementation detail.

### Effective group resolved server-side

Returning the token's raw empty group would be a lie to the user, who would see "no group" for a key that actually bills at the owner's group rate. The handler resolves the empty case through `model.GetUserGroup`. If that lookup fails the request reports the generic error rather than guessing a group.

### Generic rejection for unknown keys

Every unmatched key gets one identical localized message and status. Distinguishing "never existed" from "deleted" or "belongs to a banned user" would turn the endpoint into an oracle. The rate limit narrows the remaining brute-force surface; combined, an attacker learns only "valid/invalid", which is unavoidable for a feature whose entire purpose is answering that question for the legitimate key holder.

### Reuse the console's status and quota vocabulary on the page

The page imports `API_KEY_STATUSES` and `StatusBadge` and formats quota with `formatQuota`. A key's status must read identically on `/key` and in the console; a second copy of the status labels would drift. This is a deliberate cross-feature import (`@/features/key-check` → `@/features/keys/constants`); the alternative — lifting the constants to `web/src/components/` — would touch the keys feature and widen the diff for no behavioral gain.

### No navigation entry

The header nav is driven by backend-configured modules (`web/src/lib/nav-modules.ts`) and its components are mid-edit in another change. Adding an entry would require a new backend nav module plus edits to files owned by that change. The page works standalone at `/key`, which is what was asked for; a nav entry can follow as its own change.

### Model status reuses `GET /api/perf-metrics/summary`

That endpoint (`router/api-router.go:36`, `controller.GetPerfMetricsSummary`) already returns, per model, `success_rate`, `recent_success_rates[]` and `request_count` over a trailing window — exactly the badge and the bar strip. A second health computation would be a second source of truth that could contradict the pricing page's own uptime sparkline, which reads the same endpoint.

Its middleware is `HeaderNavModulePublicOrUserAuth("pricing")`, so it is public whenever the pricing module is public. When an operator has made pricing require sign-in, the metrics call fails for an anonymous visitor and the model status section shows its error state; the key check and setup sections keep working. That is the accepted behavior — a new always-public metrics route would quietly override an operator's deliberate decision to hide model data.

The Operational / Down threshold is 90 percent success rate, and the bar colours use 90 / 50 percent boundaries. These are presentation thresholds chosen here because the endpoint reports a rate, not a state; they live in one constant so an operator-facing setting can replace them later without touching the rendering.

### Setup script is served as plain text over `GET`, with the key in the query string

The copy-paste one-liner (`irm "…" | iex`, `curl -fsSL "…" | sh`) can only issue a `GET`, and the URL is the only place the key can travel. This contradicts the key-check endpoint's rule of never putting a key in a URL, and the contradiction is deliberate and confined to this endpoint: without it the feature cannot exist in the form requested.

Mitigations: `middleware.CriticalRateLimit()`; the handler logs no query string; the endpoint returns only a script, never key metadata; and the page displays the command with the key masked while the clipboard receives the working command, so a shared screenshot does not leak the key. Alternatives considered — a short-lived one-time script token, or a `POST` plus a two-step "download then run" — were rejected for this change: the first adds a token store and expiry semantics for marginal gain over the rate limit, and the second breaks the single-line paste the feature is built around. The one-time-token route stays open as a follow-up if key-in-URL proves to be a real problem in operation.

### Scripts write config files; environment variables only where a CLI leaves no choice

Every CLI here is configured through a file, and the file is what the scripts write:

| App | File (Windows form) | Base URL key | Credential key | Model slots |
|---|---|---|---|---|
| Claude Code | `~/.claude/settings.json` (`%USERPROFILE%\.claude\settings.json`) | `env.ANTHROPIC_BASE_URL` | `env.ANTHROPIC_AUTH_TOKEN` | `env.ANTHROPIC_DEFAULT_OPUS_MODEL`, `…_SONNET_MODEL`, `…_HAIKU_MODEL`, `env.CLAUDE_CODE_SUBAGENT_MODEL` |
| Codex | `~/.codex/config.toml` (`%USERPROFILE%\.codex\config.toml`) | `model_providers.<id>.base_url` (must end `/v1`) | `model_providers.<id>.env_key` → an env var | `model` + one profile per slot |
| OpenCode | `~/.config/opencode/opencode.json` | `provider.<id>.options.baseURL` | `provider.<id>.options.apiKey` | `model`, `small_model` (`provider/model` form) |
| Pi | `~/.pi/agent/models.json` | `providers.<id>.baseUrl` | `providers.<id>.apiKey` | none documented |
| Oh My Pi | `~/.omp/agent/models.yml` | `providers.<id>.baseUrl` | `providers.<id>.apiKey` | none used here |

Codex is the single exception to the no-environment-variable rule: its provider table names the credential through `env_key`, and there is no documented way to put the key itself in `config.toml`. Its script therefore persists exactly that one variable, with a comment saying why. Pretending otherwise would produce a config that silently fails to authenticate.

Codex also requires `wire_api = "responses"` — the `chat` value was removed upstream, so a gateway that only speaks Chat Completions cannot drive Codex at all. This gateway registers `POST /v1/responses` through the host protocol registry (`router/task-plugin-protocol-router.go:29`, `types.RelayFormatOpenAIResponses`), so the Responses surface exists and the generated Codex config is viable.

Claude Code's own OAuth login is bound to Anthropic's endpoint, so the generated settings use `ANTHROPIC_AUTH_TOKEN` (sent as a bearer token) rather than expecting `/login` to work through the gateway.

### Pi and Oh My Pi get no model selectors

Base Pi has no documented model-role configuration — the model is chosen in-session — and Oh My Pi's role table (`default`, `smol`, `slow`, `vision`, `plan`, `designer`, `commit`, `tiny`, `task`, `advisor`) lives in a settings file whose exact top-level structure could not be verified from its documentation. Rather than guess a schema and ship a file that fails validation, their scripts register the provider with every model the key's group enables and leave role selection to the CLI. Oh My Pi's `models.yml` additionally rejects unknown root keys, so the script writes nothing outside `providers`.

This is a conscious partial delivery for those two applications, limited to model-role pre-selection; provider registration, base URL, credential and model list are complete. Adding role routing later needs only a verified schema.

### Merge, don't overwrite

Each script parses the existing configuration file, updates only the keys in the table above, and writes the result back. An unparsable existing file stops the script with the file's path rather than being replaced — these files hold user configuration the gateway did not create, and silently clobbering a hand-tuned `settings.json` would be a worse failure than not installing.

### Windows configure step is native PowerShell; POSIX keeps its Node.js dependency and says so

The first implementation ran every config-file merge (JSON, TOML, YAML) as a small embedded Node.js program on both platforms, invoked via `& node $tmp` (Windows) or `node "$tmp"` (POSIX). That silently required Node.js on Windows too, for a step the proposal never described as needing any runtime at all — a visitor who followed the shown install command and then pasted the configure one-liner could hit `node: command not found` with no warning.

The fix is an explicit, asymmetric trade-off:

- **Windows is runtime-free.** The merge is native PowerShell (5.1-compatible — `ConvertFrom-Json` has no `-AsHashtable` there, so JSON configs are edited as `PSCustomObject` graphs via `Add-Member -Force`), with its own TOML section editor for Codex and its own `providers:`-block YAML editor for Oh My Pi, ported line-for-line from the Node.js originals so both platforms accept and reject the same files. `renderPowerShellScript` no longer wraps a Node payload; it wraps this PowerShell payload directly. PowerShell itself is bundled with Windows, so this genuinely removes the dependency rather than moving it.
- **POSIX keeps Node.js** — rewriting a second, independent implementation of three file-format editors was not worth it for a change already fixing a correctness bug in the existing ones, and Node.js is a reasonable baseline for a developer machine that is about to run an AI coding CLI. But the script now checks `command -v node` before doing anything else and exits non-zero with an actionable message naming Node.js as the requirement if it is missing, so the failure is loud and immediate instead of a bare `node: command not found` after the user has already committed to the one-liner.
- **The UI discloses the POSIX dependency.** Every application's macOS/Linux install step in `web/src/features/key-check/lib/applications.ts` carries a `runtimeNoteKey` stating that the configure command needs Node.js; the Windows step never carries one.

### Codex's TOML editor accepts tabs and multi-line arrays

The original `tomlEditorJS`/`basicSanityCheck` rejected any file containing a tab character and any line that wasn't a complete header or assignment on its own — which rejects ordinary, valid TOML: tab-indented entries, and a multi-line array like `args = [\n  "-y",\n  "everything"\n]`, the shape a real `~/.codex/config.toml` `[mcp_servers.*]` entry takes. Both are common enough that Codex setup would fail outright for users with an existing MCP server configuration.

The fix: tabs are simply accepted as whitespace (removing the blanket rejection is sufficient, since every check already runs against `line.trim()`). Multi-line arrays are handled by `netBracketDelta`, which tracks the net `[`/`]` depth of a line while ignoring bracket characters inside a quoted string; `basicSanityCheck` treats a still-open array as a continuation rather than a new statement. `editToml` itself needed no change: none of the keys this change owns are ever multi-line, so a continuation line never matches an owned top-level key and is always carried through verbatim. The PowerShell TOML editor mirrors this exactly, so both platforms accept the same files.

## Risks / Trade-offs

- **Unauthenticated endpoint enables online key-validity probing** → Key in body only, `middleware.CriticalRateLimit()`, generic rejection message, no account identity in the response, and no DB write path reachable from the handler (an unauthenticated request can still trigger a Redis write: `model.GetTokenByKey` calls `cacheInitToken` on a cold cache, matching the existing `TokenAuthReadOnly` behavior).
- **`POST` used for a read** → Documented above; the alternative leaks credentials into logs. The handler stays side-effect free so retries and caching intermediaries cannot cause harm.
- **Cross-feature import of `API_KEY_STATUSES`** → Accepted to keep one source of truth for status labels; noted here so a future refactor knows the dependency exists.
- **Page is only reachable by URL** → Accepted, in scope of this change; discoverability is deferred to a follow-up change that owns the nav files.
- **Additive edits to `en.json` / `vi.json` while another change also edits them** → Restricted to appending new keys; no existing key is renamed or removed, so the merge stays mechanical.
- **`GetUserGroup` adds a lookup on the hot path of an unauthenticated endpoint** → It is Redis-first and only runs for tokens with an empty group; the rate limit bounds the call volume.
- **The key appears in the setup script URL** → Confined to that one endpoint; rate limited, query string not logged, key masked on screen while the clipboard gets the working command. A one-time script token remains the escalation path.
- **A generated script writes into a file the user owns** → Merge-only, refuse to touch an unparsable file, create missing parent directories, and never write outside the documented keys.
- **OpenCode's user-level config path on Windows is not documented upstream** → The scripts use the XDG-style `%USERPROFILE%\.config\opencode\opencode.json`, which OpenCode's maintainers confirm is supported even though the docs omit it. If it proves wrong in the field the fix is one path constant, and the project-level `./opencode.json` is the documented fallback.
- **Pi's static `models.json` schema is corroborated rather than quoted from its docs** → Field names match Pi's documented provider-registration signature; if the file shape turns out to differ, only Pi's template changes.
- **Model status disappears when the pricing module is set to require sign-in** → Accepted; the section degrades to its error state and the rest of the page is unaffected.
- **Health thresholds (90 / 50 percent) are chosen here, not configured** → Kept in one constant so they can become a setting without touching rendering.

## Migration Plan

No schema change, no data backfill, no feature flag. Deploy is a normal build: new route registers on startup, new page ships in the frontend bundle. Rollback is reverting the change — no persisted state depends on it, and `GET /api/usage/token` is untouched, so existing clients are unaffected either way.
