# public-setup-script Specification

## Purpose

Turns a checked API key into a ready-to-paste one-line command that configures a supported AI coding CLI against this gateway, by writing that CLI's own configuration file — so a key holder gets a working client without hand-editing config or reading provider docs.

## Requirements

### Requirement: Setup section offers the supported applications and operating systems

The key page SHALL include a setup section listing exactly these applications: Claude Code, Codex, OpenCode, Pi, and Oh My Pi. For the selected application the section SHALL offer two operating system targets, Windows and macOS/Linux, and SHALL show the install prerequisite and the generated command for the selected combination.

#### Scenario: Switching application

- **WHEN** the visitor selects Codex in the application list
- **THEN** the section shows Codex's install prerequisite, Codex's model slots, and a Codex command

#### Scenario: Switching operating system

- **WHEN** the visitor switches from macOS/Linux to Windows
- **THEN** the install prerequisite and the generated command both change to the PowerShell form

#### Scenario: Default selection

- **WHEN** the setup section first renders
- **THEN** an application and an operating system are already selected, so a command is shown without any interaction

### Requirement: Setup section requires a checked key

The setup section SHALL be available only after a key check has succeeded, because the generated command embeds that key and the model list depends on that key's group. Before a successful check the section SHALL explain that a key must be checked first.

#### Scenario: No key checked yet

- **WHEN** the page has loaded and no key check has succeeded
- **THEN** the setup section shows a localized prompt to check a key first and offers no command

#### Scenario: Key check fails

- **WHEN** a key check fails
- **THEN** the setup section does not offer a command

### Requirement: Model choices come only from the key's group

Every model selector in the setup section SHALL be populated exclusively from the checked key's `available_models`. The generated command SHALL NOT reference any model outside that list.

#### Scenario: Selector contents

- **WHEN** the checked key's group enables three models
- **THEN** every model selector offers exactly those three models

#### Scenario: Group with no enabled models

- **WHEN** the checked key's group enables no model
- **THEN** the setup section states that the key's group has no available model and offers no command

#### Scenario: Re-checking a different key

- **WHEN** a second key from a different group is checked
- **THEN** the model selectors are repopulated from the second key's group and any previously selected model that the new group does not enable is replaced by a model the new group does enable

### Requirement: Setup script endpoint validates every model against the key's group

The system SHALL expose an endpoint that returns the setup script as plain text for a given application, operating system, and model selection, authenticated only by the supplied key. The endpoint SHALL resolve the key's effective group and SHALL reject the request when any requested model is not enabled for that group. The endpoint SHALL reject an unknown key, an unknown application, and an unknown operating system. The endpoint SHALL be rate limited with the project's critical-operation rate limit.

#### Scenario: Model outside the key's group

- **WHEN** a script is requested with a model the key's group does not enable
- **THEN** the request is rejected with a localized error and no script is returned

#### Scenario: Unknown application

- **WHEN** a script is requested for an application that is not one of the five supported ones
- **THEN** the request is rejected with a localized error

#### Scenario: Unknown key

- **WHEN** a script is requested with a key that matches no token
- **THEN** the request is rejected with the same generic invalid-key message used by the key check endpoint

#### Scenario: Model omitted for a slot

- **WHEN** a script is requested without a model for a slot the application defines
- **THEN** the request is rejected with a localized error naming the missing slot

### Requirement: Generated scripts write configuration files, never environment variables

Each generated script SHALL configure its CLI by writing that CLI's own configuration file. A script SHALL NOT configure the CLI by exporting shell environment variables, except where the CLI provides no documented way to read a credential from a file; in that case the script SHALL persist only that one credential variable and SHALL state in a comment why it is required.

The scripts SHALL write these files, using the gateway's configured server address as the base URL and the supplied key as the credential:

- **Claude Code**: `~/.claude/settings.json` (on Windows, `%USERPROFILE%\.claude\settings.json`), setting `env.ANTHROPIC_BASE_URL`, `env.ANTHROPIC_AUTH_TOKEN`, `env.ANTHROPIC_DEFAULT_OPUS_MODEL`, `env.ANTHROPIC_DEFAULT_SONNET_MODEL`, `env.ANTHROPIC_DEFAULT_HAIKU_MODEL`, and `env.CLAUDE_CODE_SUBAGENT_MODEL`.
- **Codex**: `~/.codex/config.toml` (on Windows, `%USERPROFILE%\.codex\config.toml`), defining a provider table with `base_url` ending in `/v1`, `wire_api = "responses"`, and `env_key`, plus the default `model` and `model_provider`, and one profile per model slot. Because Codex reads the credential only from the environment variable named by `env_key`, the script SHALL additionally persist that single variable and SHALL say so in a comment.
- **OpenCode**: `~/.config/opencode/opencode.json` (on Windows, `%USERPROFILE%\.config\opencode\opencode.json`), defining a provider whose `npm` is `@ai-sdk/openai-compatible` with `options.baseURL` and `options.apiKey`, the enabled models, and the top-level `model` and `small_model` in `provider/model` form.
- **Pi**: `~/.pi/agent/models.json` (on Windows, `%USERPROFILE%\.pi\agent\models.json`), defining a provider with `baseUrl`, `api`, `apiKey`, and the key group's models.
- **Oh My Pi**: `~/.omp/agent/models.yml` (on Windows, `%USERPROFILE%\.omp\agent\models.yml`), defining a provider with `baseUrl`, `api`, `apiKey`, and the key group's models. The script SHALL NOT write any key outside the documented `providers` root, because unknown root keys fail that file's schema validation.

#### Scenario: Claude Code settings written

- **WHEN** the Claude Code script runs on a machine with no prior configuration
- **THEN** `~/.claude/settings.json` exists and contains the gateway base URL, the key, and the four selected models under the documented keys, and the file is valid JSON

#### Scenario: No environment variable used where a file suffices

- **WHEN** the Claude Code, OpenCode, Pi, or Oh My Pi script runs
- **THEN** it persists no environment variable, and the CLI is configured entirely by its configuration file

#### Scenario: Codex credential exception

- **WHEN** the Codex script runs
- **THEN** it writes `config.toml` with the provider and model settings and persists exactly one environment variable, the one named by `env_key`, with a comment explaining that Codex reads the credential only from the environment

### Requirement: Generated scripts preserve unrelated existing configuration

A script SHALL merge into an existing configuration file rather than replacing it: it SHALL update only the keys it owns and SHALL leave every other key intact. When the existing file cannot be parsed the script SHALL stop with a clear message and SHALL NOT overwrite it, and SHALL tell the user where the unparsable file is.

#### Scenario: Existing settings preserved

- **WHEN** the Claude Code script runs on a machine whose `settings.json` already defines unrelated keys
- **THEN** those keys are still present afterwards and only the gateway and model keys are changed

#### Scenario: Unparsable existing file

- **WHEN** the target configuration file exists but is not valid for its format
- **THEN** the script stops with a message naming the file and makes no change to it

#### Scenario: Missing parent directory

- **WHEN** the target configuration file's directory does not exist
- **THEN** the script creates it before writing

### Requirement: Setup section shows the install prerequisite for the selected application

For each application and operating system the section SHALL show the documented install command as a separate, copyable step, presented as a prerequisite rather than as part of the configuration command.

#### Scenario: Claude Code on Windows

- **WHEN** Claude Code and Windows are selected
- **THEN** the section shows Claude Code's documented PowerShell install command as a separate copyable step

#### Scenario: Codex prerequisite

- **WHEN** Codex is selected
- **THEN** the section shows Codex's documented install command and its runtime requirement

### Requirement: Displayed command masks the key while the copied command carries it

The generated command SHALL be displayed with the key masked, showing only enough leading and trailing characters to recognise it. Copying the command SHALL place the unmasked, working command on the clipboard.

#### Scenario: Command on screen

- **WHEN** a command is displayed
- **THEN** the key inside it appears masked

#### Scenario: Copying the command

- **WHEN** the visitor activates the copy control
- **THEN** the clipboard contains the command with the real key, and the visitor is told the command was copied

### Requirement: Setup section states which slots map to which CLI setting

For every model selector the section SHALL label which role that model fills for the selected application, so the visitor understands what they are choosing. An application that has no documented model-role configuration SHALL show no model selector and SHALL state that model choice is made inside that CLI.

#### Scenario: Claude Code slots

- **WHEN** Claude Code is selected
- **THEN** four labelled selectors are shown for its Opus, Sonnet, Haiku and subagent roles

#### Scenario: OpenCode slots

- **WHEN** OpenCode is selected
- **THEN** two labelled selectors are shown, for its main model and its small model

#### Scenario: Application without documented model roles

- **WHEN** Pi or Oh My Pi is selected
- **THEN** no model selector is shown, the generated configuration registers every model the key's group enables, and the section states that the model is chosen inside that CLI

### Requirement: Setup section text is localized

Every user-facing string in the setup section SHALL be resolved through the translation layer and SHALL have an entry in both the English and Vietnamese locale files. The generated script content itself is not user interface text and is not translated.

#### Scenario: Vietnamese visitor

- **WHEN** the interface language is Vietnamese
- **THEN** the application names' surrounding labels, slot labels, step headings, prerequisite wording, and copy feedback are shown in Vietnamese
