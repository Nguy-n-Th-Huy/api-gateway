# public-key-check Specification

## Purpose

Lets anyone holding an API key inspect that key's own usage and configuration — name, group, quota, status, expiry, model limits, and the entries recorded against it — without an account, a session, or console access.

## ADDED Requirements

### Requirement: Public key usage log endpoint accepts the key in the request body

The system SHALL expose an unauthenticated endpoint `POST /api/token/logs` that reads the API key from the JSON request body field `key` and returns the log entries recorded against that key, newest first. The endpoint SHALL NOT accept the key from the URL path or query string.

Paging SHALL use the project's existing page query parameters (`p`, `page_size`) with the project's existing clamping, and the response SHALL carry the page, the page size, the total number of entries for the key, and the entries of the requested page.

The endpoint SHALL answer a key that matches no token with the same generic localized "invalid key" message and status the check endpoint uses, SHALL answer empty input with the localized "key is required" message and a `400` status, and SHALL answer a log query failure with a `500` status, a localized generic message and a server-side log entry that does not contain the submitted key.

#### Scenario: Entries for the supplied key

- **WHEN** a client sends `POST /api/token/logs` with body `{"key": "sk-abc123def456"}` for an existing key that has recorded usage
- **THEN** the response status is `200` and the body contains `success: true` with the key's entries, newest first, and the total number of entries

#### Scenario: Key supplied only in the query string

- **WHEN** a client sends `POST /api/token/logs?key=sk-abc123def456` with an empty body
- **THEN** the response status is `400` and the body contains `success: false` with a localized "key is required" message

#### Scenario: Unknown key

- **WHEN** a key matching no token is submitted
- **THEN** the response contains `success: false`, the generic localized invalid-key message, and no entries

#### Scenario: Paging

- **WHEN** a key has more entries than the requested page size and the client asks for a later page
- **THEN** the response carries only that page's entries while the total still counts every entry for the key

#### Scenario: Page size above the platform maximum

- **WHEN** the client requests more entries than the platform's maximum page size
- **THEN** the response is capped at the platform's maximum page size rather than returning every entry

#### Scenario: Key check endpoint is unaffected

- **WHEN** `POST /api/token/check` is called
- **THEN** it still returns the report documented in "Key check reports the key's full usage and configuration", with no log entries added and no field removed

### Requirement: Key usage log entries exclude account identity and infrastructure fields

Each returned entry SHALL contain exactly these fields: `created_at`, `type`, `model_name`, `quota`, `prompt_tokens`, `completion_tokens`, `use_time`, `is_stream`, `group`, and `request_id`.

An entry SHALL NOT contain the owning account's id, username, or email; SHALL NOT contain the requesting client's IP address; and SHALL NOT contain the upstream channel's id or name, the token's stored id, or the raw log metadata blob.

This field set SHALL have exactly one producer, so every surface reporting a key's entries can be changed in one place rather than per surface.

#### Scenario: Entry for a consumed request

- **WHEN** a key with recorded consumption is queried
- **THEN** each entry carries the request's timestamp, log type, model name, quota, prompt and completion token counts, duration, streaming flag, group, and request id

#### Scenario: No identity or infrastructure leak

- **WHEN** any key is queried
- **THEN** no entry in the response body contains `user_id`, `username`, `email`, `ip`, `channel`, `channel_name`, `token_id`, `token_name`, or `other`

### Requirement: Key check page presents the checked key's usage log

The page SHALL render a usage-log section. Before a key check has succeeded the section SHALL state that a key has to be checked first and SHALL show no entries.

After a successful check the section SHALL load that key's entries and present them as a table whose columns are the request time, the log type, the model, the token counts, the cost, and the duration. The section SHALL offer controls to move to the previous and the next page, SHALL show which page is being displayed together with the total number of pages, and SHALL disable the control that cannot be used on the current page.

When the key has recorded no entry the section SHALL show a localized empty state instead of a table. When loading the entries fails the section SHALL show a localized error and a retry control, SHALL keep the key's report visible, and SHALL NOT present the entries of a previously checked key as current.

The section SHALL NOT display the checked key, and a new successful check SHALL replace the previous key's entries rather than appending to them.

#### Scenario: Before any check

- **WHEN** the page is opened and no key has been checked
- **THEN** the section states that a key must be checked first and shows no table

#### Scenario: Entries after a successful check

- **WHEN** a key with recorded usage is checked
- **THEN** the section shows that key's entries with their time, type, model, tokens, cost and duration

#### Scenario: Paging controls

- **WHEN** the key has more entries than one page
- **THEN** the section shows the current page and the total number of pages, allows moving to the next page, and does not allow moving before the first page

#### Scenario: Key with no recorded usage

- **WHEN** a key with no entries is checked
- **THEN** the section shows a localized empty state

#### Scenario: Loading the entries fails

- **WHEN** the entries request fails
- **THEN** the section shows a localized error with a retry control while the key's report stays visible

#### Scenario: A different key is checked

- **WHEN** a second, successful check follows a first one
- **THEN** the section shows only the second key's entries and is back on the first page

#### Scenario: The key is never displayed

- **WHEN** the section renders, with or without entries
- **THEN** the full submitted key does not appear anywhere in the section
