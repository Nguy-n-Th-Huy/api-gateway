## Purpose

Defines how an operator marks a group as usable by administrators only, and how every surface that lists, offers, prices, or accepts a group behaves for a requester who is not an administrator — including an anonymous visitor and an API key that already carries such a group.

## ADDED Requirements

### Requirement: A group can be marked admin-only

The system SHALL let an operator mark any group in the group registry as admin-only, and SHALL persist that marking as configuration alongside the other group settings. The marking SHALL be independent of whether the group is user-selectable, whether it carries a ratio, and whether it participates in automatic group selection: marking a group admin-only SHALL NOT alter any of those settings.

A group that is not marked admin-only SHALL behave exactly as it does today. An operator who has never used this feature SHALL observe no change in behaviour on any surface.

The system SHALL tolerate a marked name that is not present in the group registry, treating it as inert rather than as an error, so that deleting a group does not require the marking to be cleaned up first.

#### Scenario: Marking a group admin-only

- **WHEN** an operator marks an existing group as admin-only and saves
- **THEN** the marking is persisted, and the group's ratio, description, user-selectable state and automatic-selection membership are unchanged

#### Scenario: No group is marked

- **WHEN** no group is marked admin-only
- **THEN** every group resolves for every requester exactly as it did before this capability existed

#### Scenario: A marked group is removed from the registry

- **WHEN** a group that was marked admin-only is deleted from the group registry
- **THEN** the leftover marking has no effect and produces no error on any surface

### Requirement: Only administrators can resolve an admin-only group

The system SHALL resolve a requester's usable groups from the requester's role. A requester whose role is administrator or higher SHALL have admin-only groups resolved normally. A requester below that role SHALL have every admin-only group removed from their usable groups, and the removal SHALL take precedence over every other rule that would otherwise grant the group — including the per-user-group override that explicitly adds a group, and the rule that grants a requester their own user group.

A requester with no authenticated session SHALL be treated as below the administrator role.

#### Scenario: Administrator resolves usable groups

- **WHEN** an administrator's usable groups are resolved
- **THEN** admin-only groups are present, with the same description and ratio a non-marked group would carry

#### Scenario: Ordinary user resolves usable groups

- **WHEN** an ordinary user's usable groups are resolved
- **THEN** no admin-only group is present

#### Scenario: Anonymous requester

- **WHEN** usable groups are resolved for a request that carries no authenticated session
- **THEN** no admin-only group is present

#### Scenario: Per-user-group override tries to grant an admin-only group

- **WHEN** an ordinary user's user group carries an override that explicitly adds a group that is marked admin-only
- **THEN** that group is still absent from their usable groups

#### Scenario: Ordinary user is assigned an admin-only group as their own group

- **WHEN** an ordinary user's own user group is marked admin-only
- **THEN** that group is absent from their usable groups, and their requests through it are refused like any other group they may not use

### Requirement: Admin-only groups are absent from the group list offered to a requester

The system SHALL NOT include an admin-only group in the group list returned to a requester below the administrator role, on any endpoint that answers "which groups may this requester pick". This covers both the authenticated form of that endpoint and any form of it reachable without a session. The response SHALL carry no name, description, or ratio belonging to an admin-only group.

The administrator-facing group registry SHALL keep returning every group, admin-only groups included, so that an administrator can still assign them to channels and to users.

#### Scenario: Ordinary user opens the API key creation form

- **WHEN** an ordinary user requests the groups they may pick for a new API key
- **THEN** admin-only groups are not offered, and the form presents only the groups they may actually use

#### Scenario: Administrator opens the API key creation form

- **WHEN** an administrator requests the groups they may pick for a new API key
- **THEN** admin-only groups are offered alongside the others

#### Scenario: Unauthenticated request for the group list

- **WHEN** the group list is requested without a session
- **THEN** the response contains no admin-only group

#### Scenario: Administrator manages channels

- **WHEN** an administrator lists the groups available for channel and user assignment
- **THEN** every group in the registry is returned, including admin-only groups

### Requirement: Admin-only groups are absent from the model catalogue shown to a requester

The system SHALL exclude admin-only groups from the model and pricing catalogue served to a requester below the administrator role. Specifically, for such a requester the response SHALL NOT contain an admin-only group in its usable-group map, SHALL NOT contain a ratio for an admin-only group, SHALL NOT list an admin-only group among the automatically selected groups, and SHALL NOT include a model whose only enabling group is admin-only.

A model that is enabled for both an admin-only group and a group the requester may use SHALL remain visible to that requester.

The per-requester model list SHALL apply the same rule: a model reachable only through an admin-only group SHALL NOT appear for a requester below the administrator role, and requesting the model list filtered by an admin-only group SHALL yield no models for such a requester.

#### Scenario: Ordinary user views the model catalogue

- **WHEN** an ordinary user loads the model and pricing catalogue
- **THEN** no admin-only group appears in the usable groups or the ratio table, and no model exclusive to an admin-only group is listed

#### Scenario: Anonymous visitor views the model catalogue

- **WHEN** a visitor with no session loads the public model and pricing catalogue
- **THEN** the response contains no trace of any admin-only group — no name, no description, no ratio, no exclusive model

#### Scenario: Administrator views the model catalogue

- **WHEN** an administrator loads the model and pricing catalogue
- **THEN** admin-only groups and their exclusive models are present

#### Scenario: Model shared between an admin-only group and a normal group

- **WHEN** a model is enabled for both an admin-only group and a group an ordinary user may use
- **THEN** that user still sees the model, priced by the group they may use

#### Scenario: Ordinary user requests their model list for an admin-only group

- **WHEN** an ordinary user asks for the models of a group that is marked admin-only
- **THEN** no models are returned

### Requirement: Admin-only groups are excluded from automatic group selection for non-administrators

The system SHALL exclude admin-only groups from the automatic group selection offered to, stored for, and applied to a requester below the administrator role. Such a requester SHALL NOT be offered an admin-only group when choosing an automatic-selection order, SHALL NOT be able to save one into a key's automatic-selection order, and SHALL NOT have a request routed through one when their key resolves its group automatically — even if the operator has placed that group in the global automatic-selection order.

#### Scenario: Ordinary user edits the automatic selection order

- **WHEN** an ordinary user lists the groups available for a key's automatic selection order
- **THEN** no admin-only group is offered

#### Scenario: Ordinary user submits an admin-only group in the automatic selection order

- **WHEN** an ordinary user tries to save a key whose automatic selection order names an admin-only group
- **THEN** the request is rejected with the same invalid-group error any other unusable group would produce, and the order is not stored

#### Scenario: Global automatic order contains an admin-only group

- **WHEN** an operator has placed an admin-only group in the global automatic-selection order and an ordinary user's key resolves its group automatically
- **THEN** the admin-only group is skipped and the request is served by a group the user may use, or refused if none remains

#### Scenario: Administrator's key resolves automatically

- **WHEN** an administrator's key resolves its group automatically and an admin-only group is in the order
- **THEN** the admin-only group participates normally

### Requirement: An existing API key carrying an admin-only group is refused for non-administrators

Group enforcement at request time SHALL use the key owner's role. A key whose group is admin-only SHALL be refused when its owner is below the administrator role, with the same forbidden-group response the system already returns for a group the owner may not use. This SHALL hold for a key created before the group was marked admin-only, and for a key whose group was set through any path that does not validate the group at creation time.

A key belonging to an administrator SHALL continue to work with an admin-only group. Demoting a user below the administrator role SHALL cause their keys on admin-only groups to be refused from that point on.

#### Scenario: Ordinary user's pre-existing key on a newly marked group

- **WHEN** a group carrying live keys is marked admin-only and an ordinary user's key on that group is used
- **THEN** the request is refused with the existing forbidden-group response

#### Scenario: Administrator's key on an admin-only group

- **WHEN** an administrator uses a key whose group is admin-only
- **THEN** the request is served normally

#### Scenario: Administrator is demoted

- **WHEN** a user holding a key on an admin-only group is demoted below the administrator role
- **THEN** subsequent requests with that key are refused

### Requirement: The playground rejects an admin-only group override for non-administrators

Where a request may name a group to use for a single call rather than taking it from the key, the system SHALL refuse an admin-only group named by a requester below the administrator role, with the same access-denied response it already returns for a group the requester may not use.

#### Scenario: Ordinary user overrides the group in the playground

- **WHEN** an ordinary user sends a playground request naming an admin-only group
- **THEN** the request is refused with the existing group access-denied response

#### Scenario: Administrator overrides the group in the playground

- **WHEN** an administrator sends a playground request naming an admin-only group
- **THEN** the request is served through that group

### Requirement: The group editor exposes the admin-only flag and its consequence

The group management surface in system settings SHALL let an operator set and clear the admin-only flag on each group in the same place where the group's ratio, description and user-selectable state are set, SHALL show the flag's current state for each group in the group listing and in the group's detail view, and SHALL state plainly, where the flag is set, that marking a group admin-only makes it unusable and invisible to every user below the administrator role — including any user whose own user group is that group.

All text introduced by this surface SHALL be available in both languages the product ships.

#### Scenario: Operator sets the flag while creating a group

- **WHEN** an operator adds a new group and marks it admin-only in the same form
- **THEN** the group is created with the flag set, and the listing shows it as admin-only

#### Scenario: Operator reviews an existing group

- **WHEN** an operator opens the detail view of a group that is marked admin-only
- **THEN** the detail view shows the group as admin-only

#### Scenario: Operator is told what the flag does

- **WHEN** an operator views the control that sets the flag
- **THEN** the consequence for non-administrator users, including users assigned to that group, is stated in the interface

#### Scenario: Interface language

- **WHEN** an operator uses the group editor in either shipped language
- **THEN** every label, state and explanation introduced by this capability is presented in that language
