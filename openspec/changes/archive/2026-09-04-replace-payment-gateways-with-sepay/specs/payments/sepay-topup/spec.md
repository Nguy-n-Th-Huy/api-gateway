## Purpose

Lets users top up their wallet balance by Vietnamese domestic bank transfer through SePay: the system issues a pending order carrying a unique transfer memo, shows the user a VietQR payment code, and credits the wallet when SePay reports the matching incoming transfer.

## ADDED Requirements

### Requirement: SePay top-up availability is advertised to the client

The top-up info endpoint SHALL report whether SePay top-up is available, together with the parameters the client needs to render the payment form: minimum top-up amount, local-currency price per USD, preset amount options, and discount tiers. SePay SHALL be reported as available only when payment compliance has been confirmed by an administrator, the SePay integration is enabled, and every required SePay setting (bank account number, bank code, account holder, webhook API key) is non-empty.

#### Scenario: SePay fully configured and compliance confirmed

- **WHEN** an authenticated user requests top-up info while SePay is enabled, fully configured, and compliance is confirmed
- **THEN** the response reports SePay as the available online payment provider, including its minimum top-up amount and the local-currency price per USD

#### Scenario: SePay enabled but incompletely configured

- **WHEN** an authenticated user requests top-up info while SePay is enabled but at least one required SePay setting is empty
- **THEN** the response reports SePay as unavailable and the client offers no online top-up entry point

#### Scenario: Payment compliance not confirmed

- **WHEN** an authenticated user requests top-up info while payment compliance has not been confirmed
- **THEN** the response reports SePay as unavailable and exposes no payment method list, matching the pre-existing compliance gate

### Requirement: Creating a SePay top-up order

The system SHALL let an authenticated user create a pending SePay top-up order for a requested top-up amount. The order SHALL record the credited amount, the payable amount in Vietnamese Dong, a unique trade number, a unique transfer memo, the SePay payment provider, the creation time, and an expiry time. The response SHALL return everything the client needs to pay: the transfer memo, the payable VND amount, the destination bank account number, bank code, account holder name, the VietQR image URL, the trade number, and the expiry timestamp.

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

### Requirement: User-entered top-up amount

The user SHALL be able to type any top-up amount they choose, subject only to the configured minimum, the per-order upper bound, and the wallet capacity limit. The configured preset amount options SHALL act as one-tap shortcuts that fill the amount field and remain editable afterwards; they SHALL NOT be the only selectable amounts. The server SHALL validate the submitted amount independently of the presets, so a client that submits an amount matching no preset is accepted when it is within bounds.

#### Scenario: Arbitrary amount typed by the user

- **WHEN** a user types an amount that matches no preset option and is within the configured bounds
- **THEN** the system creates the order for exactly that amount and the payable Dong amount is computed from it

#### Scenario: Preset used as a shortcut

- **WHEN** a user taps a preset amount option
- **THEN** the amount field is filled with that value and the user can still edit it before submitting

#### Scenario: Malformed amount submitted

- **WHEN** a request carries an amount that is not a positive whole number
- **THEN** the system rejects it with a validation error and creates no order

### Requirement: The QR code carries the amount and the memo

The VietQR image issued for an order SHALL encode the destination account number, bank code, the order's payable Dong amount, and the order's transfer memo, so that a banking app scanning it pre-fills both the amount and the transfer description. The amount encoded SHALL be the amount frozen at order creation, so rescanning an existing order's QR never produces a different payable amount.

#### Scenario: Scanning the QR pre-fills the transfer

- **WHEN** a user scans the QR image shown for their order in a banking app
- **THEN** the app pre-fills the destination account, the payable Dong amount, and the transfer memo, leaving the user only to confirm

#### Scenario: QR reflects the user's own amount

- **WHEN** a user creates an order for a self-entered amount
- **THEN** the QR encodes the payable Dong amount derived from that exact amount, not a preset or rounded-up value

#### Scenario: Price changes while an order is open

- **WHEN** an administrator changes the price per USD while a user's order panel is still open
- **THEN** the already-issued QR keeps encoding the amount frozen at creation, and paying it settles the order in full

### Requirement: Transfer memo uniqueness and bank-safe format

Each pending SePay order SHALL carry a transfer memo that is unique among all orders that are pending or already settled, and that consists only of uppercase ASCII letters and digits so that no bank strips or rewrites it in transit. The memo SHALL be the sole key used to match an incoming transfer to an order.

#### Scenario: Memo format

- **WHEN** the system generates a transfer memo for a new order
- **THEN** the memo contains only uppercase ASCII letters and digits and no whitespace or punctuation

#### Scenario: Memo collision

- **WHEN** memo generation produces a value that already belongs to another order
- **THEN** the system generates a different memo rather than reusing it, so that no two orders can ever share a memo

### Requirement: Currency conversion for the payable amount

The payable amount SHALL be derived from the requested top-up amount using the configured price in Vietnamese Dong per USD (default 1000, meaning one thousand Dong buys one USD of balance), the user's top-up group ratio, and any configured preset discount, and SHALL be stored and displayed as a whole number of Vietnamese Dong. Conversion SHALL never yield a zero, negative, or saturated amount; when it would, the request SHALL be rejected instead.

#### Scenario: Standard conversion

- **WHEN** a user requests a top-up amount with a configured price per USD, a group ratio, and an applicable discount tier
- **THEN** the payable VND amount equals the amount multiplied by price, group ratio, and discount, rounded to a whole Dong

#### Scenario: Conversion at the default price

- **WHEN** a user requests a $10 top-up at the default price of 1000 Dong per USD, with a group ratio of 1 and no discount
- **THEN** the payable amount is 10,000 Dong

#### Scenario: Conversion yields an unusably small amount

- **WHEN** the computed payable amount rounds to zero or less Dong
- **THEN** the system rejects the request with an error and creates no order

### Requirement: SePay webhook authentication

The SePay webhook endpoint SHALL accept a request only when it presents the configured webhook API key in the `Authorization` header using the `Apikey <key>` scheme, compared in constant time. Requests with a missing, malformed, or incorrect key SHALL be rejected with HTTP 401 and SHALL NOT alter any order or wallet balance. The endpoint SHALL also be rejected when the SePay integration is disabled.

#### Scenario: Valid API key

- **WHEN** SePay posts a webhook carrying the configured API key in the `Authorization: Apikey <key>` header
- **THEN** the system processes the notification

#### Scenario: Missing or wrong API key

- **WHEN** a request reaches the webhook endpoint without the configured key, or with a wrong key, or with a differently formatted `Authorization` header
- **THEN** the system responds HTTP 401, credits nothing, and records a warning that identifies the caller IP

#### Scenario: Integration disabled

- **WHEN** a webhook arrives while the SePay integration is disabled
- **THEN** the system rejects the request and credits nothing

### Requirement: Crediting a wallet from an incoming transfer

On an authenticated webhook describing an incoming transfer, the system SHALL locate the pending order whose transfer memo appears in the transfer content, verify that the order belongs to the SePay provider and is still pending, verify that the transferred amount is at least the order's payable amount, and then atomically mark the order successful and credit the user's wallet within a single database transaction using a row lock on the order. Crediting SHALL respect the wallet quota ceiling and SHALL write a top-up log entry recording the credited quota, paid amount, and payment provider.

#### Scenario: Matching transfer credits the wallet

- **WHEN** SePay reports an incoming transfer whose content contains the memo of a pending order and whose amount is at least the payable amount
- **THEN** the order becomes successful, the user's wallet quota increases by the ordered quota, a top-up log entry is written, and the endpoint responds with success

#### Scenario: Underpaid transfer

- **WHEN** SePay reports an incoming transfer matching a pending order but for less than the payable amount
- **THEN** the system leaves the order pending, credits nothing, records a warning identifying the order and the shortfall, and responds with success so SePay does not retry indefinitely

#### Scenario: Overpaid transfer

- **WHEN** SePay reports an incoming transfer matching a pending order for more than the payable amount
- **THEN** the system credits exactly the ordered quota, leaves the surplus uncredited, and records a warning identifying the order and the surplus for manual handling

#### Scenario: Outgoing transfer

- **WHEN** SePay reports a transfer whose direction is outgoing
- **THEN** the system credits nothing and responds with success

#### Scenario: No matching order

- **WHEN** SePay reports an incoming transfer whose content matches no known memo
- **THEN** the system credits nothing, records the unmatched transfer with its reference code for reconciliation, and responds with success

#### Scenario: Wallet ceiling reached between order creation and payment

- **WHEN** a matching transfer arrives but crediting the order would push the user's wallet above the maximum representable wallet quota
- **THEN** the system does not credit, does not mark the order successful, and records an error identifying the order for manual handling

### Requirement: Webhook idempotency

Repeated webhook deliveries for the same transfer, and concurrent deliveries matching the same order, SHALL credit the wallet at most once. Idempotency SHALL be enforced in the database, not only in process memory, so that it holds across multiple application instances.

#### Scenario: Duplicate delivery of the same transfer

- **WHEN** SePay delivers the same transfer notification a second time after the order was already credited
- **THEN** the system credits nothing further, leaves the order successful, and responds with success

#### Scenario: Concurrent deliveries across instances

- **WHEN** two application instances process webhook deliveries for the same order at the same time
- **THEN** exactly one delivery credits the wallet and the other observes the order as already settled

### Requirement: Order expiry

A pending SePay order SHALL expire after the configured expiry window. Expired orders SHALL be marked expired by a background sweep and SHALL NOT be credited by any later webhook; a transfer arriving for an expired order SHALL be recorded for manual reconciliation instead of silently discarded.

#### Scenario: Order expires unpaid

- **WHEN** a pending order reaches its expiry time with no matching transfer
- **THEN** the background sweep marks the order expired and the user's payment panel reports it as expired

#### Scenario: Transfer arrives after expiry

- **WHEN** a matching transfer arrives for an order that is already marked expired
- **THEN** the system credits nothing, records the transfer with its reference code and order number for manual reconciliation, and responds with success

### Requirement: Order status polling

An authenticated user SHALL be able to query the current status of their own SePay order by trade number and receive its status, payable amount, memo, and expiry time. A user SHALL NOT be able to read another user's order.

#### Scenario: Owner polls a pending order

- **WHEN** the owning user queries an order that is still pending and unexpired
- **THEN** the response reports the pending status with the remaining time to expiry

#### Scenario: Owner polls a settled order

- **WHEN** the owning user queries an order after the webhook credited it
- **THEN** the response reports the successful status and the client stops polling and refreshes the displayed balance

#### Scenario: Non-owner polls an order

- **WHEN** an authenticated user queries a trade number belonging to a different user
- **THEN** the system responds with a not-found error and discloses no order details

### Requirement: Payment panel user experience

The wallet UI SHALL present a SePay payment panel after an order is created, showing the VietQR image, the payable VND amount, the destination bank account number, bank code, account holder name, and the transfer memo, with the memo and the account number individually copyable. The panel SHALL show a countdown to expiry, poll order status until the order becomes successful or expires, and state explicitly that the memo must be used as the transfer description. All panel text SHALL be internationalized.

#### Scenario: Panel rendered after order creation

- **WHEN** an order is created successfully
- **THEN** the panel shows the QR image, payable amount, bank details, copyable memo, countdown, and the memo-required instruction

#### Scenario: Payment confirmed while the panel is open

- **WHEN** polling reports the order became successful
- **THEN** the panel shows a success state, the wallet balance shown in the UI is refreshed, and polling stops

#### Scenario: Panel reaches expiry

- **WHEN** the countdown reaches zero and polling reports the order expired
- **THEN** the panel shows an expired state, stops polling, and offers to start a new top-up
