# payments/legacy-gateway-removal Specification

## Purpose
Defines the removal contract for the five retired payment gateways: which endpoints cease to exist, what callers observe when they hit them, how historical records stay readable, and how orders left pending at cutover are handled.

## Requirements

### Requirement: Retired checkout and configuration endpoints

The system SHALL no longer expose any endpoint that creates a payment through Epay, Stripe, Creem, Waffo, or Waffo Pancake, nor any administration endpoint that configures them. This includes the Epay checkout and amount endpoints, the Stripe, Creem, Waffo, and Waffo Pancake checkout and amount endpoints, the corresponding subscription checkout endpoints, and the Waffo Pancake catalog, pairing, save, and subscription-product option endpoints.

#### Scenario: Retired checkout endpoint is called

- **WHEN** a client calls a retired checkout endpoint with a valid session
- **THEN** the system responds with HTTP 404 and creates no order

#### Scenario: Retired administration endpoint is called

- **WHEN** an administrator client calls a retired gateway configuration endpoint
- **THEN** the system responds with HTTP 404 and changes no stored configuration

### Requirement: Retired webhook endpoints

The system SHALL no longer expose webhook or notify endpoints for the retired gateways, including the Stripe, Creem, Waffo, and Waffo Pancake webhooks, the Epay notify endpoint in both its GET and POST forms, and the Epay subscription notify and return endpoints. A retired webhook SHALL NOT settle any order under any circumstance.

#### Scenario: Retired webhook receives a delivery

- **WHEN** a retired gateway posts a settlement notification to its former webhook path
- **THEN** the system responds with HTTP 404, credits no wallet, and activates no subscription

#### Scenario: Webhook availability report

- **WHEN** the webhook availability diagnostic is requested
- **THEN** it reports only the SePay webhook and no retired gateway

### Requirement: Historical records remain readable

Existing top-up and subscription order records whose payment provider is a retired gateway SHALL remain stored, listable, searchable, and readable in both user-facing history and administrator views, showing their original provider and method values. No migration SHALL delete, rewrite, or re-label those rows.

#### Scenario: User views legacy top-up history

- **WHEN** a user opens their top-up history containing orders paid through a retired gateway
- **THEN** those orders are listed with their original amounts, statuses, and provider labels

#### Scenario: Administrator searches legacy orders

- **WHEN** an administrator searches all top-up orders by a legacy trade number
- **THEN** the matching order is returned with its original provider value

#### Scenario: Legacy provider label has no translation gap

- **WHEN** the UI renders an order whose provider is a retired gateway
- **THEN** it displays a readable label for that provider rather than an empty cell or a raw translation key

### Requirement: Orders left pending at cutover

Orders that are still pending at the time this change ships SHALL remain pending and SHALL NOT be settleable by any automated path, because their gateways no longer report to the system. The administrator manual-completion path SHALL remain the only way to settle them, and the background expiry sweep SHALL NOT silently expire them without an operator-visible record.

#### Scenario: Legacy pending order after deployment

- **WHEN** an order created by a retired gateway is still pending after this change is deployed
- **THEN** no automated path settles it, and it remains visible to administrators as pending

#### Scenario: Operator settles a legacy pending order

- **WHEN** an operator confirms out of band that a legacy pending order was paid and manually completes it
- **THEN** the user is credited exactly once and the action is attributed to the administrator in the log

### Requirement: Removed gateway data is absent from client payloads

Endpoints that describe payment capability to clients SHALL contain no field, flag, product list, or payment-method entry belonging to a retired gateway. Clients SHALL have no way to discover a retired gateway as available.

#### Scenario: Top-up info payload after the change

- **WHEN** a client fetches top-up info
- **THEN** the payload contains no enablement flag, product list, or payment-method entry for Epay, Stripe, Creem, Waffo, or Waffo Pancake

#### Scenario: Frontend has no retired gateway path

- **WHEN** the wallet and subscription screens are rendered under any configuration
- **THEN** no retired gateway option, icon, or settings panel can be reached

### Requirement: Retired dependencies are removed from the build

The retired gateway SDKs SHALL be removed from the Go module requirements, and the project SHALL build and pass its test suite without them.

#### Scenario: Build without retired SDKs

- **WHEN** the project is built after this change
- **THEN** the Epay, Stripe, Waffo, and Waffo Pancake SDKs appear in neither the module requirements nor the resolved dependency graph, and the build succeeds
