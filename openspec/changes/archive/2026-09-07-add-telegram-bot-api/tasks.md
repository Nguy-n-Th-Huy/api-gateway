## 1. Settings and options

- [x] 1.1 Add the Telegram bot integration settings holder alongside the existing Telegram constants: enablement flag, service key, outbound callback address, callback signing secret, and the registration-handle requirement flag
- [x] 1.2 Register every new option in the option map and its apply switch, defaulting the registration-handle requirement to enabled through option initialization rather than a GORM boolean default tag
- [x] 1.3 Treat the service key and the callback signing secret as write-only on save: an empty submitted value keeps the stored secret, matching how the SePay webhook key behaves
- [x] 1.4 Add a configuration predicate reporting whether the bot integration can actually serve requests, and a companion reporting which required fields are missing
- [x] 1.5 Reject enabling the integration while the service key is empty, with a message naming what is missing ← (verify: secrets never round-trip to the client, an empty save preserves the stored value, and enabling without a key is refused)

## 2. Data model

- [x] 2.1 Add the Telegram handle column to the user model as an additive, non-unique, unindexed varchar(32) and include it in automatic migration
- [x] 2.2 Add handle normalization (trim, strip a leading `@`, lower-case) and format validation (5–32 characters, letters digits underscore only, must start with a letter) as reusable model-level behavior
- [x] 2.3 Extend the auth-flow creation input to accept a caller-supplied token, still persisting only its hash, so a short human-typeable code can use the existing single-use expiring flow machinery
- [x] 2.4 Add the Telegram link purpose constant and a code generator using an 8-character alphabet with visually ambiguous characters removed, normalizing to upper case before hashing
- [x] 2.5 Expose the handle on the self-user payload and on the account-bindings surface without exposing it as a lookup path ← (verify: migration is additive and idempotent, no unique index or lookup by handle exists anywhere, and the auth-flow token column still stores only a hash)

## 3. Shared producers extracted before any new caller exists

- [x] 3.1 Extract SePay top-up order creation from the existing handler into a service function taking a user identifier and an amount, returning the created order and its payment details, preserving the current validation chain, minimum and maximum bounds, wallet-capacity check, currency conversion, memo generation, and expiry exactly
- [x] 3.2 Rewire the existing web handler to call that function and confirm the existing SePay order tests pass unchanged
- [x] 3.3 Extract the key report body from the public key-check handler into a shared producer returning the documented field set, and rewire the public handler to call it
- [x] 3.4 Confirm the existing public key-check tests pass unchanged against the extracted producer ← (verify: this is a move not a rewrite — existing SePay and key-check tests pass without modification, and no validation step or field was dropped in the move)

## 4. Registration and self-service handle

- [x] 4.1 Accept the Telegram handle on registration, normalize and validate it, and store it on the created account
- [x] 4.2 Reject registration with a missing or invalid handle while the requirement option is enabled, creating no account; keep the handle optional while it is disabled
- [x] 4.3 Let a signed-in user change their own handle through the existing self-update path, applying the same normalization and validation, without touching any binding
- [x] 4.4 Publish the requirement flag on the public status payload so the sign-up screen can follow it ← (verify: an account is never created when the requirement is on and the handle is absent or malformed, and changing a handle never alters a binding)

## 5. Link code issuance and redemption

- [x] 5.1 Implement code issuance for a Telegram account identifier: refuse when that identifier is already bound, otherwise create a single-use flow carrying the identifier with a five-minute expiry and return the code, its expiry, and the redemption address
- [x] 5.2 Implement the website redemption endpoint behind user authentication, consuming the code and binding inside one transaction
- [x] 5.3 Apply every guard the existing Telegram binding path applies: refuse an already-bound Telegram identity, refuse an account that already holds a Telegram binding, refuse a disabled or deleted account, refuse a revoked session, and claim the external identity through the existing unique-claim path
- [x] 5.4 Overwrite the stored handle with the value reported for the linked Telegram account, clearing it when none is reported
- [x] 5.5 Return identical responses for unknown, expired, and already-consumed codes, and rate limit redemption per session and per client address
- [x] 5.6 Record an operator-visible entry for every refusal, naming the reason, the account, and the Telegram identity, and containing no code or credential ← (verify: concurrent redemption of one code binds exactly once, every refusal leaves all bindings untouched, and the three failure modes are indistinguishable to the caller)

## 6. Bot surface authentication, throttling, and auditing

- [x] 6.1 Add service-key authentication reading the `Bot` authorization scheme and comparing constant-time, rejecting missing, malformed, and non-matching credentials identically
- [x] 6.2 Reject session tokens and personal access tokens on this surface
- [x] 6.3 Return not-found for every request while the integration is disabled or unkeyed, so an unconfigured deployment exposes no surface
- [x] 6.4 Add a rate limiter keyed by Telegram account identifier that delegates to the existing pre-built-key user limiter, and apply the ordinary global limit to the endpoint that takes no Telegram user
- [x] 6.5 Add Telegram identity resolution shared by every account-scoped handler, returning a distinct not-linked outcome and a separate outcome for a disabled or deleted account
- [x] 6.6 Audit every account-scoped request with the Telegram identifier, resolved account, endpoint, and outcome, and ensure no key material or credential reaches an audit entry or log line
- [x] 6.7 Mount the route group and confirm it does not collide with existing routes ← (verify: constant-time comparison is genuinely used, throttling one Telegram identifier leaves others unaffected, and no log or audit line can contain a key)

## 7. Bot surface endpoints

- [x] 7.1 Health report: version, top-up availability, minimum and maximum amounts, and the flags that change bot behavior
- [x] 7.2 Identity resolution: linked state, and when linked the account identifier, console username, and status, with no balance, key, order, or log data
- [x] 7.3 Link code issuance endpoint delegating to the issuance logic from group 5
- [x] 7.4 Account summary: remaining balance, used quota, effective group, status, and stored handle, in the units the console uses
- [x] 7.5 Top-up configuration: availability, bounds, presets, conversion price, and order lifetime
- [x] 7.6 Top-up order creation delegating to the extracted producer, returning trade number, memo, payable amount, transfer QR payload, and deadline
- [x] 7.7 Order status lookup scoped to the owning account, returning an indistinguishable not-found for a non-owner and for an unknown trade number
- [x] 7.8 Paginated top-up history restricted to the resolved account
- [x] 7.9 Key inspection delegating to the extracted report producer and adding the requester-ownership flag, reading the key only from the request body
- [x] 7.10 Key listing returning name, group, quota, status, and expiry, and never a key value or any fragment of one
- [x] 7.11 Paginated usage log listing restricted to the resolved account, filterable by type, time range, model name, and key name
- [x] 7.12 Usage statistics over a requested range, matching the figures the console reports
- [x] 7.13 Key-scoped usage log, reading the key only from the request body
- [x] 7.14 Reject a key supplied only in a path segment or query parameter on every key-accepting endpoint, and confirm no unlink route exists on this surface ← (verify: all thirteen endpoints match their spec scenarios, ownership scoping holds on orders and logs, key listing leaks no key material, and no key is ever read from a URL)

## 8. Outbound events

- [x] 8.1 Add the event sender: signed with the configured secret over the exact body, carrying a stable event identifier, with a request timeout and bounded increasing-delay retry, abandoning with an operator-visible entry
- [x] 8.2 Emit nothing while the callback address, signing secret, or integration flag is missing, and nothing for an account with no Telegram binding
- [x] 8.3 Emit the credited event after a top-up credit has committed, carrying the Telegram identity, trade number, credited amount, and resulting balance
- [x] 8.4 Emit the expired event from the expiry sweep, carrying the Telegram identity and trade number
- [x] 8.5 Emit the linked event after a successful redemption, carrying the Telegram identity and console username
- [x] 8.6 Ensure no payload carries an API key, key fragment, service key, signing secret, or session token ← (verify: an unreachable, hanging, or erroring callback never delays, reverses, or duplicates a credit, the payment webhook still acknowledges normally, and retries reuse the same event identifier)

## 9. Frontend

- [x] 9.1 Add the Telegram handle field to the sign-up form and its schema, required only while the status flag says so, with client-side validation matching the server rules
- [x] 9.2 Add handle display and editing to the profile settings surface
- [x] 9.3 Add link-code entry to the profile account-bindings tab, with pending, success, and failure states and a message for each distinct refusal
- [x] 9.4 Add the bot integration settings to the Telegram area of the authentication settings screen, treating the service key and signing secret as write-only inputs that never display a stored value
- [x] 9.5 Add English and Vietnamese entries for every new string, with English source strings as keys ← (verify: no user-visible string bypasses translation, no secret is ever rendered back into an input, and the sign-up field follows the flag in both states)

## 10. Tests

- [x] 10.1 Service-key authentication: correct key, wrong key, missing header, malformed scheme, session token, personal access token, and disabled integration
- [x] 10.2 Handle normalization and format validation as a table test covering leading `@`, mixed case, surrounding whitespace, too short, too long, illegal characters, and a leading digit
- [x] 10.3 Registration under both states of the requirement option, including the account-not-created assertions
- [x] 10.4 Link code lifecycle: valid redemption, reuse, expiry, unknown code, indistinguishable failure responses, and concurrent redemption binding exactly once
- [x] 10.5 Redemption guards: already-bound identity, already-bound account, disabled account, revoked session, each asserting bindings are unchanged
- [x] 10.6 Order ownership: owner reads their order, non-owner and unknown trade number produce identical responses
- [x] 10.7 Key listing response contains no key value or fragment, asserted against the serialized payload
- [x] 10.8 Key-accepting endpoints reject a key supplied only in the query string
- [x] 10.9 Bot order creation shares the console validation chain: below minimum, above maximum, and beyond wallet capacity each rejected with no order created
- [x] 10.10 Per-Telegram-user throttling isolates one identifier from another
- [x] 10.11 Event delivery failure leaves the credit intact and the order settled ← (verify: tests assert real contracts rather than implementation details, use table form with explicit expected values, and initialize their own database, context, and settings state)

## 11. Database verification

- [x] 11.1 Run migration and startup against SQLite, MySQL, and PostgreSQL on a fresh database
- [x] 11.2 Run migration against a database created by the latest released version, confirming existing rows, indexes, constraints, and uniqueness guarantees survive
- [x] 11.3 Run startup twice on each engine to prove migration idempotency, confirming no repeated schema alteration on restart
- [x] 11.4 Record the exact engine versions, commands, and results in the handoff ← (verify: all three engines exercised on both fresh and upgraded databases, with the evidence recorded rather than asserted)

## 12. Contract documentation for the separate bot repository

- [x] 12.1 Document every endpoint with its authentication, request shape, response shape, and error codes, including the distinct not-linked code
- [x] 12.2 Document the outbound event payloads, the signature scheme, and the event identifier semantics
- [x] 12.3 State the consumer obligation to delete chat messages carrying an API key immediately after handling, and name the key-listing endpoint as the alternative that keeps key material out of a chat
- [x] 12.4 Document operator setup: generating and rotating the service key, configuring the callback address and secret, and disabling the integration as the rollback ← (verify: the document is sufficient to implement a client without reading gateway source, and the key-handling obligation is stated prominently)
