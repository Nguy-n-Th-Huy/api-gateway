# payments/sepay-subscription Specification

## Purpose
Lets users buy a subscription plan by Vietnamese domestic bank transfer through SePay, reusing the same pending-order, transfer-memo, and webhook mechanics as wallet top-up, while keeping the existing wallet-balance purchase path unchanged.

## Requirements

### Requirement: Subscription payment options

The subscription purchase surface SHALL offer exactly two payment paths: payment from the user's wallet balance, and SePay bank transfer. SePay SHALL be offered only when payment compliance is confirmed, the SePay integration is enabled, and every required SePay setting is present. No other external gateway SHALL be offered.

#### Scenario: Both paths available

- **WHEN** a user opens the purchase dialog for an enabled plan while SePay is fully configured and compliance is confirmed
- **THEN** the dialog offers wallet-balance payment and SePay bank transfer, and nothing else

#### Scenario: SePay unavailable

- **WHEN** a user opens the purchase dialog while SePay is disabled or incompletely configured
- **THEN** the dialog offers only wallet-balance payment

#### Scenario: Wallet balance insufficient

- **WHEN** a user whose wallet balance is below the plan price opens the purchase dialog while SePay is available
- **THEN** the wallet-balance option is presented as unavailable with a reason, and SePay remains selectable

### Requirement: Creating a SePay subscription order

The system SHALL let an authenticated user create a pending subscription order paid via SePay for an enabled plan. The order SHALL record the plan, the user, the payable amount in Vietnamese Dong, a unique trade number, a unique transfer memo, the SePay payment provider, and an expiry time. The response SHALL return the transfer memo, payable VND amount, destination bank details, VietQR image URL, trade number, and expiry timestamp, in the same shape as a SePay top-up order.

#### Scenario: Valid purchase request

- **WHEN** an authenticated user requests a SePay subscription order for an enabled plan
- **THEN** the system creates a pending subscription order and returns the payment details needed to complete the bank transfer

#### Scenario: Plan disabled or missing

- **WHEN** an authenticated user requests a SePay subscription order for a plan that is disabled or does not exist
- **THEN** the system rejects the request and creates no order

#### Scenario: Plan price converts to an unusable amount

- **WHEN** the plan price converts to zero or fewer Dong under the configured price per USD
- **THEN** the system rejects the request with an error and creates no order

### Requirement: Activating a subscription from an incoming transfer

On an authenticated webhook describing an incoming transfer whose content contains a pending subscription order's memo, the system SHALL verify that the order belongs to the SePay provider, is still pending, and that the transferred amount is at least the payable amount, then atomically mark the order successful and activate the user's subscription entitlement in a single database transaction under a row lock on the order. The activation SHALL write a purchase log entry naming the plan, paid amount, and payment provider.

#### Scenario: Matching transfer activates the subscription

- **WHEN** SePay reports an incoming transfer matching a pending subscription order for at least the payable amount
- **THEN** the order becomes successful, the user's subscription entitlement is created or extended per the plan, and a purchase log entry is written

#### Scenario: Underpaid subscription transfer

- **WHEN** SePay reports an incoming transfer matching a pending subscription order for less than the payable amount
- **THEN** the system leaves the order pending, activates nothing, and records a warning identifying the order and the shortfall

#### Scenario: Duplicate delivery for a settled subscription order

- **WHEN** SePay redelivers a transfer notification for a subscription order that is already successful
- **THEN** the system activates nothing further and responds with success

#### Scenario: Concurrent deliveries for one subscription order

- **WHEN** two application instances process deliveries for the same subscription order at the same time
- **THEN** exactly one delivery activates the subscription and the other observes the order as already settled

### Requirement: Memo routing between top-up and subscription orders

A single webhook endpoint SHALL serve both wallet top-up and subscription orders. The system SHALL resolve an incoming transfer to at most one order across both order types, and SHALL never settle two orders from one transfer.

#### Scenario: Transfer resolves to a subscription order

- **WHEN** an incoming transfer's content matches a pending subscription order's memo
- **THEN** the system settles that subscription order and touches no top-up order

#### Scenario: Transfer resolves to a top-up order

- **WHEN** an incoming transfer's content matches a pending top-up order's memo
- **THEN** the system settles that top-up order and touches no subscription order

#### Scenario: Content matches more than one memo

- **WHEN** an incoming transfer's content contains text matching more than one pending order's memo
- **THEN** the system settles no order, records an error naming every candidate order and the transfer reference code, and responds with success for manual reconciliation

### Requirement: Subscription order expiry

A pending SePay subscription order SHALL expire after the configured expiry window, SHALL be marked expired by the same background sweep that expires top-up orders, and SHALL NOT be activated by any later webhook.

#### Scenario: Subscription order expires unpaid

- **WHEN** a pending subscription order reaches its expiry time with no matching transfer
- **THEN** the sweep marks the order expired and the purchase dialog reports it as expired

#### Scenario: Transfer arrives after subscription order expiry

- **WHEN** a matching transfer arrives for an already expired subscription order
- **THEN** the system activates nothing and records the transfer with its reference code for manual reconciliation

### Requirement: Wallet-balance purchase path is unchanged

Purchasing a subscription from the wallet balance SHALL keep its existing behavior: it debits the wallet, activates the entitlement immediately, and records the payment method and provider as balance. This change SHALL NOT alter its endpoint, request shape, or response shape.

#### Scenario: Balance purchase after the change

- **WHEN** a user with sufficient balance buys a plan using wallet balance after this change ships
- **THEN** the wallet is debited, the entitlement is activated immediately, and the recorded payment method and provider are unchanged from before the change
