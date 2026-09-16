# api-key-groups Specification

## Purpose
Defines how an API key's target groups are chosen in the console — one group, an ordered list of several groups, or the administrator's global Auto order — how that choice is stored on the key, how it is authorised, and how it is presented on the keys table.

## Requirements

### Requirement: The key form offers one ordered group picker

The API key form SHALL present one group control for every key, create and edit alike, regardless of which groups the operator has made usable, and that control SHALL NOT be hidden behind a separate control that has to be opened to reach a multi-group choice. The control SHALL be an ordered group picker, except while the global Auto order control is on, when it SHALL show the global Auto order the key will use instead (see "The global Auto order remains an explicit choice"). The picker SHALL hold an ordered list of groups and SHALL let the requester add a group, remove a group, and change the position of a group in the list. The picker SHALL NOT require the `auto` pseudo-group to be selected first.

The picker SHALL offer exactly the groups the requester may select for a key: the groups returned by the requester's group endpoint, excluding the `auto` pseudo-group, which is represented by its own toggle rather than as a list entry.

#### Scenario: Picker is present for a key that targets one group

- **WHEN** a requester opens the API key form while `auto` is not among the groups offered to them
- **THEN** the ordered group picker is shown, and it accepts one or more groups

#### Scenario: Picker is present when editing an existing key

- **WHEN** a requester opens an existing key for editing
- **THEN** the picker is shown with that key's current groups in their stored order

#### Scenario: Group list contents

- **WHEN** the requester opens the picker's group choices
- **THEN** only groups they may select for a key are offered, and `auto` is not one of the list entries

### Requirement: A key's groups are stored as one group, an ordered list, or the global Auto order

Saving a key SHALL map the form's group selection onto the key's existing stored fields, with no additional stored field and no change to the meaning of the existing ones:

- exactly one group selected: the key's group is that group, it carries no group snapshot, and cross-group retry is off;
- two or more groups selected: the key's group is `auto`, it carries the selected groups as an ordered snapshot, and cross-group retry takes the value of the form's cross-group retry switch;
- the global Auto order selected: the key's group is `auto`, it carries no group snapshot, and cross-group retry takes the value of the form's cross-group retry switch;
- no group selected: the key's group is empty, it carries no group snapshot, and cross-group retry is off, so the key follows its owner's group.

Reading a key back into the form SHALL invert this mapping: a key whose group is `auto` with a non-empty snapshot yields the explicit ordered list, a key whose group is `auto` with no snapshot yields the global Auto order selection, any other non-empty group yields a one-entry list, and an empty group yields an empty list. A snapshot that names a group the requester may no longer select SHALL be dropped when the key is loaded into the form, and the remaining list SHALL be truncated to the per-key group limit.

#### Scenario: One group selected

- **WHEN** a key is saved with exactly one group selected
- **THEN** the key stores that group, no snapshot, and cross-group retry off

#### Scenario: Several groups selected

- **WHEN** a key is saved with several groups selected, in a given order, and cross-group retry on
- **THEN** the key stores the group `auto`, those groups as its ordered snapshot, and cross-group retry on

#### Scenario: Global Auto order selected

- **WHEN** a key is saved with the global Auto order selected
- **THEN** the key stores the group `auto`, no snapshot, and cross-group retry from the form's switch

#### Scenario: No group selected

- **WHEN** a key is saved with no group selected
- **THEN** the key stores an empty group, no snapshot, and cross-group retry off

#### Scenario: Round-trip of a multi-group key

- **WHEN** a key stored with the group `auto` and an ordered snapshot, at least one of whose groups the requester may still select, is opened in the form and saved without any edit
- **THEN** the stored group, snapshot and cross-group retry value are unchanged

#### Scenario: A snapshot group is no longer selectable

- **WHEN** a key whose snapshot names a group the requester may no longer select is opened in the form
- **THEN** that group is absent from the picker and from what the form would save

#### Scenario: No snapshot group is still selectable

- **WHEN** a key whose snapshot names only groups the requester may no longer select is opened in the form and saved without a new selection
- **THEN** the picker is empty and the key stores an empty group, no snapshot and cross-group retry off

### Requirement: The global Auto order remains an explicit choice

The form SHALL offer a control that makes the key follow the administrator's global Auto order, and SHALL offer it only when the requester's group endpoint offers `auto`. Turning the control on SHALL clear the explicit group list, and the picker SHALL NOT be editable while the control is on; turning the control off SHALL restore the picker with its emptied list. When the requester's group endpoint does not offer `auto`, the control SHALL NOT be shown and the key SHALL NOT be storable in the global Auto order mode.

While the control is on, the form SHALL show the groups of the global Auto order in their order, with each group's ratio, so the requester can see which groups the key will try.

#### Scenario: Toggle offered

- **WHEN** the requester's group endpoint offers `auto`
- **THEN** the form shows the control that makes the key follow the global Auto order

#### Scenario: Global order is shown while the toggle is on

- **WHEN** the global Auto order control is on
- **THEN** the form lists the groups of the global Auto order, in order, with their ratios

#### Scenario: Toggle withheld

- **WHEN** the requester's group endpoint does not offer `auto`
- **THEN** the form shows no such control and a key cannot be saved in the global Auto order mode

#### Scenario: Turning the toggle on

- **WHEN** the requester turns the global Auto order control on while the picker holds groups
- **THEN** the picker is emptied and the key saves with no snapshot

#### Scenario: The picker while the toggle is on

- **WHEN** the global Auto order control is on
- **THEN** the explicit group picker is replaced by the global Auto order preview and cannot be edited, and turning the control off restores the picker with the emptied list

### Requirement: The cross-group retry switch follows the selection

The form SHALL show the cross-group retry switch when the selection can span more than one group — that is, when the picker holds two or more groups or the global Auto order control is on — and SHALL NOT show it otherwise. The switch SHALL default to on when the selection becomes able to span more than one group, unless the requester turns it off, and a key stored with cross-group retry on SHALL show the switch on when it is opened. A key that targets a single group SHALL be stored with cross-group retry off.

#### Scenario: Two or more groups selected

- **WHEN** the picker holds two or more groups
- **THEN** the cross-group retry switch is shown and its value is saved with the key

#### Scenario: Global Auto order selected

- **WHEN** the global Auto order control is on
- **THEN** the cross-group retry switch is shown and its value is saved with the key

#### Scenario: One group selected

- **WHEN** the picker holds exactly one group
- **THEN** the cross-group retry switch is not shown and the key is saved with cross-group retry off

### Requirement: Group selection is validated and limited

The system SHALL accept a key's group selection only when every named group is one the owner may select and no name repeats, and SHALL refuse more groups than the per-key group limit reported by the group-order endpoint. On refusal the key SHALL NOT be stored with the offending selection and the response SHALL carry the existing invalid-group, duplicate-group or too-many-groups error.

#### Scenario: A group the owner may not select

- **WHEN** a key is submitted naming a group the owner may not select
- **THEN** the request is refused with the invalid-group error and nothing is stored

#### Scenario: A repeated group

- **WHEN** a key is submitted naming the same group twice
- **THEN** the request is refused with the duplicate-group error and nothing is stored

#### Scenario: More groups than the limit

- **WHEN** a key is submitted with more groups than the per-key group limit
- **THEN** the request is refused with the too-many-groups error and nothing is stored

#### Scenario: The limit is the one the endpoint reports

- **WHEN** the group-order endpoint reports a per-key group limit
- **THEN** both the form and the write path enforce that same limit

### Requirement: Authentication accepts a multi-group key without the auto pseudo-group

Token authentication SHALL accept a key whose group is `auto` when the key carries a valid non-empty group snapshot, whether or not `auto` is among the owner's usable groups. This is what makes a multi-group key usable for an owner who may not select the `auto` pseudo-group.

Token authentication SHALL continue to refuse a key whose group is `auto` that carries no snapshot when `auto` is not among the owner's usable groups, with the same forbidden-group response as today. A snapshot that cannot be parsed SHALL be treated as no snapshot, so it grants no exemption. A key whose group is any other value SHALL keep today's behaviour, unchanged.

#### Scenario: Multi-group key of an owner without the auto pseudo-group

- **WHEN** a request is made with a key whose group is `auto` and whose snapshot names at least one group, and the owner's usable groups do not include `auto`
- **THEN** the request is authenticated and proceeds with the key's groups

#### Scenario: Global Auto key of an owner without the auto pseudo-group

- **WHEN** a request is made with a key whose group is `auto` and which carries no snapshot, and the owner's usable groups do not include `auto`
- **THEN** the request is refused with the forbidden-group response, exactly as today

#### Scenario: Unparsable snapshot

- **WHEN** a request is made with a key whose group is `auto`, whose snapshot cannot be parsed, and whose owner's usable groups do not include `auto`
- **THEN** the request is refused with the forbidden-group response

#### Scenario: Single-group and ungrouped keys

- **WHEN** a request is made with a key naming a single group, or with an ungrouped key
- **THEN** the request is authorised exactly as it was before the multi-group selection existed

### Requirement: A request is routed through the key's groups in order

For a key whose group is `auto`, the system SHALL consider the key's groups in their stored order and SHALL serve the request through the first group that has a channel available for the requested model. With cross-group retry enabled, a group that has exhausted its retry allowance SHALL hand the request to the next group in order; without it, the request SHALL NOT move to a later group.

Every group considered SHALL be re-authorised against the owner's currently usable groups before it is used, so a group the owner may no longer select is skipped and a snapshot whose groups are all no longer usable leaves the request with no group to run on. The global Auto order SHALL be used only for a key that carries no snapshot; a key that carries a snapshot SHALL never fall back to the global Auto order.

#### Scenario: First group with an available channel serves the request

- **WHEN** a key whose ordered groups are `vip` then `default` is used for a model with no channel in `vip` but a channel in `default`
- **THEN** the request is served through `default`

#### Scenario: Cross-group retry advances the order

- **WHEN** cross-group retry is enabled and the channels of the first group are exhausted for the requested model
- **THEN** the request is served through the next group in the order that has an available channel

#### Scenario: Cross-group retry disabled

- **WHEN** cross-group retry is disabled and no channel in the first group can serve the requested model
- **THEN** the request is not moved to a later group

#### Scenario: A snapshot group became unauthorised

- **WHEN** a key's snapshot names a group the owner may no longer select
- **THEN** that group is skipped when the request is routed

#### Scenario: No usable group remains

- **WHEN** a key whose group is `auto` carries a snapshot and none of its groups can serve the request
- **THEN** the request is refused, and no fallback to the global Auto order or to the owner's own group occurs

### Requirement: The serving group is the group that prices and records the request

For a request served through a key whose group is `auto`, the group that served it SHALL be the group used to price the request and the group recorded on the request's log entry and on any task the request creates. The key's stored selection SHALL NOT be the billed group.

#### Scenario: Billing follows the serving group

- **WHEN** a multi-group key's request is served through the second group in its order
- **THEN** the request is billed at that second group's ratio

#### Scenario: Log and task group

- **WHEN** a request served through the second group in the order produces a log entry or a task row
- **THEN** that row records the second group, not the key's `auto` placeholder

### Requirement: The keys table shows a key's real groups

The keys table's group column SHALL show the key's actual target groups. A key naming a single group SHALL render as it does today. A key whose group is `auto` and which carries a snapshot SHALL render its groups as an ordered sequence of group chips, each with that group's ratio, together with the cross-group indicator, and SHALL expose the stored order to the reader. A key whose group is `auto` with no snapshot SHALL render as it does today.

#### Scenario: Single-group key

- **WHEN** the table renders a key naming one group
- **THEN** that group's chip is shown with its ratio, unchanged from today

#### Scenario: Multi-group key

- **WHEN** the table renders a key whose group is `auto` with a snapshot
- **THEN** the snapshot's groups are shown in their stored order, each with its ratio, alongside the cross-group indicator

#### Scenario: Global Auto key

- **WHEN** the table renders a key whose group is `auto` with no snapshot
- **THEN** the cross-group badge and Auto ratio rendering are shown as they are today

#### Scenario: Stored order is readable

- **WHEN** the reader inspects a multi-group key's group cell
- **THEN** the stored order of the groups is available without opening the key's edit form

### Requirement: Existing keys keep their behaviour

Reloading this feature SHALL NOT change how an existing key behaves. A key created before this change that names one group SHALL keep that group, and a key that names `auto` with no snapshot SHALL keep following the global Auto order. Neither SHALL acquire a snapshot, and neither SHALL be rewritten by being opened in the form without an edit.

#### Scenario: Pre-existing single-group key

- **WHEN** a key that was created naming one group is used, listed, or opened in the form
- **THEN** it names that same group and behaves as it did before

#### Scenario: Pre-existing global Auto key

- **WHEN** a key that was created naming `auto` with no snapshot is used, listed, or opened in the form
- **THEN** it continues to follow the global Auto order

### Requirement: New interface text is translated

Every string this capability introduces or changes in the interface SHALL exist in both languages the product ships, in the product's translation resources, and SHALL follow the project's translation glossary.

#### Scenario: Both languages

- **WHEN** the key form and the keys table are used in either shipped language
- **THEN** every label, placeholder, description and message introduced by this capability is presented in that language
