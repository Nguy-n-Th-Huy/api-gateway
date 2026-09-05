## Purpose

Lets anyone holding an API key inspect that key's own usage and configuration — name, group, quota, status, expiry and model limits — without an account, a session, or console access.

## ADDED Requirements

### Requirement: Public key check endpoint accepts the key in the request body

The system SHALL expose an unauthenticated endpoint `POST /api/token/check` that reads the API key from the JSON request body field `key`. The system SHALL NOT accept the key from the URL path or query string on this endpoint, so keys never reach access logs or `Referer` headers.

#### Scenario: Key supplied in body

- **WHEN** a client sends `POST /api/token/check` with body `{"key": "sk-abc123def456"}` for an existing key
- **THEN** the response status is `200` and the body contains `success: true` with the key's report data

#### Scenario: Key supplied only in query string

- **WHEN** a client sends `POST /api/token/check?key=sk-abc123def456` with an empty body
- **THEN** the response status is `400` and the body contains `success: false` with a localized "key is required" message

#### Scenario: Malformed JSON body

- **WHEN** a client sends `POST /api/token/check` with a body that is not valid JSON
- **THEN** the response status is `400` and the body contains `success: false` with a localized "key is required" message

### Requirement: Submitted keys are normalized before lookup

The system SHALL normalize the submitted key before lookup by trimming surrounding whitespace, removing a leading `Bearer ` or `bearer ` prefix, removing a leading `sk-` prefix, and keeping only the segment before the first `-` that remains. This matches the normalization already applied to relay and read-only token authentication, so a key pasted in any of those forms resolves to the same token.

#### Scenario: Key pasted with the sk- prefix

- **WHEN** the submitted key is `sk-abc123def456` and a token with key `abc123def456` exists
- **THEN** the report for that token is returned

#### Scenario: Key pasted with a Bearer prefix and surrounding whitespace

- **WHEN** the submitted key is `"  Bearer sk-abc123def456  "` and a token with key `abc123def456` exists
- **THEN** the report for that token is returned

#### Scenario: Key pasted with a trailing segment

- **WHEN** the submitted key is `sk-abc123def456-extra` and a token with key `abc123def456` exists
- **THEN** the report for that token is returned

### Requirement: Key check reports the key's full usage and configuration

For an existing key the system SHALL return a report containing exactly these fields: `name`, `group`, `status`, `unlimited_quota`, `total_granted`, `total_used`, `total_available`, `expires_at`, `created_time`, `accessed_time`, `model_limits_enabled`, `model_limits`, and `available_models`.

`total_granted` SHALL equal the key's remaining quota plus its used quota. `total_used` SHALL be the used quota and `total_available` the remaining quota, both in raw quota units. `expires_at` SHALL carry the stored expiry timestamp, where `-1` means the key never expires. `model_limits` SHALL be the map of allowed models and is only meaningful when `model_limits_enabled` is true.

#### Scenario: Report for a normal key

- **WHEN** a valid key belonging to a token named `my-key` with `used_quota` 500000 and `remain_quota` 1500000 is checked
- **THEN** the report has `name` `my-key`, `total_used` 500000, `total_available` 1500000, and `total_granted` 2000000

#### Scenario: Report for a never-expiring key

- **WHEN** a valid key whose stored expiry is `-1` is checked
- **THEN** the report has `expires_at` equal to `-1`

### Requirement: Key check reports the models available to the key's group

The report SHALL include `available_models`, the list of model names enabled for the key's effective group. This list SHALL be the sole source for the model choices offered anywhere on the page, so a visitor is never offered a model the key cannot actually call.

#### Scenario: Group with a restricted model set

- **WHEN** a key whose effective group enables three models is checked
- **THEN** `available_models` contains exactly those three model names

#### Scenario: Group with no enabled models

- **WHEN** a key whose effective group has no enabled model is checked
- **THEN** `available_models` is an empty list and the response still succeeds

### Requirement: Key check reports the key's effective group

The system SHALL report the group that actually applies to the key: the token's own group when the token has a non-empty group, otherwise the group of the user who owns the token.

#### Scenario: Token has its own group

- **WHEN** a key whose token group is `vip` is checked
- **THEN** the report has `group` `vip`

#### Scenario: Token inherits the owner's group

- **WHEN** a key whose token group is empty is checked and the owning user's group is `default`
- **THEN** the report has `group` `default`

### Requirement: Key check reports the effective status without mutating stored state

The system SHALL report the key's status as evaluated at query time using these rules, in order: `disabled` (2) when the stored status is disabled; `expired` (3) when the key has an expiry other than `-1` that is already in the past; `exhausted` (4) when the key does not have unlimited quota and its remaining quota is at most zero; otherwise `enabled` (1). The endpoint SHALL NOT write the recomputed status, or any other field, back to the database.

#### Scenario: Disabled key still returns a report

- **WHEN** a key whose stored status is disabled is checked
- **THEN** the response status is `200` and the report has `status` `2`

#### Scenario: Key past its expiry

- **WHEN** a key with an expiry timestamp in the past and a positive remaining quota is checked
- **THEN** the report has `status` `3`

#### Scenario: Key out of quota

- **WHEN** a key that is enabled, has no expiry, does not have unlimited quota, and has remaining quota `0` is checked
- **THEN** the report has `status` `4`

#### Scenario: Unlimited-quota key with zero remaining quota

- **WHEN** a key that is enabled, has no expiry, has unlimited quota, and has remaining quota `0` is checked
- **THEN** the report has `status` `1` and `unlimited_quota` true

#### Scenario: Stored state is untouched

- **WHEN** an expired or exhausted key is checked
- **THEN** the token row in the database keeps the status value it had before the request

### Requirement: Key check never discloses account identity

The report SHALL NOT contain the owning account's id, username, email, display name, or any other account identifier, because the person holding a key is not necessarily the account owner.

#### Scenario: Report omits account fields

- **WHEN** any valid key is checked
- **THEN** the response body contains no user id, username, or email field

### Requirement: Unknown keys get a generic rejection

The system SHALL answer a submitted key that matches no token with a single generic localized "invalid key" message and SHALL use the same message and status for every unmatched key, so the response does not distinguish a never-existing key from a deleted one. Empty input SHALL be answered with a localized "key is required" message. A database failure SHALL be answered with a `500` status, a localized generic error message, and a server-side log entry that does not contain the submitted key.

#### Scenario: Key does not exist

- **WHEN** a syntactically valid key matching no token is checked
- **THEN** the response contains `success: false` and the generic localized invalid-key message, with no indication of whether the key ever existed

#### Scenario: Empty key

- **WHEN** the request body has `key` set to an empty or whitespace-only string
- **THEN** the response status is `400` with a localized "key is required" message

### Requirement: Key check endpoint is rate limited

The endpoint SHALL be protected by the project's critical-operation rate limit so that repeated lookups from one source cannot be used to enumerate keys at speed.

#### Scenario: Rapid repeated lookups

- **WHEN** a client exceeds the critical rate limit on `POST /api/token/check`
- **THEN** further requests are rejected by the rate limiter until the window resets

### Requirement: Existing token usage endpoint keeps its contract

`GET /api/usage/token` SHALL keep every field it returns today with unchanged names and meanings, and SHALL keep its existing authentication behavior.

#### Scenario: Existing consumer unaffected

- **WHEN** an existing client calls `GET /api/usage/token` with a valid enabled key
- **THEN** the response still contains `name`, `total_granted`, `total_used`, `total_available`, `unlimited_quota`, `model_limits`, `model_limits_enabled`, and `expires_at` with the same meanings as before this change

### Requirement: Public key check page is reachable without signing in

The system SHALL serve a page at the path `/key` that any visitor can open without an account or session, and that does not redirect to sign-in.

#### Scenario: Anonymous visitor opens the page

- **WHEN** a visitor with no session opens `/key`
- **THEN** the key check page renders with its input form and no sign-in redirect occurs

### Requirement: Key check page validates input before calling the API

The page SHALL require a key value, trim it, and require at least 8 characters. When validation fails the page SHALL show a localized error message beneath the input and SHALL NOT call the backend.

#### Scenario: Submitting an empty field

- **WHEN** the visitor submits the form with the key field empty
- **THEN** a localized "key is required" message appears beneath the input and no request is sent

#### Scenario: Submitting a too-short key

- **WHEN** the visitor submits a key shorter than 8 characters after trimming
- **THEN** a localized minimum-length message appears beneath the input and no request is sent

#### Scenario: Submitting with the Enter key

- **WHEN** the visitor types a valid key in the input and presses Enter
- **THEN** the lookup is submitted just as it is by activating the submit button

### Requirement: Key check page presents every documented state

The page SHALL present four distinct states: idle before any lookup, loading during a lookup, error after a failed lookup, and success after a completed lookup.

In the idle state the page SHALL show guidance explaining what to paste and SHALL NOT show a result panel. In the loading state the submit control SHALL be disabled and SHALL show a busy indicator. In the error state the page SHALL show the server's localized message inline in a live region and SHALL leave the form editable so the visitor can submit again. In the success state the page SHALL show the result panel.

#### Scenario: Idle state

- **WHEN** the page is first opened
- **THEN** guidance text is shown and no result panel is present

#### Scenario: Loading state

- **WHEN** a lookup request is in flight
- **THEN** the submit control is disabled and shows a busy indicator

#### Scenario: Error state

- **WHEN** the backend rejects the key
- **THEN** the localized error message is shown inline, the previous result panel is not shown as if it were current, and the visitor can edit the key and submit again

#### Scenario: Success state

- **WHEN** the backend returns a report
- **THEN** the result panel is shown with the report's values

### Requirement: Result panel shows the complete key report

The result panel SHALL display the key name, the group, the status as a badge using the project's existing API key status presentation, the used / remaining / granted quota formatted in the site's configured quota display, the share of quota already used, the expiry, the creation time, the last-access time, and the model restriction.

When the key has unlimited quota the panel SHALL state that the quota is unlimited instead of showing a remaining-quota figure or a usage share. When the expiry is `-1` or `0` the panel SHALL state that the key never expires. When model limits are disabled the panel SHALL state that all models are allowed; when enabled it SHALL list the allowed models.

#### Scenario: Limited-quota key

- **WHEN** the report has `total_used` 500000, `total_available` 1500000, `total_granted` 2000000 and `unlimited_quota` false
- **THEN** the panel shows the three quota figures in the site's quota display format and a used share of 25%

#### Scenario: Unlimited-quota key

- **WHEN** the report has `unlimited_quota` true
- **THEN** the panel states the quota is unlimited and shows no remaining-quota figure or usage share

#### Scenario: Never-expiring key

- **WHEN** the report has `expires_at` `-1`
- **THEN** the panel states that the key never expires

#### Scenario: Key restricted to specific models

- **WHEN** the report has `model_limits_enabled` true with two allowed models
- **THEN** the panel lists those two models

#### Scenario: Key with no model restriction

- **WHEN** the report has `model_limits_enabled` false
- **THEN** the panel states that all models are allowed

### Requirement: Key check page does not re-display the full key

The page SHALL NOT render the submitted key back in the result panel in full. If the key is shown at all it SHALL be masked.

#### Scenario: Result panel after a successful lookup

- **WHEN** a report is shown
- **THEN** the full submitted key does not appear anywhere in the result panel

### Requirement: Key check page is usable by keyboard and assistive technology

The key input SHALL have an associated visible label. The region that carries the result and the error message SHALL be announced politely to assistive technology. After a successful lookup focus SHALL move to the result region. All controls SHALL be reachable and operable by keyboard, and text SHALL meet WCAG 2.1 AA contrast.

#### Scenario: Screen reader announcement after lookup

- **WHEN** a lookup finishes, successfully or with an error
- **THEN** the result region announces the new content politely

#### Scenario: Focus after a successful lookup

- **WHEN** a lookup succeeds
- **THEN** focus moves to the result region

### Requirement: Key check page text is localized

Every user-facing string on the page SHALL be resolved through the translation layer and SHALL have an entry in both the English and Vietnamese locale files.

#### Scenario: Vietnamese visitor

- **WHEN** the interface language is Vietnamese
- **THEN** every label, hint, validation message, and result field label on the page is shown in Vietnamese
