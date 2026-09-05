## Purpose

Defines how the public header arranges its fixed navigation modules into groups, how those groups behave when the backend disables some of them, and how a visitor operates the grouped navigation with a pointer, a touchscreen, a keyboard, or a small screen.

## ADDED Requirements

### Requirement: Navigation modules are arranged into four top-level entries

The public header SHALL present its navigation as four top-level entries in this order: a home entry, a models group, a resources group, and a console entry. The models group SHALL contain the model catalogue and the rankings entries. The resources group SHALL contain the documentation and about entries. The system SHALL NOT introduce, remove, or re-route any navigation destination as part of this arrangement; every destination reachable before the change SHALL remain reachable after it.

#### Scenario: All navigation modules enabled

- **WHEN** every navigation module is enabled
- **THEN** the header shows exactly four top-level entries, and the model catalogue, rankings, documentation and about destinations are each reachable through them

#### Scenario: Destinations are preserved

- **WHEN** a visitor opens a group and activates one of its children
- **THEN** they arrive at the same destination that child pointed to before the grouping was introduced

### Requirement: A group adapts to how many of its children are enabled

The backend independently enables or disables each navigation module. The system SHALL render a group only when at least one of its children is enabled. A group left with exactly one enabled child SHALL render as a plain top-level link to that child, without a disclosure affordance and without a panel. A group with two or more enabled children SHALL render as a group entry that opens a panel.

#### Scenario: Every child of a group is disabled

- **WHEN** all children of a group are disabled
- **THEN** that group entry is not rendered at all, and no empty panel is reachable

#### Scenario: A group retains exactly one enabled child

- **WHEN** only one child of a group is enabled
- **THEN** that group renders as a plain link leading directly to the remaining child, with no panel and no disclosure indicator

#### Scenario: A group retains several enabled children

- **WHEN** two or more children of a group are enabled
- **THEN** the group renders as a group entry whose panel lists exactly the enabled children

### Requirement: Group panels open by pointer, touch and keyboard

The system SHALL allow a group panel to be opened with a pointer, with a touch, and from the keyboard, and SHALL NOT make the panel reachable only by hovering. An open panel SHALL close when the visitor presses Escape, and focus SHALL return to the group entry that opened it. The group entry SHALL expose its expanded state and the panel it controls to assistive technology, and the visitor SHALL be able to move between the panel's items with the arrow keys.

#### Scenario: Opening with a touchscreen

- **WHEN** a visitor taps a group entry on a touchscreen
- **THEN** the panel opens and its children can be activated by tapping

#### Scenario: Opening from the keyboard

- **WHEN** a visitor moves keyboard focus to a group entry and activates it
- **THEN** the panel opens and keyboard focus can reach every item inside it

#### Scenario: Dismissing with Escape

- **WHEN** a panel is open and the visitor presses Escape
- **THEN** the panel closes and focus returns to the group entry that opened it

#### Scenario: State exposed to assistive technology

- **WHEN** a group entry is rendered
- **THEN** it reports whether it is expanded and which panel it controls

### Requirement: Children keep their existing access and link behaviour

Grouping SHALL NOT change how an individual child behaves. A child whose module requires authentication for a signed-out visitor SHALL still raise the existing sign-in prompt rather than navigating, and opening that prompt SHALL close the panel. A documentation entry configured with an external address SHALL open in a new tab from inside the panel, with the same safety attributes it carries elsewhere. The entry matching the current page SHALL be marked as the current page for assistive technology.

#### Scenario: Child requiring authentication, visitor signed out

- **WHEN** a signed-out visitor activates a child whose module requires authentication
- **THEN** the sign-in prompt appears instead of navigation, and the panel closes

#### Scenario: Externally configured documentation link

- **WHEN** the documentation module points to an external address
- **THEN** activating it from inside the panel opens that address in a new tab

#### Scenario: Current page indicated

- **WHEN** the visitor is on a page reachable from the navigation
- **THEN** the entry for that page is marked as the current page

### Requirement: Small screens use a disclosure list, not a panel

On small screens the system SHALL present groups inside the existing mobile sheet as an expandable list, and SHALL NOT present the desktop panel. Every interactive row in the mobile navigation SHALL offer a touch target of at least 44 pixels in height.

#### Scenario: Group on a small screen

- **WHEN** a visitor opens the mobile navigation and activates a group
- **THEN** the group expands in place to reveal its children, and no floating panel appears

#### Scenario: Touch target size

- **WHEN** the mobile navigation is displayed
- **THEN** every row a visitor can activate is at least 44 pixels tall

### Requirement: Panel content is translated and free of unsourced figures

Every string introduced by the grouped navigation — group labels, child descriptions and highlight-cell copy — SHALL be available in both English and Vietnamese and SHALL be rendered through the translation layer rather than written into the markup. The panel SHALL NOT display a count of available models or any comparable figure, because the frontend has no source for it on this surface. The layout SHALL remain intact in both languages, given that Vietnamese labels are materially longer than their English equivalents.

#### Scenario: Vietnamese interface

- **WHEN** the interface language is Vietnamese
- **THEN** every group label, child description and highlight-cell string is shown in Vietnamese, and the panel layout is not broken by the longer text

#### Scenario: No fabricated counts

- **WHEN** the models panel is displayed
- **THEN** it presents its highlight cell without stating a number of available models
