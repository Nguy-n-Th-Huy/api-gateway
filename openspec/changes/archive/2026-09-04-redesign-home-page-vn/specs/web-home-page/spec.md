## Purpose

Defines what the public home page presents to an unauthenticated visitor, which data source backs each block, and how every block behaves when its data is missing, still loading, or turned off by an administrator.

## ADDED Requirements

### Requirement: Home page block composition

The public home page SHALL present, in order: a hero, a provider strip, a statistics block, a bank-transfer top-up block, a pricing preview, a feature overview, an integrations block, a comparison block, a frequently-asked-questions block, a closing call to action, and the site footer. The page SHALL render this composition only when no administrator-supplied custom home page content is configured; when custom content is configured it SHALL continue to be rendered instead, unchanged.

#### Scenario: Default home page

- **WHEN** an unauthenticated visitor opens the site root and no custom home page content is configured
- **THEN** the blocks above are rendered in that order

#### Scenario: Custom home page content takes precedence

- **WHEN** an administrator has configured custom home page content as a URL, HTML, or Markdown
- **THEN** that content is rendered instead of the default composition, exactly as before this change

### Requirement: Hero states the offer and the primary actions

The hero SHALL state, in the language of the visitor, what the gateway does and who it is for, and SHALL offer a path to sign up, a path to the documentation, and a path to the pricing information on the same page. The documentation path SHALL use the administrator-configured documentation link when one is set.

#### Scenario: Signed-out visitor

- **WHEN** an unauthenticated visitor views the hero
- **THEN** a sign-up action is offered

#### Scenario: Signed-in visitor

- **WHEN** an authenticated visitor views the hero
- **THEN** the sign-up action is replaced by an action leading into the console

### Requirement: Top-up block describes the transfer flow without figures

The top-up block SHALL describe the bank-transfer payment flow as an ordered sequence of steps and SHALL offer an action leading to sign-up for unauthenticated visitors or to the wallet for authenticated ones. Because bank account details, transfer codes, and minimum amounts are only available to authenticated users, the block SHALL NOT display any amount, bank account number, or transfer code.

#### Scenario: No account data is exposed publicly

- **WHEN** any visitor views the top-up block
- **THEN** no bank account number, transfer code, or currency amount is shown

### Requirement: Pricing preview is sourced from live pricing data

The pricing preview SHALL display model prices obtained from the public pricing data, and SHALL NOT display any hardcoded or invented price. It SHALL show a loading treatment while the data is being fetched, an error treatment with a way to retry when the fetch fails, and an empty treatment when the pricing data contains no models. Every treatment SHALL keep the surrounding page usable.

#### Scenario: Pricing data available

- **WHEN** the pricing data returns one or more models
- **THEN** a bounded preview of those models with their prices is shown, alongside a path to the full pricing page

#### Scenario: Pricing data fails to load

- **WHEN** the pricing request fails
- **THEN** an error message and a retry action are shown in place of the table, and the rest of the page still renders

#### Scenario: Pricing data is empty

- **WHEN** the pricing data returns no models
- **THEN** an empty-state message is shown in place of the table

### Requirement: FAQ block reflects administrator configuration

The FAQ block SHALL render only the questions and answers configured by an administrator. When the FAQ feature is disabled, or when it is enabled but no entries are configured, the block SHALL NOT be rendered at all — no heading, no placeholder, and no empty container.

#### Scenario: FAQ configured

- **WHEN** the FAQ feature is enabled and at least one entry is configured
- **THEN** the block renders those entries, each expandable to reveal its answer

#### Scenario: FAQ disabled or empty

- **WHEN** the FAQ feature is disabled, or is enabled with no entries
- **THEN** no FAQ block appears on the page

### Requirement: No fabricated figures

Every number, claim, quotation, or logo presented on the home page SHALL be traceable to a live data source or to committed project content. Availability percentages, latency figures, customer counts, and customer testimonials SHALL NOT be shown unless such a source exists.

#### Scenario: Unsourced claim is absent

- **WHEN** a metric has no backing data source
- **THEN** it does not appear on the page in any form, including as illustrative or placeholder text

### Requirement: Home page accessibility and responsiveness

Every home page block SHALL be usable from a 360 CSS-pixel-wide viewport up through desktop widths without horizontal scrolling of the page body. Interactive elements SHALL expose an accessible name, reach a minimum 44 by 44 CSS pixel activation area on coarse pointers, and meet the WCAG 2.1 AA contrast floor. Decorative imagery SHALL be hidden from assistive technology.

#### Scenario: Narrow viewport

- **WHEN** the page is viewed at 360 CSS pixels wide
- **THEN** all blocks reflow to a single column and the page body does not scroll horizontally

#### Scenario: Keyboard navigation

- **WHEN** a visitor moves through the page with the keyboard alone
- **THEN** every action and every expandable FAQ entry is reachable, operable, and shows a visible focus indicator

### Requirement: Home page text is translatable

All user-facing home page text SHALL be resolved through the translation layer using English source strings as keys, with Vietnamese translations supplied. No user-facing string SHALL be hardcoded in a single language in the component tree.

#### Scenario: Language switch

- **WHEN** a visitor switches the interface language
- **THEN** every home page block renders in the selected language, falling back to the English source string when a translation is missing
