# web-theming Specification

## Purpose

Defines the single visual theme the frontend presents: the design token contract every screen reads from, light/dark parity, and which appearance choices a viewer is offered.

## Requirements

### Requirement: Single application theme

The frontend SHALL present exactly one visual theme. No preset, font-family, corner-radius, type-scale or content-width variant SHALL be selectable by a viewer, and no such variant SHALL be reachable through a URL parameter, cookie or stored preference.

#### Scenario: No appearance customization surface

- **WHEN** an authenticated viewer opens any page in the application
- **THEN** no control for choosing a theme preset, font family, corner radius, type scale or content width is present anywhere in the interface

#### Scenario: Stored appearance preference is inert

- **WHEN** a browser still holds an appearance preference written by the previous theme-customization system
- **THEN** the application renders the single theme unchanged, does not apply the stored value, and does not error

#### Scenario: Appearance override attempt has no effect

- **WHEN** a request supplies a theme-preset or appearance-axis value through any client-controlled channel
- **THEN** the rendered appearance is identical to a request with no such value

### Requirement: Design token contract

All colors, radii, typography and elevation SHALL be exposed as CSS custom properties on the document root, and every screen SHALL derive its appearance from those properties rather than from literal values embedded in components. Color tokens SHALL be authored in `oklch`.

#### Scenario: Screens read tokens, not literals

- **WHEN** the value of a color, radius or typography token is changed at the root
- **THEN** every screen that uses that token reflects the new value with no other code change

#### Scenario: Token set is complete for both modes

- **WHEN** the application resolves the token set in either light or dark mode
- **THEN** every token referenced by any screen resolves to a defined value, and no screen falls back to a browser default color

### Requirement: Light and dark parity

The application SHALL support light and dark modes, both expressing the same design language. Switching modes SHALL change only token values — never layout, spacing, component anatomy or which elements are visible.

#### Scenario: Viewer switches mode

- **WHEN** a viewer toggles between light and dark mode on any screen
- **THEN** the screen keeps identical layout, spacing and content, and only colors and elevation change

#### Scenario: Mode preference persists

- **WHEN** a viewer selects a mode and later reloads the application
- **THEN** the previously selected mode is applied on first paint, without a flash of the other mode

#### Scenario: System preference is honored by default

- **WHEN** a viewer who has never chosen a mode opens the application
- **THEN** the mode follows the operating system preference

### Requirement: Contrast floor in both modes

Every token pairing used for text or interactive controls SHALL meet WCAG 2.1 AA contrast in both light and dark mode: at least 4.5:1 for body text and at least 3:1 for large text, icons conveying meaning, and control boundaries.

#### Scenario: Body text contrast

- **WHEN** body text is rendered on its intended surface token in either mode
- **THEN** the contrast ratio is at least 4.5:1

#### Scenario: Status colors remain distinguishable

- **WHEN** success, warning, error and neutral status treatments are rendered in either mode
- **THEN** each meets the contrast floor and remains distinguishable by more than hue alone

### Requirement: Chart palette derives from theme tokens

Data visualizations SHALL take their series colors from the theme token set, so charts change with the theme and match the surrounding interface in both modes.

#### Scenario: Chart follows the active mode

- **WHEN** a viewer switches between light and dark mode on a screen containing a chart
- **THEN** the chart's series colors, axes, gridlines and labels re-render from the active mode's tokens and remain legible against the chart surface

#### Scenario: Categorical series stay distinguishable

- **WHEN** a chart renders multiple categorical series
- **THEN** adjacent series are distinguishable from one another in both modes
