# payments/gateway-administration Specification

## Purpose
Defines the administrator-facing configuration surface for payments after consolidating on a single gateway: SePay credentials and limits, the shared pricing settings that apply to every order, and the compliance gate that must be confirmed before any payment feature becomes usable.

## Requirements

### Requirement: SePay configuration settings

Administrators SHALL be able to configure SePay through system settings with the following fields: integration enabled flag, bank account number, bank code, account holder name, webhook API key, minimum top-up amount, and order expiry window in minutes. Settings SHALL persist across restarts and take effect without a restart.

#### Scenario: Saving a complete configuration

- **WHEN** an administrator saves SePay settings with every required field populated
- **THEN** the settings persist, SePay becomes available to users without a restart, and the top-up info endpoint begins reporting it as available

#### Scenario: Enabling with an incomplete configuration

- **WHEN** an administrator enables SePay while a required field is empty
- **THEN** the save is rejected with a message naming the missing field, or the integration is stored as unavailable and reported as unavailable to users; in neither case does an incompletely configured gateway accept orders

#### Scenario: Invalid expiry window

- **WHEN** an administrator saves an expiry window that is zero, negative, or above the allowed maximum
- **THEN** the save is rejected with a validation message and the previous value is retained

### Requirement: Webhook API key handling

The SePay webhook API key SHALL be treated as a secret: it SHALL NOT be returned in any response available to non-administrators, SHALL be masked in the administration UI once stored, and submitting an empty value SHALL leave the stored key unchanged rather than clearing it.

#### Scenario: Key masked after saving

- **WHEN** an administrator reopens SePay settings after saving a key
- **THEN** the key field shows a masked placeholder rather than the stored value

#### Scenario: Saving without retyping the key

- **WHEN** an administrator changes another SePay field and saves while leaving the key field empty
- **THEN** the previously stored key is retained and the webhook keeps working

#### Scenario: Key never exposed to users

- **WHEN** any non-administrator endpoint returns payment configuration
- **THEN** the response contains no webhook API key

### Requirement: Shared pricing settings

The general payment settings SHALL continue to expose the local-currency price per USD, the minimum top-up amount, the preset top-up amount options, and the amount-based discount tiers, and these SHALL apply to SePay orders. Under SePay the local currency is Vietnamese Dong, so the price setting means Dong charged per one USD of credited balance, and its default value SHALL be 1000 — one thousand Dong buys one USD of balance. The general settings SHALL no longer contain a payment-method list, because SePay is the only gateway and needs no per-method routing.

#### Scenario: Default price on a deployment that has never configured one

- **WHEN** SePay is enabled on a deployment where no price per USD has been set
- **THEN** the effective price is 1000 Dong per USD, and a $10 top-up is payable as 10,000 Dong

#### Scenario: Price change applies to new orders

- **WHEN** an administrator changes the local-currency price per USD
- **THEN** subsequently created SePay orders use the new price, while already pending orders keep the amount they were created with

#### Scenario: Price is presented in Dong

- **WHEN** an administrator opens the general payment settings under SePay
- **THEN** the price field is labelled as Dong per USD, so the entered number is unambiguous

#### Scenario: Discount tier applies

- **WHEN** a user requests a top-up amount that matches a configured discount tier
- **THEN** the payable amount reflects the discount

#### Scenario: Payment method list is gone

- **WHEN** an administrator opens the general payment settings
- **THEN** no payment-method list editor is present and no payment-method JSON is stored or sent to clients

### Requirement: Payment settings surface contains only SePay

The payment settings screen SHALL present exactly two areas: shared general settings and SePay settings. No configuration for Epay, Stripe, Creem, Waffo, or Waffo Pancake SHALL be reachable, and no stored option belonging to those gateways SHALL be readable or writable through the settings API.

#### Scenario: Settings screen after the change

- **WHEN** an administrator opens payment settings
- **THEN** exactly one gateway tab is shown, for SePay, alongside the general settings

#### Scenario: Attempting to write a removed gateway option

- **WHEN** a request tries to write a configuration option belonging to a removed gateway
- **THEN** the system rejects it as an unknown option and stores nothing

### Requirement: Payment compliance gate

The existing payment compliance confirmation SHALL continue to gate all payment functionality. Until an administrator confirms the current compliance terms version, the top-up info endpoint SHALL report no available payment provider and SHALL disable redemption, and SePay order creation SHALL be rejected.

#### Scenario: Compliance unconfirmed blocks order creation

- **WHEN** a user attempts to create a SePay order while compliance is unconfirmed
- **THEN** the request is rejected and no order is created

#### Scenario: Confirming compliance enables payments

- **WHEN** an administrator confirms the current compliance terms version while SePay is fully configured
- **THEN** users can create SePay orders and the top-up info endpoint reports SePay as available

### Requirement: Administrative order visibility and manual completion

Administrators SHALL be able to list and search all top-up orders, including orders whose payment provider is one of the removed gateways, and SHALL be able to manually complete a pending order by trade number. Manual completion SHALL remain idempotent and SHALL credit the correct quota for the order.

#### Scenario: Listing mixed-provider history

- **WHEN** an administrator lists top-up orders after the change
- **THEN** orders created by removed gateways are listed with their original provider values alongside SePay orders

#### Scenario: Manually completing a stranded legacy order

- **WHEN** an administrator manually completes a pending order that was created by a removed gateway and confirmed paid out of band
- **THEN** the order becomes successful, the user is credited exactly once, and a log entry attributes the completion to the administrator

#### Scenario: Manual completion repeated

- **WHEN** an administrator submits manual completion twice for the same trade number
- **THEN** the user is credited exactly once
