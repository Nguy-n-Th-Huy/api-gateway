## Purpose

Defines which currency the interface presents monetary amounts in, the rate used to convert them, and how the administrator's display mode interacts with the visitor's language — so a visitor sees prices in the currency they will actually pay, at the rate they will actually be charged.

## ADDED Requirements

### Requirement: Currency follows the interface language

When the interface language is Vietnamese, every monetary amount SHALL be presented in Vietnamese Dong. When the interface language is English, every monetary amount SHALL be presented in US dollars. The choice SHALL be made from the active interface language, not from a separate setting, and SHALL apply uniformly to model prices, the home-page pricing preview, wallet balances, usage costs and any other amount the interface renders.

#### Scenario: Vietnamese interface

- **WHEN** a visitor has Vietnamese as the active interface language
- **THEN** prices on the pricing page, the home-page pricing preview, and balances in the wallet all render in Dong

#### Scenario: English interface

- **WHEN** a visitor has English as the active interface language
- **THEN** the same amounts render in US dollars

#### Scenario: Language switch

- **WHEN** a visitor switches the interface language while viewing prices
- **THEN** the displayed currency changes with it, without a page reload

### Requirement: Dong amounts use the top-up rate

Amounts presented in Dong SHALL be converted from the internal dollar-denominated value at the Dong-per-dollar rate the platform charges for top-ups, as published in the public status payload. No other rate SHALL be used for display, so that a price shown in Dong equals what a visitor would pay in Dong for the same credit.

#### Scenario: Display matches payment

- **WHEN** the top-up rate is R Dong per dollar and a model costs D dollars per unit
- **THEN** the pricing page shows R × D Dong per unit for that model

#### Scenario: Rate changed by the administrator

- **WHEN** the administrator changes the top-up rate
- **THEN** every Dong amount in the interface reflects the new rate on the next status refresh, with no code change

### Requirement: Dong is presented in whole units

Amounts in Dong SHALL be rendered with the `₫` symbol, grouped digits, and no fractional part. Rounding to a whole unit SHALL be applied only at presentation; the underlying value SHALL keep its precision.

#### Scenario: Fractional result

- **WHEN** a conversion yields a fractional Dong amount
- **THEN** the interface shows the nearest whole Dong, and any subsequent calculation still uses the unrounded value

#### Scenario: Sub-unit price

- **WHEN** a per-token price converts to less than one Dong
- **THEN** the interface presents it in a way that does not collapse distinct prices to the same displayed value, such as per-thousand or per-million units, consistently with how dollar sub-unit prices are already presented

### Requirement: Administrator display mode is currency or tokens

The administrator SHALL be able to choose between presenting amounts as currency or as raw token units. When currency is chosen, the currency itself SHALL be determined by the interface language as above; the administrator SHALL NOT be offered a choice of which currency. A stored display-mode value from before this change that names a specific currency SHALL be treated as the currency mode.

#### Scenario: Tokens mode

- **WHEN** the administrator selects the tokens display mode
- **THEN** amounts render as token units in every language

#### Scenario: Legacy stored value

- **WHEN** the stored display mode is a value that previously selected a specific currency
- **THEN** the interface behaves exactly as in currency mode, following the interface language

#### Scenario: Selector shows two modes

- **WHEN** the administrator opens the pricing display settings
- **THEN** exactly two modes are offered — currency following the interface language, and tokens
