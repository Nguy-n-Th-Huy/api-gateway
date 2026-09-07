# telegram/account-link Specification

## Purpose
Covers how a person's Telegram handle enters the system and how it becomes trustworthy: the self-declared handle captured at registration, the administrator toggle that makes it mandatory, the one-time code that carries a linking intent from a chat to an authenticated browser session, and the moment a claimed handle is replaced by a verified one.

## Requirements

### Requirement: Accounts carry a Telegram handle distinct from the Telegram identity

The system SHALL store a Telegram handle on each account, separate from the verified Telegram account identifier. The handle SHALL be normalized on write by trimming surrounding whitespace, removing a leading `@`, and lower-casing it. The handle SHALL NOT be unique across accounts and SHALL NOT be usable as a lookup key for authentication, binding, or authorization, because a Telegram handle can be changed, released, or transferred between Telegram accounts. The verified Telegram account identifier SHALL remain the only identity key.

#### Scenario: Handle is normalized on write

- **WHEN** a handle is submitted as `"  @JohnDoe  "`
- **THEN** it is stored as `johndoe`

#### Scenario: Two accounts declare the same handle

- **WHEN** two different accounts each declare the handle `johndoe`
- **THEN** both are stored, because the handle is not an identity key

#### Scenario: Handle is never an authentication key

- **WHEN** any authentication, binding, or authorization decision is made
- **THEN** it uses the verified Telegram account identifier and never the stored handle

### Requirement: Telegram handle format validation

The system SHALL accept a Telegram handle only when, after normalization, it is 5 to 32 characters long, contains only letters, digits, and underscores, and begins with a letter. A handle failing any of these SHALL be rejected with a message naming the rule that failed.

#### Scenario: Valid handle

- **WHEN** the normalized handle is `john_doe99`
- **THEN** it is accepted

#### Scenario: Too short

- **WHEN** the normalized handle is `abcd`
- **THEN** it is rejected

#### Scenario: Too long

- **WHEN** the normalized handle exceeds 32 characters
- **THEN** it is rejected

#### Scenario: Illegal character

- **WHEN** the normalized handle contains a hyphen, a dot, or a space
- **THEN** it is rejected

#### Scenario: Starts with a digit

- **WHEN** the normalized handle is `1johndoe`
- **THEN** it is rejected

### Requirement: A stored handle is untrusted until the account is linked

The system SHALL treat a handle supplied by a person as self-declared and unverified. It SHALL NOT be used to identify, match, or merge accounts. When an account completes a Telegram link, the system SHALL overwrite the stored handle with the handle reported by Telegram for the linked account, at which point it is a verified snapshot for display.

#### Scenario: Self-declared handle does not grant anything

- **WHEN** an account declares the handle of a Telegram account it does not control
- **THEN** no binding, no privilege, and no account match results from that declaration

#### Scenario: Verified handle replaces the declared one

- **WHEN** an account completes a Telegram link and Telegram reports a handle for that account
- **THEN** the stored handle is overwritten with the reported handle

#### Scenario: Linked Telegram account has no handle

- **WHEN** an account completes a Telegram link and the linked Telegram account has no handle set
- **THEN** the link still succeeds and the stored handle is cleared rather than left showing an unverified value

### Requirement: Administrators can require a Telegram handle at registration

The system SHALL provide an administrator option, enabled by default, that makes the Telegram handle mandatory during registration. While enabled, a registration without a valid handle SHALL be rejected and no account SHALL be created. While disabled, the handle SHALL be optional and a registration omitting it SHALL succeed.

#### Scenario: Requirement enabled and handle omitted

- **WHEN** the option is enabled and a registration arrives without a Telegram handle
- **THEN** the registration is rejected and no account is created

#### Scenario: Requirement enabled and handle invalid

- **WHEN** the option is enabled and a registration supplies a handle that fails format validation
- **THEN** the registration is rejected and no account is created

#### Scenario: Requirement enabled and handle valid

- **WHEN** the option is enabled and a registration supplies a valid handle
- **THEN** the account is created with the normalized handle stored

#### Scenario: Requirement disabled

- **WHEN** the option is disabled and a registration arrives without a Telegram handle
- **THEN** the account is created and no handle is stored

#### Scenario: Registration form follows the option

- **WHEN** the option is enabled
- **THEN** the sign-up screen presents the Telegram handle field as required, and when the option is disabled the field is not presented as required

### Requirement: A person can correct their declared handle

The system SHALL let a signed-in user change their stored Telegram handle, applying the same normalization and format validation used at registration. Changing the handle SHALL NOT create, move, or remove any Telegram binding.

#### Scenario: User corrects a typo

- **WHEN** a signed-in user submits a corrected, valid handle
- **THEN** the normalized value replaces the stored one and the account's Telegram binding is unchanged

#### Scenario: User submits an invalid handle

- **WHEN** a signed-in user submits a handle that fails validation
- **THEN** the change is rejected and the previously stored handle is retained

### Requirement: Link codes carry a linking intent from a chat to a browser session

The system SHALL issue, on request from the bot integration surface for a given Telegram account identifier, a short-lived one-time link code together with its expiry and the address at which it is redeemed. The code SHALL be single-use and SHALL expire within a short window. Issuing a code SHALL NOT by itself create, change, or remove any binding.

#### Scenario: Code issued for an unlinked Telegram account

- **WHEN** the bot requests a link code for a Telegram account identifier that is bound to no account
- **THEN** a single-use code is returned with its expiry and redemption address, and no binding is created

#### Scenario: Code issued for an already-linked Telegram account

- **WHEN** the bot requests a link code for a Telegram account identifier already bound to an account
- **THEN** the request is refused with an already-linked indication and no code is issued

#### Scenario: Issuing a code changes nothing

- **WHEN** a code is issued and never redeemed
- **THEN** no account gains, loses, or changes a Telegram binding

### Requirement: Redemption requires an authenticated session

The system SHALL redeem a link code only for a signed-in user, through a website endpoint that consumes the code and binds the Telegram account identifier the code was issued for to the redeeming account. Redemption SHALL NOT be reachable from the bot integration surface, so possession of the bot service key alone can never complete a link.

#### Scenario: Signed-in user redeems a valid code

- **WHEN** a signed-in user submits an unexpired, unused code
- **THEN** the Telegram account identifier the code was issued for is bound to that user's account, the stored handle is updated to the verified value, and the code is consumed

#### Scenario: Anonymous redemption attempt

- **WHEN** a caller without a valid session submits a code
- **THEN** the request is rejected and the code remains unconsumed

#### Scenario: Redemption is absent from the bot surface

- **WHEN** a caller holding the bot service key attempts to redeem a code through the bot integration surface
- **THEN** no such route exists and no binding is created

### Requirement: Link codes are single-use and expire

The system SHALL reject a link code that has already been consumed, has expired, or does not exist, and SHALL make these outcomes indistinguishable to the caller so codes cannot be probed. A rejected redemption SHALL leave every binding unchanged.

#### Scenario: Code reused

- **WHEN** a code that was already redeemed is submitted again
- **THEN** the request is rejected and no second binding is created

#### Scenario: Code expired

- **WHEN** a code is submitted after its expiry
- **THEN** the request is rejected and no binding is created

#### Scenario: Code never existed

- **WHEN** a fabricated code is submitted
- **THEN** the response is indistinguishable from the expired and already-used cases

#### Scenario: Concurrent redemption of one code

- **WHEN** the same valid code is redeemed twice concurrently
- **THEN** exactly one redemption succeeds and the other is rejected

### Requirement: Redemption obeys the existing binding safety rules

The system SHALL apply to link-code redemption every safeguard that governs the existing Telegram binding path: it SHALL refuse when the Telegram account is already bound to another account, refuse when the redeeming account already holds a Telegram binding, refuse when the redeeming account is disabled or deleted, and refuse when the redeeming session is no longer valid. Each refusal SHALL leave every binding unchanged.

#### Scenario: Telegram account already bound elsewhere

- **WHEN** a user redeems a code for a Telegram account already bound to a different account
- **THEN** the redemption is refused and neither account's binding changes

#### Scenario: Redeeming account already linked

- **WHEN** a user who already has a Telegram binding redeems a code
- **THEN** the redemption is refused and the existing binding is retained

#### Scenario: Redeeming account disabled

- **WHEN** the redeeming account is disabled between code issuance and redemption
- **THEN** the redemption is refused and no binding is created

#### Scenario: Session revoked before redemption

- **WHEN** the redeeming session has been revoked
- **THEN** the redemption is refused and no binding is created

### Requirement: A declared handle never causes an automatic link

The system SHALL NOT bind a Telegram account to an account because the Telegram account's handle equals that account's stored handle. Any match SHALL remain informational only, so that declaring another person's handle cannot capture their Telegram identity and holding a transferred handle cannot inherit an account.

#### Scenario: Handle matches an unlinked account

- **WHEN** a Telegram account whose handle equals an unlinked account's declared handle interacts with the bot
- **THEN** no binding is created and the person is directed through the link-code flow

#### Scenario: Handle was transferred between Telegram accounts

- **WHEN** a Telegram handle previously declared by an account is transferred to a different Telegram account
- **THEN** the new holder gains no binding, no session, and no access to that account
