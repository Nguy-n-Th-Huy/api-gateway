# public-model-status Specification

## Purpose

Shows, on the public key page, how each model has actually been behaving over the recent window — an operational badge plus a bar strip of recent success rates — so a key holder can tell a broken model from a broken key before opening a support ticket.

## Requirements

### Requirement: Model status section is shown on the public key page

The key page SHALL include a model status section that lists models with their recent operational health, reachable without signing in and rendered independently of whether a key has been checked.

#### Scenario: Anonymous visitor opens the page

- **WHEN** a visitor with no session opens `/key`
- **THEN** the model status section renders with the available model health data and no sign-in redirect occurs

#### Scenario: Model status loads before any key is checked

- **WHEN** the page has loaded and no key has been submitted
- **THEN** the model status section already shows model health

### Requirement: Model status is derived from recorded request performance

The model status section SHALL source its data from the existing performance metrics summary over the trailing 24-hour window. It SHALL NOT introduce a second, separately-computed notion of model health, and SHALL NOT require a schema change.

#### Scenario: Metrics already recorded

- **WHEN** the performance metrics summary reports a model with a success rate and a series of recent success rates
- **THEN** the model status section presents that model using those values

### Requirement: Each model shows an operational state badge

For each listed model the section SHALL show one of exactly three states, determined from that model's recent success rate over the window:

- `Operational` when the model has recorded traffic and its success rate is at or above 90 percent,
- `Down` when the model has recorded traffic and its success rate is below 90 percent,
- `No Data` when the model has no recorded traffic in the window.

#### Scenario: Healthy model

- **WHEN** a model recorded traffic with a 98 percent success rate
- **THEN** its badge reads `Operational`

#### Scenario: Failing model

- **WHEN** a model recorded traffic with a 20 percent success rate
- **THEN** its badge reads `Down`

#### Scenario: Silent model

- **WHEN** a model recorded no requests in the window
- **THEN** its badge reads `No Data` and its bar strip is rendered entirely as the no-data colour

#### Scenario: Boundary at the threshold

- **WHEN** a model recorded traffic with exactly a 90 percent success rate
- **THEN** its badge reads `Operational`

### Requirement: Each model shows a bar strip of recent intervals

For each listed model the section SHALL render a horizontal strip of bars, one bar per recorded interval in the window, oldest on the left and newest on the right. Each bar SHALL be coloured by that interval's success rate: healthy at or above 90 percent, degraded from 50 up to 90 percent, failing below 50 percent, and a distinct neutral colour for an interval with no recorded traffic. Colour SHALL NOT be the only carrier of meaning: each bar SHALL expose its interval and success rate as accessible text on hover and to assistive technology.

#### Scenario: Mixed strip

- **WHEN** a model's recent intervals include full-success, partial-failure, total-failure and empty intervals
- **THEN** the strip shows the corresponding healthy, degraded, failing and neutral bars in chronological order

#### Scenario: Bar detail

- **WHEN** a visitor hovers or focuses a single bar
- **THEN** that interval's time range and success rate are shown as text

#### Scenario: Model with no intervals recorded

- **WHEN** a model has no recorded intervals
- **THEN** the strip renders as a full row of neutral bars rather than an empty or collapsed element

### Requirement: Model list narrows to the checked key's group

Before any key is checked the section SHALL list every model the performance metrics summary reports. After a key check succeeds the section SHALL list only the models in that key's `available_models`, and SHALL state that the list is filtered to the key's group. A model in `available_models` that has no metrics entry SHALL still be listed, with the `No Data` state.

#### Scenario: Filtering after a key check

- **WHEN** a key whose group enables two models is checked and the metrics summary covers ten models
- **THEN** the section lists only those two models and indicates the list is scoped to the key's group

#### Scenario: Group model without metrics

- **WHEN** a key's group enables a model that has no metrics entry
- **THEN** that model is listed with the `No Data` state

#### Scenario: Checking a second key

- **WHEN** a visitor checks one key and then checks a different key whose group enables a different model set
- **THEN** the section lists the second key's models, with no model from the first key left over

### Requirement: Model status handles its own loading, empty and error states

The section SHALL show a loading indicator while its data is in flight, a localized empty message when no model has data, and a localized error message with a retry control when the request fails. A failure in the model status section SHALL NOT prevent the key check form or its result from working.

#### Scenario: Metrics request fails

- **WHEN** the performance metrics request returns an error
- **THEN** the section shows a localized error message with a retry control, and the key check form remains fully usable

#### Scenario: No models have data

- **WHEN** the metrics summary reports no models and no key has been checked
- **THEN** the section shows a localized empty message rather than an empty container

### Requirement: Model status text is localized

Every user-facing string in the model status section, including the three state badges, SHALL be resolved through the translation layer and SHALL have an entry in both the English and Vietnamese locale files.

#### Scenario: Vietnamese visitor

- **WHEN** the interface language is Vietnamese
- **THEN** the section heading, state badges, hover details, and empty/error messages are shown in Vietnamese
