## MODIFIED Requirements

### Requirement: Creating a SePay top-up order

The system SHALL let an authenticated user create a pending SePay top-up order for a requested top-up amount, and SHALL let the Telegram bot integration surface create the same kind of order on behalf of a Telegram user bound to an account. The order SHALL record the credited amount, the payable amount in Vietnamese Dong, a unique trade number, a unique transfer memo, the SePay payment provider, the creation time, and an expiry time. The response SHALL return everything the client needs to pay: the transfer memo, the payable VND amount, the destination bank account number, bank code, account holder name, the VietQR image URL, the trade number, and the expiry timestamp.

An order created through the bot integration surface SHALL be indistinguishable from a console-created order in every respect that affects money: the same minimum and maximum amount bounds, the same wallet-capacity check, the same memo format and uniqueness guarantee, the same currency conversion, the same expiry window, and the same webhook settlement path. Creating an order through the bot SHALL NOT introduce an alternative payment provider.

#### Scenario: Valid top-up request

- **WHEN** an authenticated user requests a SePay top-up for an amount at or above the configured minimum and within the wallet capacity limit
- **THEN** the system creates a pending order and returns the transfer memo, payable VND amount, destination bank details, VietQR image URL, trade number, and expiry timestamp

#### Scenario: Amount below the configured minimum

- **WHEN** an authenticated user requests a SePay top-up below the configured minimum top-up amount
- **THEN** the system rejects the request with an error naming the minimum, and creates no order

#### Scenario: Amount would exceed the wallet quota ceiling

- **WHEN** an authenticated user requests a SePay top-up whose credited quota would push the user's wallet above the maximum representable wallet quota
- **THEN** the system rejects the request with a quota-limit error and creates no order

#### Scenario: Amount above the per-order upper bound

- **WHEN** an authenticated user requests a SePay top-up above the maximum single-order amount
- **THEN** the system rejects the request without creating an order and without producing a saturated or negative payable amount

#### Scenario: SePay not configured

- **WHEN** an authenticated user requests a SePay top-up while SePay is disabled or incompletely configured
- **THEN** the system rejects the request with a configuration error and creates no order

#### Scenario: Bot-initiated order for a linked Telegram user

- **WHEN** the bot integration surface requests a top-up for a Telegram user bound to an account, for an amount at or above the configured minimum and within the wallet capacity limit
- **THEN** the system creates a pending SePay order owned by that account and returns the same payment details a console-created order returns

#### Scenario: Bot-initiated order fails the same bounds

- **WHEN** the bot integration surface requests a top-up below the minimum, above the per-order maximum, or beyond the wallet capacity limit
- **THEN** the request is rejected on exactly the same grounds as the equivalent console request, and no order is created

#### Scenario: Bot-initiated order for an unlinked Telegram user

- **WHEN** the bot integration surface requests a top-up for a Telegram user bound to no account
- **THEN** no order is created and the caller receives the not-linked indication rather than a payment error

#### Scenario: Bot-initiated order settles through the SePay webhook

- **WHEN** a matching bank transfer arrives for an order created through the bot integration surface
- **THEN** the owning account's wallet is credited exactly once by the same settlement path that serves console-created orders

### Requirement: Order status polling

An authenticated user SHALL be able to query the current status of their own SePay order by trade number and receive its status, payable amount, memo, and expiry time. A user SHALL NOT be able to read another user's order. The Telegram bot integration surface SHALL be able to query an order only on behalf of the Telegram user bound to the account that owns it, under the same ownership rule.

#### Scenario: Owner polls a pending order

- **WHEN** the owning user queries an order that is still pending and unexpired
- **THEN** the response reports the pending status with the remaining time to expiry

#### Scenario: Owner polls a settled order

- **WHEN** the owning user queries an order after the webhook credited it
- **THEN** the response reports the successful status and the client stops polling and refreshes the displayed balance

#### Scenario: Non-owner polls an order

- **WHEN** an authenticated user queries a trade number belonging to a different user
- **THEN** the system responds with a not-found error and discloses no order details

#### Scenario: Bot polls an order for its owner

- **WHEN** the bot integration surface queries an order using the Telegram identifier bound to the owning account
- **THEN** the order's status, payable amount, memo, and expiry are returned

#### Scenario: Bot polls an order for a non-owner

- **WHEN** the bot integration surface queries an order using a Telegram identifier bound to a different account
- **THEN** the system responds with a not-found error and discloses nothing about whether that trade number exists
