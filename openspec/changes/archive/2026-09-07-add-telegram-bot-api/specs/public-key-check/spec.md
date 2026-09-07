## MODIFIED Requirements

### Requirement: Key check reports the key's full usage and configuration

For an existing key the system SHALL return a report containing exactly these fields: `name`, `group`, `status`, `unlimited_quota`, `total_granted`, `total_used`, `total_available`, `expires_at`, `created_time`, `accessed_time`, `model_limits_enabled`, `model_limits`, and `available_models`.

`total_granted` SHALL equal the key's remaining quota plus its used quota. `total_used` SHALL be the used quota and `total_available` the remaining quota, both in raw quota units. `expires_at` SHALL carry the stored expiry timestamp, where `-1` means the key never expires. `model_limits` SHALL be the map of allowed models and is only meaningful when `model_limits_enabled` is true.

This report is a shared contract with exactly one producer. Every surface that reports on a key — the public unauthenticated endpoint and the Telegram bot integration surface — SHALL return this same field set with identical semantics, so the two can never drift apart. A surface MAY add fields that describe the requester's relationship to the key rather than the key itself; such an addition SHALL NOT alter, remove, or re-interpret any field listed above.

#### Scenario: Report for a normal key

- **WHEN** a valid key belonging to a token named `my-key` with `used_quota` 500000 and `remain_quota` 1500000 is checked
- **THEN** the report has `name` `my-key`, `total_used` 500000, `total_available` 1500000, and `total_granted` 2000000

#### Scenario: Report for a never-expiring key

- **WHEN** a valid key whose stored expiry is `-1` is checked
- **THEN** the report has `expires_at` equal to `-1`

#### Scenario: Same key reported through both surfaces

- **WHEN** the same key is checked through the public endpoint and through the bot integration surface
- **THEN** both responses carry the same values for every field in the shared field set

#### Scenario: Field set changes in one place

- **WHEN** the shared report gains or loses a field
- **THEN** both the public endpoint and the bot integration surface reflect that change, because neither maintains its own copy of the field set

### Requirement: Public key check endpoint accepts the key in the request body

The system SHALL expose an unauthenticated endpoint `POST /api/token/check` that reads the API key from the JSON request body field `key`. The system SHALL NOT accept the key from the URL path or query string on this endpoint, so keys never reach access logs or `Referer` headers.

This body-only rule SHALL extend to every other endpoint in the system that accepts an API key for inspection or reporting, including those on the Telegram bot integration surface. No such endpoint SHALL read a key from a path segment or query parameter under any circumstance.

#### Scenario: Key supplied in body

- **WHEN** a client sends `POST /api/token/check` with body `{"key": "sk-abc123def456"}` for an existing key
- **THEN** the response status is `200` and the body contains `success: true` with the key's report data

#### Scenario: Key supplied only in query string

- **WHEN** a client sends `POST /api/token/check?key=sk-abc123def456` with an empty body
- **THEN** the response status is `400` and the body contains `success: false` with a localized "key is required" message

#### Scenario: Malformed JSON body

- **WHEN** a client sends `POST /api/token/check` with a body that is not valid JSON
- **THEN** the response status is `400` and the body contains `success: false` with a localized "key is required" message

#### Scenario: Key supplied in a query string on the bot surface

- **WHEN** a caller supplies a key only as a query parameter to a key-accepting endpoint on the bot integration surface
- **THEN** the request is rejected as missing the key and no lookup is performed with that value
