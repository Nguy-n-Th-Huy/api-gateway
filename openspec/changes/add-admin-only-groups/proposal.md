## Why

An operator needs a group that only administrators can use — an internal group carrying an expensive or experimental model set, a staff-only channel pool, a group whose pricing must not be advertised. Today the gateway has no way to express that. A group is either in `UserUsableGroups`, in which case every visitor — including anonymous ones, because `/api/user/groups` and `/api/pricing` are reachable without a session — sees its name, its description and its ratio, can pick it in the API-key form, and can relay traffic through it; or it is absent, in which case administrators cannot use it either.

The only workaround available today is `GroupSpecialUsableGroup`, which removes a group per *user group* with a `-:` entry. That is an enumeration: every non-admin user group must be listed, and a user group added later silently regains access. It fails closed for nobody and it fails open by default.

## What Changes

- Introduce an **admin-only** flag on a group. A group marked admin-only stays fully functional for administrators and root users, and behaves as if it did not exist for everyone else.
- The flag lives in a new `AdminOnlyGroups` option (a JSON array of group names) alongside the existing group options. `UserUsableGroups` keeps its current shape, so no migration and no compatibility break.
- Usable-group resolution becomes role-aware. **BREAKING (internal Go API):** `service.GetUserUsableGroups` and its derived helpers gain a role parameter, so every call site is forced by the compiler to state which role it is resolving for. No role-blind variant survives.
- Every surface that exposes or accepts a group honours the flag for non-administrators:
  - the group list behind the API-key form and the playground group picker,
  - the pricing/model catalogue — its model rows, its `group_ratio` map, its `usable_group` map and its `auto_groups` list,
  - the per-user model list,
  - the auto-group order editor and the auto-group resolution used when relaying,
  - token authentication, which is the point where an already-stored token group is actually enforced.
- An anonymous visitor is treated as the lowest role and therefore never sees an admin-only group.
- A non-administrator whose own user group is marked admin-only is denied consistently rather than being granted an implicit exemption. The settings UI states this consequence where the flag is set.
- The administrator-facing group registry (`/api/group`, admin-only already) keeps returning every group, admin-only included — an administrator must still be able to assign these groups to channels and users.
- The group editor in system settings gains an **Admin only** control on each group row and shows the flag in the group detail panel, with English and Vietnamese strings.

Not in scope, deliberately:

- No change to how groups are stored. Groups remain keys across the existing option maps; this change adds one more option beside them and does not introduce a groups table, a schema change, or a migration.
- No per-user or per-user-group allow-list for admin-only groups. The gate is the requester's role, nothing finer. `GroupSpecialUsableGroup` keeps working exactly as it does today for everything else.
- No new role tier. The existing `RoleAdminUser` threshold is the line.
- No hiding of admin-only groups from administrator-facing surfaces — usage logs, channel editing, and user editing continue to show them.

## Capabilities

### New Capabilities

- `admin-only-groups`: how a group is marked usable by administrators only, what role is required to see or use it, and how every surface that lists, offers, or accepts a group behaves for a requester below that role — including anonymous requesters and already-issued API keys.

### Modified Capabilities

<!-- None. No existing capability under openspec/specs/ states requirements about group visibility or usable-group resolution. -->

## Impact

**Backend (Go).**

- `setting/` — new admin-only group option, following the shape of `setting/auto_group.go` (JSON string in, list out, plus a membership check).
- `model/option.go` — register the option in `common.OptionMap` and handle it in the option-update switch, the same way `AutoGroups` is handled.
- `service/group.go` — the chokepoint. `GetUserUsableGroups`, `GroupInUserUsableGroups`, `IsUserSelectableGroup`, `GetUserAutoGroup`, `FilterUserTokenAutoGroups` and `GetRequestAutoGroups` become role-aware.
- `constant/context_key.go` and `model/user_cache.go` — the relay path has the caller's role in the user cache but never puts it on the request context; a context key is needed so `GetRequestAutoGroups` can read it.
- `controller/group.go`, `controller/pricing.go`, `controller/user.go`, `controller/token.go` — pass the requester's role through. `/api/user/groups` has no auth middleware at all, so it must default to the lowest role rather than assume a session.
- `middleware/auth.go` — token authentication already rejects a token whose group is not usable; it must make that decision with the token owner's role. This is the enforcement point that matters, because token creation does not validate the group.
- `middleware/distributor.go` — the playground group override.

**Frontend (`web/`).**

- `web/src/features/system-settings/models/group-ratio-visual-editor.tsx` — the group table row model, its serializer, the new column, and the group detail panel.
- `web/src/features/system-settings/models/` sibling form/section wiring that passes the option values into the editor.
- `web/src/i18n/locales/en.json` and `web/src/i18n/locales/vi.json` — new strings, keyed by their English source.
- No change to the API-key form, the pricing page, or the playground picker: they render whatever the server returns, and the server will stop returning admin-only groups to non-administrators.

**Not affected.**

- No database schema, no migration, no `AutoMigrate` change — the new option is a row in the existing options store, written through the existing option path. The three-database verification matrix does not apply to this change.
- No relay adaptor, no billing arithmetic, no quota conversion.
