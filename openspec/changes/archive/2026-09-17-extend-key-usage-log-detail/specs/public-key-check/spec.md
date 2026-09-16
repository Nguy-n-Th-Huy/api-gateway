# public-key-check Specification

## Purpose

Lets anyone holding an API key inspect that key's own usage and configuration — name, group, quota, status, expiry, model limits, and the entries recorded against it — without an account, a session, or console access.

## MODIFIED Requirements

### Requirement: Key usage log entries exclude account identity and infrastructure fields

Each returned entry SHALL contain exactly these fields: `created_at`, `type`, `model_name`, `quota`, `prompt_tokens`, `completion_tokens`, `cache_tokens`, `cache_creation_tokens`, `request_path`, `use_time`, `is_stream`, `group`, and `request_id`.

`cache_tokens` SHALL carry the entry's cache-read token count and `cache_creation_tokens` its cache-write token count, both zero when the entry recorded neither. `request_path` SHALL carry the relay endpoint the entry was recorded for, and SHALL be empty when the entry has none. Both are read from the entry's own user-visible metadata, which the response SHALL NOT return as a whole.

An entry SHALL NOT contain the owning account's id, username, or email; SHALL NOT contain the requesting client's IP address; and SHALL NOT contain the upstream channel's id or name, the token's stored id, or the raw log metadata blob. A metadata value that cannot be read SHALL leave those fields empty or zero rather than failing the request.

This field set SHALL have exactly one producer, so every surface reporting a key's entries can be changed in one place rather than per surface.

#### Scenario: Entry for a consumed request

- **WHEN** a key with recorded consumption is queried
- **THEN** each entry carries the request's timestamp, log type, model name, quota, prompt and completion token counts, cache-read and cache-write token counts, the endpoint it was recorded for, duration, streaming flag, group, and request id

#### Scenario: Entry that recorded no cache usage

- **WHEN** an entry whose metadata carries no cache figures is queried
- **THEN** its `cache_tokens` and `cache_creation_tokens` are `0` and the response still succeeds

#### Scenario: Entry with unreadable metadata

- **WHEN** an entry's metadata is not parseable
- **THEN** the entry is still returned with an empty `request_path` and zero cache counts

#### Scenario: No identity or infrastructure leak

- **WHEN** any key is queried
- **THEN** no entry in the response body contains `user_id`, `username`, `email`, `ip`, `channel`, `channel_name`, `token_id`, `token_name`, or `other`

### Requirement: Key check page presents the checked key's usage log

The page SHALL render a usage-log section. Before a key check has succeeded the section SHALL state that a key has to be checked first and SHALL show no entries.

After a successful check the section SHALL load that key's entries and present them as a table whose columns are, in order: the request time, the request type, the outcome, the input token count, the cached token count, the output token count, the cost, and the duration. The request type SHALL be the kind of endpoint the entry was recorded for, in the reader's language, and SHALL fall back to the entry's log-type wording when the entry records no endpoint. The outcome SHALL read as successful for a consumed request, as failed for an errored request, and SHALL be blank for an entry that is neither. Token and cache columns SHALL show the entry's own counts, not a combined total.

The section SHALL offer controls to move to the previous and the next page, SHALL show which page is being displayed together with the total number of pages, and SHALL disable the control that cannot be used on the current page.

When the key has recorded no entry the section SHALL show a localized empty state instead of a table. When loading the entries fails the section SHALL show a localized error and a retry control, SHALL keep the key's report visible, and SHALL NOT present the entries of a previously checked key as current.

The section SHALL NOT display the checked key, and a new successful check SHALL replace the previous key's entries rather than appending to them.

#### Scenario: Before any check

- **WHEN** the page is opened and no key has been checked
- **THEN** the section states that a key must be checked first and shows no table

#### Scenario: Entries after a successful check

- **WHEN** a key with recorded usage is checked
- **THEN** the section shows that key's entries with their time, request type, outcome, input, cached and output token counts, cost and duration

#### Scenario: Successful and failed requests

- **WHEN** a key's entries include a consumed request and an errored request
- **THEN** the consumed entry reads as successful and the errored entry reads as failed

#### Scenario: Entry with no endpoint

- **WHEN** an entry records no endpoint
- **THEN** its request type column shows the entry's log-type wording

#### Scenario: Entry with no cache usage

- **WHEN** an entry recorded no cache tokens
- **THEN** its cached token column shows no figure rather than a zero count

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
