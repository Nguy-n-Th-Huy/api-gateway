# interface-locales Specification

## Purpose
Defines which languages the interface and the backend messages are offered in, how a visitor's language is detected and remembered, and how a previously stored language that is no longer offered is handled.

## Requirements

### Requirement: Exactly two interface languages

The interface SHALL be offered in Vietnamese and English and in no other language. The language switcher and the profile language preference SHALL list exactly these two. Translation resources for any other language SHALL NOT be shipped in the interface bundle.

#### Scenario: Switcher contents

- **WHEN** a visitor opens the language switcher or the profile language preference
- **THEN** exactly Vietnamese and English are offered

#### Scenario: No other locale bundled

- **WHEN** the interface bundle is built
- **THEN** it contains translation resources for Vietnamese and English only

### Requirement: Language detection resolves to an offered language

A visitor's language SHALL be detected from their stored preference first and their browser language second. A browser language of Vietnamese, in any regional variant, SHALL resolve to Vietnamese. Any other browser language, including Chinese, SHALL resolve to English. English SHALL remain the fallback for any string missing a Vietnamese translation.

#### Scenario: Vietnamese browser

- **WHEN** a first-time visitor's browser reports Vietnamese
- **THEN** the interface opens in Vietnamese

#### Scenario: Other browser language

- **WHEN** a first-time visitor's browser reports a language other than Vietnamese or English
- **THEN** the interface opens in English

#### Scenario: Missing translation

- **WHEN** a Vietnamese translation is missing for a string
- **THEN** the English source string is shown for that string, and the rest of the interface stays in Vietnamese

### Requirement: A stored language that is no longer offered falls back cleanly

A visitor whose stored preference names a language that is no longer offered SHALL be treated as having no stored preference: detection proceeds from the browser language, the interface renders normally, and the stored value is replaced by the resolved language.

#### Scenario: Stored Chinese preference

- **WHEN** a returning visitor's stored preference is a language that was removed
- **THEN** the interface opens in the language resolved from their browser, shows no error, and stores that resolved language

### Requirement: Backend messages are available in Vietnamese and English

Messages the backend returns to the interface — errors, refusals, status text — SHALL be available in Vietnamese and English, selected from the request's language header. A header requesting Vietnamese in any regional variant SHALL receive Vietnamese; a header requesting English SHALL receive English; any other or missing header SHALL receive English. Every message key available in English SHALL have a Vietnamese translation.

#### Scenario: Vietnamese request

- **WHEN** a request carries a Vietnamese language header and triggers a backend message
- **THEN** the message is returned in Vietnamese

#### Scenario: Unsupported header

- **WHEN** a request carries a language header for a language the backend does not offer
- **THEN** the message is returned in English

#### Scenario: Complete Vietnamese coverage

- **WHEN** the backend message bundles are compared
- **THEN** every key present in the English bundle is present in the Vietnamese bundle
