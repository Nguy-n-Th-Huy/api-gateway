# auth/google-oauth Specification

## Purpose
Lets a visitor create an account and sign in with their Google identity, lets an existing account bind that identity, and defines the operator configuration that turns the whole thing on.

## Requirements

### Requirement: Google sign-in is offered only when configured

The Google sign-in action SHALL appear on the sign-in and sign-up pages only when an operator has enabled Google authentication. The public status payload SHALL carry a flag stating whether Google authentication is enabled, and SHALL NOT expose the client secret or any other credential.

#### Scenario: Google disabled

- **WHEN** an operator has not enabled Google authentication
- **THEN** no Google action appears on the sign-in or sign-up page, and an authorization attempt against the Google provider is rejected

#### Scenario: Google enabled

- **WHEN** an operator has enabled Google authentication and supplied its credentials
- **THEN** a Google action appears alongside the other enabled providers

#### Scenario: No credential leaks

- **WHEN** the public status payload is fetched by an unauthenticated visitor
- **THEN** it states whether Google authentication is enabled and contains no client secret

### Requirement: First Google sign-in creates an account

When a visitor authorizes with a Google identity that is not yet bound to any account, and registration is open, the system SHALL create a new account bound to that Google identity and sign the visitor in. When registration is closed, the system SHALL refuse and explain that registration is disabled, without creating an account.

#### Scenario: New visitor, registration open

- **WHEN** an unrecognized Google identity completes authorization and registration is open
- **THEN** an account is created, bound to that Google identity, and the visitor is signed in

#### Scenario: New visitor, registration closed

- **WHEN** an unrecognized Google identity completes authorization and registration is closed
- **THEN** no account is created and the visitor is told registration is disabled

#### Scenario: Returning visitor

- **WHEN** a Google identity already bound to an account completes authorization
- **THEN** that existing account is signed in and no second account is created

### Requirement: Google identity is bound by its stable identifier

The system SHALL identify a Google account by the stable subject identifier Google issues for it, not by email address or display name, because a Google email address can be changed or reassigned. A given Google identity SHALL bind to at most one account.

#### Scenario: Email changes upstream

- **WHEN** a visitor changes the email address on their Google account and signs in again
- **THEN** they reach the same gateway account as before

#### Scenario: Identity already bound

- **WHEN** a visitor tries to bind a Google identity that is already bound to a different account
- **THEN** the bind is refused with an explanation, and neither account is modified

### Requirement: Account creation tolerates missing or colliding profile data

Account creation from a Google identity SHALL succeed even when Google returns no email address, and SHALL produce a username that does not collide with an existing account.

#### Scenario: No email returned

- **WHEN** Google returns a profile without an email address
- **THEN** the account is still created and bound, with the email left unset

#### Scenario: Username collision

- **WHEN** the username derived from the Google profile is already taken
- **THEN** a distinct username is generated and the account is created

### Requirement: An existing account can bind Google

A signed-in user SHALL be able to bind a Google identity to their account from their account settings, and the settings SHALL show whether Google is currently bound. Once bound, the binding SHALL be removable only by an administrator, matching every other built-in provider — no self-service unbind exists for built-in providers, and this capability does not introduce one.

#### Scenario: Bind from the account settings

- **WHEN** a signed-in user with no Google binding completes Google authorization from their account settings
- **THEN** the Google identity is bound to their account and shown as bound

#### Scenario: Bound state is not self-removable

- **WHEN** a signed-in user views a Google binding that is already bound
- **THEN** the settings show it as bound and offer no self-service removal, consistent with the other built-in providers

#### Scenario: Administrator clears the binding

- **WHEN** an administrator clears a user's Google binding
- **THEN** the stored Google identifier is removed from that account and a subsequent Google authorization is treated as an unrecognized identity

### Requirement: Authorization failures are reported without leaking internals

When the authorization exchange or the profile fetch fails, the system SHALL report a message identifying which provider failed and SHALL NOT expose the client secret, the access token, or a raw upstream response body to the visitor.

#### Scenario: Upstream unreachable

- **WHEN** the token exchange with Google fails
- **THEN** the visitor sees a failure message naming Google, and no credential or token appears in what is returned to the browser

#### Scenario: Authorization code invalid

- **WHEN** the authorization code is missing or rejected
- **THEN** the sign-in attempt fails and no account is created or modified

### Requirement: The Google binding column works on every supported database

The user record SHALL gain an indexed column holding the Google subject identifier, and that column SHALL be created correctly by the migration on SQLite, MySQL and PostgreSQL, both on a fresh database and on a database created by the previous release. Running the migration more than once SHALL NOT fail or alter existing data.

#### Scenario: Fresh database

- **WHEN** the application starts against an empty database on any supported engine
- **THEN** the users table is created with the Google identifier column indexed, and startup succeeds

#### Scenario: Upgraded database

- **WHEN** the application starts against a database created by the previous release on any supported engine
- **THEN** the Google identifier column is added, existing rows and their data are preserved, and existing indexes and uniqueness guarantees still hold

#### Scenario: Repeated migration

- **WHEN** the application is started twice in a row against the same database
- **THEN** the second start makes no further schema change and does not fail

### Requirement: Google interface text is translatable

All user-facing text introduced for Google authentication SHALL resolve through the translation layer, with Vietnamese supplied and the remaining supported locales carrying the key.

#### Scenario: Language switch

- **WHEN** a visitor switches the interface language
- **THEN** the Google sign-in action and its account-settings entries render in the selected language
