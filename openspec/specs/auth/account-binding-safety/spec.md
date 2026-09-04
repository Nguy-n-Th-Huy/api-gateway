# auth/account-binding-safety Specification

## Purpose
Protects an account when an external identity is matched to it, migrated to a newer identifier, or removed from it — so that no stranger can inherit an account by reusing a released name, and no administrator can strand a real user outside their own account.

## Requirements

### Requirement: A released upstream name cannot inherit an account

When an external identity is matched to an existing account by a name-based legacy identifier rather than by the provider's stable identifier, the match SHALL be accepted only if corroborating evidence ties the incoming identity to that account. A name alone SHALL NOT be sufficient, because upstream providers release a name for re-registration after its owner abandons it.

The corroborating evidence SHALL be the email address: the address the provider reports for the incoming identity must equal the address already stored on the matched account.

#### Scenario: Legitimate legacy user returns

- **WHEN** a visitor whose account was stored under a legacy name-based identifier signs in, and the provider reports the same email address the account already holds
- **THEN** the account is matched, its stored identifier is upgraded to the provider's stable identifier, and the visitor is signed in

#### Scenario: Stranger claims a released name

- **WHEN** a visitor signs in holding a provider name that matches an account's legacy identifier, but the provider reports a different email address than that account holds
- **THEN** the sign-in is refused with an explanation, the matched account is left completely untouched, and no new account is created for the visitor

#### Scenario: Corroboration impossible

- **WHEN** a legacy name matches an account, but either the provider reports no email address or the account holds none
- **THEN** the sign-in is refused with an explanation directing the visitor to contact support, and no account is adopted or created

#### Scenario: Already migrated

- **WHEN** a visitor whose identity is already stored under the provider's stable identifier signs in
- **THEN** they are matched directly and the legacy path is never reached

### Requirement: Identifier migrations are auditable

Whenever a stored external identifier is replaced by the provider's stable identifier, the system SHALL record an audit entry naming the account, the old identifier, and the new one. A failure to persist the upgraded identifier SHALL be recorded as an error rather than passed over in silence.

#### Scenario: Migration recorded

- **WHEN** a legacy identifier is successfully upgraded to a stable one
- **THEN** an audit entry records the account and both identifiers

#### Scenario: Migration write fails

- **WHEN** the upgraded identifier cannot be persisted
- **THEN** the failure is recorded as an error

### Requirement: A binding cannot be removed if it is the last way in

Removing a binding from an account SHALL be refused when the account would be left with no means of signing in — that is, with no password set and no remaining binding of any kind, counting the email address as a binding. The refusal SHALL state why, and SHALL leave the account exactly as it was.

#### Scenario: Another method remains

- **WHEN** a binding is removed from an account that keeps a password, or keeps at least one other binding
- **THEN** the binding is removed and the account stays reachable

#### Scenario: Last method would be removed

- **WHEN** removing a binding would leave the account with no password and no other binding
- **THEN** the removal is refused with an explanation and nothing about the account changes

#### Scenario: Email is counted

- **WHEN** an account has no password and no provider binding, and holds only an email address
- **THEN** clearing that email address is refused on the same grounds

### Requirement: Refusals are reported to the operator

An administrator whose removal request is refused SHALL be told which account it concerned and why the removal was refused, rather than receiving a generic failure.

#### Scenario: Administrator sees the reason

- **WHEN** an administrator attempts a removal that would strand the account
- **THEN** the response explains that the account would be left with no way to sign in
