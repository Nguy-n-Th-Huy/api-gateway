# telegram/bot-events Specification

## Purpose
Defines how the gateway pushes account events to an externally hosted bot so a customer learns that their transfer landed without having to ask: which events are sent, how the bot proves the delivery came from the gateway, and the guarantee that a failed delivery never damages a payment or a binding.

## Requirements

### Requirement: Outbound event delivery is opt-in and configured by an administrator

The system SHALL deliver events to the bot only when an administrator has configured a callback address and a signing secret and enabled the integration. While any of these is missing, the system SHALL emit no outbound events and SHALL continue to operate normally.

#### Scenario: Delivery configured and enabled

- **WHEN** a callback address and signing secret are configured and the integration is enabled
- **THEN** qualifying events are delivered to that address

#### Scenario: Callback address not configured

- **WHEN** an event occurs while no callback address is configured
- **THEN** no delivery is attempted, no error surfaces to the user, and the originating operation completes normally

#### Scenario: Signing secret not configured

- **WHEN** an event occurs while a callback address is set but no signing secret is
- **THEN** no delivery is attempted, because an unsigned event would be indistinguishable from a forged one

### Requirement: The delivered event set

The system SHALL deliver an event when a top-up order is credited, when a pending top-up order is marked expired, and when an account completes a Telegram link. Each event SHALL identify the Telegram account it concerns so the bot can address the right chat, and SHALL carry the facts the bot needs to compose its message without calling back for them.

#### Scenario: Top-up credited

- **WHEN** a top-up order belonging to a linked account is credited
- **THEN** an event is delivered identifying the Telegram account, the trade number, the credited amount, and the resulting balance

#### Scenario: Top-up expired

- **WHEN** a pending top-up order belonging to a linked account is marked expired
- **THEN** an event is delivered identifying the Telegram account and the trade number

#### Scenario: Account linked

- **WHEN** an account completes a Telegram link
- **THEN** an event is delivered identifying the Telegram account and the console username

#### Scenario: Event concerns an unlinked account

- **WHEN** a qualifying operation completes for an account with no Telegram binding
- **THEN** no event is delivered, because there is no chat to address

### Requirement: Events are signed so the bot can reject forgeries

The system SHALL sign every delivered event with the configured secret and present the signature to the receiver, so a bot exposed on the public internet can reject a payload that did not come from the gateway. The secret SHALL NOT appear in the payload.

#### Scenario: Bot verifies a genuine delivery

- **WHEN** the bot receives an event and computes the signature over the payload with the shared secret
- **THEN** the computed signature matches the presented one

#### Scenario: Bot rejects a forged delivery

- **WHEN** a third party posts a fabricated payload to the bot's callback address
- **THEN** the presented signature does not match and the bot can reject the delivery

#### Scenario: Payload tampered in transit

- **WHEN** any byte of a signed payload is altered
- **THEN** the signature no longer matches

### Requirement: Every event carries a stable identifier for idempotency

The system SHALL assign each event an identifier that is stable across retries of that same event and distinct between different events, so a receiver that acts on an event twice can recognize the duplicate and act once.

#### Scenario: Retry carries the same identifier

- **WHEN** a delivery is retried after a transport failure
- **THEN** the retried payload carries the same event identifier as the original attempt

#### Scenario: Distinct events carry distinct identifiers

- **WHEN** two different top-up orders are credited
- **THEN** their events carry different identifiers

#### Scenario: Receiver deduplicates

- **WHEN** the bot receives the same event identifier twice
- **THEN** it can determine that it has already acted and notify the customer only once

### Requirement: Delivery is retried, bounded, and observable

The system SHALL retry a failed delivery with increasing delay up to a bounded number of attempts, and SHALL record an operator-visible entry when an event is ultimately abandoned. Retries SHALL NOT continue indefinitely and SHALL NOT accumulate without limit.

#### Scenario: Transient failure then success

- **WHEN** the first delivery attempt fails and a later attempt succeeds
- **THEN** the event is delivered exactly once from the receiver's perspective, aided by the stable event identifier

#### Scenario: Receiver down for the whole retry window

- **WHEN** every attempt fails
- **THEN** delivery stops after the bounded attempts and an operator-visible entry records the abandoned event

#### Scenario: Receiver is slow

- **WHEN** the callback address does not respond within the delivery timeout
- **THEN** the attempt is treated as failed and retried under the same bound

### Requirement: Delivery failure never damages the originating operation

The system SHALL treat outbound event delivery as strictly secondary to the operation that produced the event. A slow, failing, unreachable, or hostile callback address SHALL NOT delay, roll back, duplicate, or otherwise alter a wallet credit, an order state transition, or an account binding.

#### Scenario: Callback unreachable during settlement

- **WHEN** a wallet credit completes while the callback address is unreachable
- **THEN** the wallet is credited exactly once, the order is marked settled, and the payment webhook still acknowledges normally

#### Scenario: Callback hangs

- **WHEN** the callback address accepts a connection and never responds
- **THEN** the settlement path is not blocked waiting for it

#### Scenario: Callback returns an error after a successful credit

- **WHEN** the credit succeeds and the callback returns an error status
- **THEN** the credit is not reversed and the order is not returned to pending

### Requirement: Events disclose no key material and no credentials

The system SHALL NOT include an API key, a key fragment, the bot service key, the signing secret, or a session token in any event payload.

#### Scenario: Payload inspected for secrets

- **WHEN** any delivered event payload is inspected
- **THEN** it contains no API key, no key fragment, no service key, no signing secret, and no session token
