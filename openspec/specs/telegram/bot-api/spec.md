# telegram/bot-api Specification

## Purpose
Defines the service-to-service surface at `/api/bot/v1` that an externally hosted Telegram bot calls on behalf of a Telegram user: how the bot authenticates, how abuse by one Telegram user is isolated from every other, what each endpoint reports, and which powers the surface deliberately withholds.

The link-code lifecycle reachable at `/api/bot/v1/identity/link/start` is specified in `telegram/account-link`. Outbound delivery from the gateway back to the bot is specified in `telegram/bot-events`.

## Requirements

### Requirement: Bot service key authentication

The system SHALL authenticate every `/api/bot/v1` request with a single administrator-configured bot service key presented as `Authorization: Bot <key>`. Comparison SHALL be constant-time. A request with a missing, malformed, or non-matching credential SHALL be rejected with HTTP 401 and SHALL NOT reveal whether the key exists, whether a Telegram user is known, or any account data.

#### Scenario: Valid service key

- **WHEN** the bot calls a `/api/bot/v1` endpoint with `Authorization: Bot <configured key>`
- **THEN** the request is authenticated and processed

#### Scenario: Wrong service key

- **WHEN** a caller presents a bot service key that does not match the configured key
- **THEN** the response is HTTP 401, no account data is returned, and no order, code, or binding is created

#### Scenario: Missing or malformed credential

- **WHEN** a caller omits the `Authorization` header, or sends it without the `Bot` scheme
- **THEN** the response is HTTP 401 and the body does not distinguish this case from a wrong key

#### Scenario: Session or personal access token presented instead

- **WHEN** a caller presents a user session token or a personal access token to a `/api/bot/v1` endpoint
- **THEN** the response is HTTP 401, because the bot surface accepts only the bot service key

### Requirement: The bot API is unavailable until it is configured and enabled

The system SHALL reject every `/api/bot/v1` request with HTTP 404 while the bot integration is disabled or its service key is unset. An incompletely configured integration SHALL be treated as absent rather than as a soft failure, so an unconfigured deployment exposes no bot surface at all.

#### Scenario: Integration disabled

- **WHEN** the bot integration is disabled and any `/api/bot/v1` endpoint is called with a correct-looking credential
- **THEN** the response is HTTP 404 and nothing is created or read

#### Scenario: Enabled but no service key set

- **WHEN** the bot integration is enabled while the bot service key is empty
- **THEN** every `/api/bot/v1` request is rejected and no request is treated as authenticated

### Requirement: Rate limiting is scoped to the Telegram user, not the caller IP

Because every bot request arrives from the same host, the system SHALL apply the rate limit of user-scoped `/api/bot/v1` endpoints per `telegram_user_id` rather than per client IP. Exhausting the limit for one Telegram user SHALL NOT impede requests made on behalf of any other Telegram user.

#### Scenario: One Telegram user floods the surface

- **WHEN** requests carrying the same `telegram_user_id` exceed the configured limit
- **THEN** further requests for that `telegram_user_id` are rejected with HTTP 429 while requests carrying a different `telegram_user_id` continue to succeed

#### Scenario: Key inspection abuse is contained

- **WHEN** one Telegram user repeatedly submits candidate keys to the key inspection endpoint
- **THEN** that Telegram user is throttled and other Telegram users remain able to inspect their own keys

### Requirement: User-scoped bot requests are auditable

The system SHALL record an audit entry for every `/api/bot/v1` request that reads or changes account-scoped data, capturing the Telegram user identifier, the resolved account when one exists, the endpoint, and the outcome. Key material SHALL NOT appear in any audit entry or log line.

#### Scenario: Account read is audited

- **WHEN** the bot reads an account summary on behalf of a Telegram user
- **THEN** an audit entry records the Telegram user identifier, the resolved account, the endpoint, and the outcome

#### Scenario: Key inspection is audited without the key

- **WHEN** the bot inspects a key
- **THEN** the audit entry records that a key inspection occurred and its outcome, and contains neither the submitted key nor any prefix of it long enough to identify the key

### Requirement: Endpoints that need an account report an unlinked Telegram user distinguishably

For account-scoped endpoints, when the supplied `telegram_user_id` is not bound to any account, the system SHALL respond with a stable machine-readable error code that the bot can distinguish from an authentication failure, a rate-limit rejection, and an internal fault, so the bot can guide the person into the linking flow.

#### Scenario: Unlinked Telegram user asks for a balance

- **WHEN** the bot requests an account summary for a `telegram_user_id` bound to no account
- **THEN** the response carries a distinct not-linked error code and no account data

#### Scenario: Unlinked Telegram user tries to create a top-up order

- **WHEN** the bot attempts to create a top-up order for a `telegram_user_id` bound to no account
- **THEN** the response carries the not-linked error code and no order is created

#### Scenario: Account exists but is disabled

- **WHEN** the resolved account is disabled or deleted
- **THEN** the response distinguishes that state from the not-linked state and returns no balance, key, order, or log data

### Requirement: Integration health report

The system SHALL expose an endpoint that reports, without requiring a Telegram user, the information the bot needs to configure itself: the gateway version, whether top-up is currently available, the minimum and maximum top-up amounts, and the flags that change bot behaviour such as whether a Telegram handle is required at registration.

#### Scenario: Health report while top-up is available

- **WHEN** the bot calls the health endpoint while SePay is configured and payment compliance is confirmed
- **THEN** the report states that top-up is available and carries the current minimum and maximum amounts

#### Scenario: Health report while top-up is unavailable

- **WHEN** the bot calls the health endpoint while top-up is unavailable for any reason
- **THEN** the report states that top-up is unavailable, and the bot has enough information to hide its top-up commands

### Requirement: Telegram identity resolution

The system SHALL expose an endpoint that, given a `telegram_user_id`, reports whether that Telegram account is bound to a platform account and, when it is, the account identifier, the console username, and the account status. It SHALL NOT report balance, keys, orders, or logs.

#### Scenario: Bound Telegram account

- **WHEN** the bot resolves a `telegram_user_id` that is bound to an enabled account
- **THEN** the response reports that it is linked, along with the account identifier, console username, and status

#### Scenario: Unbound Telegram account

- **WHEN** the bot resolves a `telegram_user_id` that is bound to no account
- **THEN** the response reports that it is not linked and carries no account fields

### Requirement: Account summary

The system SHALL expose an endpoint returning, for a linked Telegram user, the account's remaining balance, used quota, effective group, account status, and stored Telegram handle. Quota figures SHALL be reported in the same units the console uses, so the bot and the console never disagree about a balance.

#### Scenario: Linked user reads a balance

- **WHEN** the bot requests the account summary for a linked Telegram user
- **THEN** the response carries the remaining balance, the used quota, the effective group, the account status, and the stored Telegram handle

### Requirement: Top-up configuration report

The system SHALL expose an endpoint reporting the current top-up parameters: whether top-up is available, the minimum and maximum amounts per order, the suggested amount presets, the conversion price, and how long a created order stays payable.

#### Scenario: Bot renders top-up choices

- **WHEN** the bot fetches the top-up configuration
- **THEN** it receives the same minimum, maximum, presets, price, and order lifetime that the web console would apply

#### Scenario: Configuration changes between requests

- **WHEN** an administrator changes the minimum amount and the bot fetches the configuration again
- **THEN** the new minimum is reported

### Requirement: Bot-initiated top-up order creation

The system SHALL expose an endpoint that creates a top-up order for a linked Telegram user and returns the trade number, the transfer memo, the payable amount in the settlement currency, the transfer QR payload, and the order deadline. The order SHALL be an ordinary SePay order, subject to exactly the same amount validation, wallet-ceiling check, memo generation, expiry, and webhook settlement as an order created from the web console. No alternative payment provider SHALL be introduced by this endpoint.

#### Scenario: Valid bot-initiated order

- **WHEN** the bot creates an order for a linked user with an amount inside the configured bounds
- **THEN** a pending SePay order is created and the response carries its trade number, memo, payable amount, QR payload, and deadline

#### Scenario: Amount below the minimum

- **WHEN** the bot submits an amount below the configured minimum
- **THEN** the request is rejected with a message naming the minimum and no order is created

#### Scenario: Amount above the per-order maximum

- **WHEN** the bot submits an amount above the per-order maximum
- **THEN** the request is rejected and no order is created

#### Scenario: Amount would exceed the wallet ceiling

- **WHEN** crediting the requested amount would push the account past the maximum representable wallet balance
- **THEN** the request is rejected and no order is created

#### Scenario: Bot-created order settles through the normal webhook

- **WHEN** a matching bank transfer for a bot-created order reaches the SePay webhook
- **THEN** the wallet is credited exactly once by the same settlement path that serves console-created orders

#### Scenario: Top-up unavailable

- **WHEN** the bot attempts to create an order while top-up is unavailable
- **THEN** the request is rejected and no order is created

### Requirement: Order lookup is scoped to the order's owner

The system SHALL return a top-up order's status only when the supplied `telegram_user_id` resolves to the account that owns that order. A request for an order belonging to another account SHALL be refused without revealing whether the trade number exists.

#### Scenario: Owner polls their order

- **WHEN** the bot polls an order using the `telegram_user_id` of the account that created it
- **THEN** the order's status, amount, and deadline are returned

#### Scenario: Non-owner polls an order

- **WHEN** the bot polls an order using a `telegram_user_id` resolving to a different account
- **THEN** the request is refused and the response does not disclose whether that trade number exists

#### Scenario: Unknown trade number

- **WHEN** the bot polls a trade number that does not exist
- **THEN** the response is indistinguishable from the non-owner case

### Requirement: Top-up history

The system SHALL expose a paginated endpoint listing a linked Telegram user's own top-up orders, reporting for each the trade number, amount, status, and creation time. Orders belonging to other accounts SHALL never appear.

#### Scenario: User reads their top-up history

- **WHEN** the bot lists top-up history for a linked Telegram user
- **THEN** only that account's orders are returned, paginated

### Requirement: Key inspection reuses the public key report

The system SHALL expose an endpoint that accepts an API key and returns the same report the public key-check endpoint returns for that key, plus a flag stating whether the key belongs to the account bound to the supplied `telegram_user_id`. The submitted key SHALL be normalized by the same rules the public endpoint applies, so a key pasted in any accepted form resolves identically.

#### Scenario: Key belonging to the requesting user

- **WHEN** the bot inspects a key owned by the account bound to the supplied `telegram_user_id`
- **THEN** the report is returned with the ownership flag set to true

#### Scenario: Key belonging to someone else

- **WHEN** the bot inspects a valid key that belongs to a different account
- **THEN** the same report is returned with the ownership flag set to false, matching what the public endpoint already discloses to any holder of that key

#### Scenario: Unknown key

- **WHEN** the bot inspects a key that matches no token
- **THEN** the response reports that the key was not found and carries no report fields

#### Scenario: Key report field set stays aligned

- **WHEN** the public key report gains or loses a field
- **THEN** the bot key inspection response carries the same change, because both read one shared report

### Requirement: Key listing never discloses key values

The system SHALL expose an endpoint listing the API keys owned by the account bound to the supplied `telegram_user_id`, reporting for each key its name, group, quota figures, status, and expiry. The response SHALL NOT contain the key value, any partial key value, or any field from which a key value can be reconstructed.

#### Scenario: User lists their keys

- **WHEN** the bot lists keys for a linked Telegram user
- **THEN** each entry carries name, group, quota, status, and expiry, and no entry carries a key value or fragment of one

#### Scenario: Listing is the safe alternative to pasting

- **WHEN** a person wants to check their keys without exposing key material in a chat
- **THEN** the listing endpoint provides that information without any key value crossing the chat

### Requirement: Usage log listing

The system SHALL expose a paginated endpoint returning a linked Telegram user's own usage log entries, filterable by entry type, time range, model name, and key name. Entries belonging to other accounts SHALL never be returned.

#### Scenario: User reads recent usage

- **WHEN** the bot lists usage logs for a linked Telegram user without filters
- **THEN** only that account's entries are returned, most recent first, paginated

#### Scenario: Filtered by model and time range

- **WHEN** the bot lists usage logs constrained to one model name and a time range
- **THEN** only that account's matching entries are returned

### Requirement: Usage statistics

The system SHALL expose an endpoint reporting aggregate usage for a linked Telegram user over a requested time range: total quota consumed, request count, and prompt and completion token totals. The figures SHALL match what the console reports for the same account and range.

#### Scenario: User reads a monthly summary

- **WHEN** the bot requests usage statistics for a linked Telegram user over a time range
- **THEN** the totals returned equal those the console reports for the same account and range

### Requirement: Key-scoped usage log

The system SHALL expose an endpoint that accepts an API key and returns the usage log entries recorded for that key. The key SHALL be supplied in the request body.

#### Scenario: Log for a supplied key

- **WHEN** the bot requests the usage log for a valid key
- **THEN** the entries recorded against that key are returned, paginated

#### Scenario: Unknown key

- **WHEN** the bot requests the usage log for a key that matches no token
- **THEN** the response reports that the key was not found and returns no entries

### Requirement: API keys are never accepted in a URL or query string

On every `/api/bot/v1` endpoint that accepts an API key, the system SHALL read it only from the JSON request body. A key supplied in the path or query string SHALL NOT be accepted, so key material never reaches access logs, proxy logs, or referrer headers.

#### Scenario: Key supplied in the body

- **WHEN** the bot submits a key in the JSON body of a key-accepting endpoint
- **THEN** the request is processed

#### Scenario: Key supplied only in the query string

- **WHEN** the bot submits a key only as a query parameter
- **THEN** the request is rejected as missing the key, and the key is not used for lookup

### Requirement: The bot surface cannot unlink an account

The system SHALL NOT expose any `/api/bot/v1` endpoint that removes a Telegram binding. Unbinding SHALL remain available only to an authenticated user on the website, so possession of the bot service key alone cannot detach any account from its Telegram identity.

#### Scenario: No unlink route exists

- **WHEN** a caller holding a valid bot service key attempts any request intended to remove a Telegram binding
- **THEN** no such route exists and no binding is removed

### Requirement: Consumers must not retain key material in chat

The published contract for this surface SHALL state that a bot submitting a key to the key inspection or key-scoped log endpoints must delete the chat message carrying that key immediately after handling it, because a key pasted into a chat otherwise persists in the messaging provider's history. The contract SHALL direct consumers to the key listing endpoint as the alternative that never moves key material through a chat.

#### Scenario: Contract states the deletion obligation

- **WHEN** an integrator reads the contract for the key-accepting endpoints
- **THEN** it states the message-deletion obligation and names the key listing endpoint as the alternative
