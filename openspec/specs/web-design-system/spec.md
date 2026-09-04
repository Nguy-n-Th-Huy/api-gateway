# web-design-system Specification

## Purpose

Defines the shared visual vocabulary every screen in the frontend must speak — the anatomy and states of the recurring interface elements — so that pages restyled at different times stay consistent with one another.

## Requirements

### Requirement: Canonical component vocabulary

Recurring interface elements — actions, status indicators, text inputs, surfaces, navigation items and data-table rows — SHALL be provided as shared components with a fixed set of variants. Screens SHALL compose those shared components rather than re-implementing an element's appearance locally.

#### Scenario: Same element looks the same across screens

- **WHEN** the same element variant appears on two different screens
- **THEN** it renders with identical geometry, typography, color treatment and interaction states

#### Scenario: Variant change propagates

- **WHEN** a variant's definition is changed in the shared component
- **THEN** every screen using that variant reflects the change without per-screen edits

### Requirement: Complete interaction states

Every interactive element SHALL define a visible treatment for its default, hover, active, focus-visible, disabled, and — where the element can express one — loading and selected states. A state SHALL never be conveyed by color alone.

#### Scenario: Keyboard focus is visible

- **WHEN** a viewer moves keyboard focus to any interactive element
- **THEN** a focus indicator is visible against that element's surface and meets the 3:1 contrast floor

#### Scenario: Disabled control is inert and legible

- **WHEN** a control is disabled
- **THEN** it is visibly distinguished from its enabled state, is not activatable by pointer or keyboard, and exposes its disabled state to assistive technology

#### Scenario: Loading action prevents double submission

- **WHEN** an action is in its loading state
- **THEN** it shows progress, and repeated activation does not trigger the action again

### Requirement: Responsive behavior preserved

The visual language SHALL be applied within the application's existing responsive breakpoints. Every screen SHALL remain usable from small viewports upward, and no screen SHALL require horizontal scrolling of the page body at any supported viewport width.

#### Scenario: Small viewport keeps existing adaptations

- **WHEN** a screen that currently switches to a card list, drawer or stacked layout at a small viewport is viewed at that width after restyling
- **THEN** the same adaptation still occurs, restyled to the new visual language

#### Scenario: No page-level horizontal scroll

- **WHEN** any screen is viewed at any supported viewport width
- **THEN** the page body does not scroll horizontally; wide content such as tables and code blocks scrolls within its own container

### Requirement: Touch target floor

Interactive elements SHALL present a touch target of at least 44 by 44 CSS pixels at viewport widths where touch is the primary input, including when the visible control is drawn smaller.

#### Scenario: Small icon control on a touch viewport

- **WHEN** an icon-only control is rendered at a touch viewport width
- **THEN** its activatable area is at least 44 by 44 CSS pixels

### Requirement: All interface copy remains translatable

Restyling SHALL NOT introduce user-facing text outside the translation system. Every string a viewer can read SHALL resolve through the translation layer using the established English source key.

#### Scenario: Restyled screen in a non-default locale

- **WHEN** a restyled screen is viewed in any supported locale
- **THEN** every user-facing string is translated, with no hardcoded literal appearing in place of a translated string

#### Scenario: Long translation does not break layout

- **WHEN** a translated string is substantially longer than its English source
- **THEN** the containing element wraps or truncates by an explicit rule, without overlapping adjacent elements or being clipped without indication
