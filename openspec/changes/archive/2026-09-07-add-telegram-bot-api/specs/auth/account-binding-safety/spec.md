## ADDED Requirements

### Requirement: A code-carried binding intent proves control of both sides

A binding created by redeeming a code issued in an external messaging context SHALL require proof of control over both identities before it is written: control of the external identity, evidenced by the code having been delivered only into that identity's own channel, and control of the platform account, evidenced by an authenticated session at the moment of redemption. Neither proof alone SHALL be sufficient.

#### Scenario: Both proofs present

- **WHEN** a code delivered into an external identity's own channel is redeemed by a caller holding a valid session for a platform account
- **THEN** the binding between that external identity and that account is written

#### Scenario: Only control of the external identity

- **WHEN** a code is held by someone who cannot present a valid session for any account
- **THEN** no binding is written and the code remains unconsumed

#### Scenario: Only control of the account

- **WHEN** an authenticated user submits a code that was never issued to them
- **THEN** the redemption fails on the code's own validity and no binding is written

### Requirement: A binding code cannot inherit an occupied identity or account

Redeeming a binding code SHALL be refused when the external identity it names is already bound to a different account, and SHALL be refused when the redeeming account already holds a binding for that provider. Each refusal SHALL leave both the incoming identity and every existing binding exactly as they were, so a code can never move an identity from one account to another and can never silently replace an account's existing binding.

#### Scenario: External identity already bound elsewhere

- **WHEN** a code naming an external identity already bound to another account is redeemed
- **THEN** the redemption is refused, the other account keeps its binding, and the redeeming account gains none

#### Scenario: Redeeming account already bound for that provider

- **WHEN** an account that already holds a binding for the provider redeems a code
- **THEN** the redemption is refused and the account's existing binding is left untouched

#### Scenario: Race between two redemptions of the same identity

- **WHEN** two accounts concurrently redeem codes naming the same external identity
- **THEN** at most one binding is written and the other redemption is refused

### Requirement: A self-declared handle is never corroborating evidence

A handle, username, or display name that an account holder typed about themselves SHALL NOT be treated as evidence tying an external identity to that account. Matching such a value SHALL NOT cause, suggest as automatic, or partially complete a binding. Only the provider's stable identifier, established through the provider's own authenticated flow or through a code proving control of the provider's channel, SHALL establish a binding.

#### Scenario: Declared handle matches an incoming identity

- **WHEN** an incoming external identity's handle equals the handle an unlinked account declared about itself
- **THEN** no binding is created, no account is matched, and the person is directed through a flow that proves control

#### Scenario: Handle was transferred to another person

- **WHEN** a handle previously declared by an account is later held by a different external identity
- **THEN** that identity gains no binding, no session, and no access to the account that declared it

#### Scenario: Someone declares another person's handle

- **WHEN** an account declares a handle belonging to a person who does not control that account
- **THEN** the real holder of that handle can still bind their own identity to their own account, and the declaring account gains nothing

### Requirement: Binding refusals are recorded for the operator

When a binding attempt is refused because the identity is occupied, the account is already bound, the account is unusable, or the session is no longer valid, the system SHALL record an operator-visible entry naming the refusal reason, the account, and the external identity involved. The entry SHALL NOT contain the code or any credential.

#### Scenario: Occupied identity refusal recorded

- **WHEN** a redemption is refused because the external identity is already bound elsewhere
- **THEN** an entry records the reason, the redeeming account, and the external identity

#### Scenario: Refusal entry carries no secret

- **WHEN** any binding refusal is recorded
- **THEN** the entry contains no code, token, key, or password
