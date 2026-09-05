## Context

See `proposal.md` — Why, for motivation. Requirements live in `specs/admin-only-groups/spec.md`.

Constraints established by reading the code before planning:

- **A group is not a record.** It is a name that appears as a key across independent option maps: `GroupRatio` and `GroupGroupRatio` and `GroupSpecialUsableGroup` (`setting/ratio_setting/group_ratio.go`), `UserUsableGroups` (`setting/user_usable_group.go`), `TopupGroupRatio`, and `AutoGroups` (`setting/auto_group.go`). "Creating a group" today means adding a row in the group table of the system-settings editor, which writes those maps. There is no groups table and no place to hang a column on.
- **`service.GetUserUsableGroups(userGroup string)` is the single chokepoint.** Every surface that answers "may this requester use this group" goes through it, directly or through `GroupInUserUsableGroups`, `IsUserSelectableGroup`, `GetUserAutoGroup`, `FilterUserTokenAutoGroups` and `GetRequestAutoGroups`. Its callers are `controller/group.go`, `controller/pricing.go`, `controller/user.go`, `controller/token.go`, `middleware/auth.go` and `middleware/distributor.go`. It takes a user group and nothing else — it has no idea who is asking.
- **Two of those surfaces answer without a session.** `/api/user/groups` is registered with no auth middleware at all, and `/api/pricing` runs `HeaderNavModuleAuth("pricing")`, which falls through to `TryUserAuth` when the module is public. Both must therefore treat "no session" as the lowest role rather than assume one.
- **Token creation does not validate the group.** `controller/token.go:AddToken` copies `token.Group` straight onto the new key; only `token.Group == "auto"` gets validated, and only for its auto-group list. The place a group is actually enforced is `middleware/auth.go` `TokenAuth`, which checks the group against the owner's usable groups on every relayed request. Any gate that lives only in the creation form is decorative.
- **The role is present but not on the relay context.** `model.UserBase` carries `Role`; `setDashboardAuthContext` puts it on the gin context as `"role"`; `TokenAuth` holds it as `userCache.Role`. But `UserBase.WriteContext` — the one function both paths call — sets group, quota, status, email, name and setting, and not role. So code downstream of `TokenAuth`, including auto-group resolution during a relay, cannot see it.
- **`GroupSpecialUsableGroup` can add groups.** `GetUserUsableGroups` applies `+:`/`-:`/bare entries per user group after copying the base map, and then unconditionally grants the requester their own user group with the description `"用户分组"` if it is missing. Both of those run after the base map is built, so a filter placed too early would be undone by them.
- **Frontend group management is one component.** `web/src/features/system-settings/models/group-ratio-visual-editor.tsx` holds the row model (`GroupPricingRow`), the parse/serialize pair (`buildGroupPricingRows` / `serializeGroupPricingRows`), the table, and the detail sheet. It receives each option as a JSON string and reports changes with `onChange(field, value)`.

## Goals / Non-Goals

**Goals:**

- Make the gate impossible to forget: after this change it must not be possible to resolve usable groups without stating whose role you are resolving for.
- Put the filter where the enforcement already is, so an already-issued key is covered, not just the form that creates one.
- Add the setting without a schema change, a migration, or a compatibility break in an option's shape.
- Keep every behaviour identical when the feature is unused.

**Non-Goals:**

- No groups table, no per-group record, no normalisation of the existing option maps. That is a much larger change and this feature does not need it.
- No new role tier and no per-user allow-list.
- No client-side hiding as the mechanism. The client may render whatever it receives; the server decides what it receives.

## Decisions

### Store the flag as its own option, not as a change to `UserUsableGroups`

`AdminOnlyGroups` becomes a new option holding a JSON array of group names, with a getter, a JSON-string updater and a membership check in `setting/`, registered in `common.OptionMap` and handled in the option-update switch in `model/option.go`. This mirrors `AutoGroups` exactly, which is the closest existing concept: a list of group names, orthogonal to the maps, edited from the same settings screen.

Alternatives rejected:

- **Encode it in `UserUsableGroups`** — the value is a description string rendered in the UI; overloading it with a marker (a prefix, a sentinel) makes an operator's description text load-bearing and would silently reinterpret existing configurations.
- **Change `UserUsableGroups` to a map of objects** — a shape change to a stored option that ships in every existing deployment, so it needs a read-time migration and a compatibility path for the old shape, all to carry one boolean.
- **Reuse `GroupSpecialUsableGroup` with `-:` entries** — this is what an operator would have to do today, and the proposal explains why it fails: it enumerates the user groups to deny instead of stating the rule, so it fails open for any user group added later.

Absent or malformed configuration parses to an empty list, which is the "nobody marked anything" case and preserves current behaviour. A name in the list that is not in the registry is inert — the list is a filter, not a source of groups.

### Make the role a required parameter, not an optional refinement

`GetUserUsableGroups`, `GroupInUserUsableGroups`, `IsUserSelectableGroup`, `GetUserAutoGroup` and `FilterUserTokenAutoGroups` gain a role parameter, and `GetRequestAutoGroups` reads the role from the request context. No role-blind variant is kept.

The point is the compiler. There are six call sites across controllers and middleware, and this feature fails silently if one is missed — the leak is a group name in a JSON response that nobody looks at. Adding `GetUserUsableGroupsForRole` beside the existing function would leave the leaky function callable and would make the next new call site default to leaking. Changing the signature makes every existing and future call site state a role or fail to build.

`GetGroups` (the administrator registry behind `AdminAuth`) does not call this path at all and is intentionally left returning every group — an administrator still has to be able to attach an admin-only group to a channel or a user.

### Filter last, inside the resolver

The filter runs at the end of `GetUserUsableGroups`, after the base map, after the `GroupSpecialUsableGroup` overrides, and after the "grant the requester their own user group" rule. Filtering earlier would let an override with a `+:` entry, or the own-group rule, put the group back — the exact holes the spec names as scenarios.

This also settles the awkward case by construction: a non-administrator whose own user group is marked admin-only ends up with that group removed, so their relayed requests are refused. That is a deliberate choice of consistency over convenience — the alternative, exempting the requester's own group, means the flag means "hidden unless you are assigned to it", which is not a security property and is much harder to explain. The consequence is stated in the settings UI where the flag is set, so an operator meets it before it bites; the UI does not query how many users sit in the group, because that is a database round trip on a settings screen to soften a warning that is true regardless of the count.

### Carry the role on the context through `UserBase.WriteContext`

Add `ContextKeyUserRole` with the underlying key `"role"` — the same key `setDashboardAuthContext` already sets — and set it inside `UserBase.WriteContext`. Both entry points call that one function (`setDashboardAuthContext` calls `user.WriteContext(c)`; `TokenAuth` calls `userCache.WriteContext(c)`), so one line covers the dashboard and the relay. Choosing the same underlying key means the existing `c.Set("role", …)` and the new typed accessor address the same value and cannot disagree.

An anonymous request sets nothing, and `common.GetContextKeyInt` yields `0`, which is `RoleGuestUser`. Anonymous therefore fails closed with no special case — which is what `/api/user/groups` and the public pricing page need.

The alternative, threading the role explicitly from each controller, was rejected for the relay path specifically: `GetRequestAutoGroups` is called deep in relay handling, far from the middleware that knows the role, and passing it down would touch far more code than one context key.

### The administrator threshold is `RoleAdminUser`

`role >= common.RoleAdminUser` — so administrators and root users pass, guests and ordinary users do not. This is the same comparison `authHelper` already uses for `AdminAuth`, so "admin-only group" means the same "admin" the rest of the product means.

### One control in the existing group editor

The admin-only flag becomes a boolean on `GroupPricingRow`, read in `buildGroupPricingRows` from the new option and written in `serializeGroupPricingRows` back to it, with a checkbox column beside the existing `User selectable` column and a row in `GroupDetailSheet`. It is set in the same place a group is created, so "create a group that only admins can use" is one form.

New strings go into both locale files keyed by their English source, as the frontend rules require. The warning about assigned users is one of those strings, placed with the control rather than in a separate help surface.

## Risks / Trade-offs

- **A missed call site leaks silently** → This is the whole reason the signature changes rather than gaining a sibling. After the change, `grep` for the old arity should return nothing and the build is the guard. The verification step must additionally walk each of the six known surfaces and confirm the response for an ordinary user, not merely that the code compiles.
- **A non-administrator assigned to an admin-only group loses all access** → Accepted and specified, not an accident. Mitigated by stating the consequence at the control, and by the fact that the refusal is the existing forbidden-group response rather than a silent failure. An operator who does this sees a clear error from the affected user's requests.
- **Existing keys break when an operator marks a live group** → This is the intended behaviour of the feature and is specified. It is also the reason enforcement belongs in `TokenAuth`: a change that only filtered the creation form would leave those keys working and the operator believing otherwise.
- **The role now travels on the relay context** → It is read-only and used for one comparison. The risk is a stale role from the user cache after a demotion; that cache is already the source of the group and the quota on the same request, so the flag inherits the freshness the rest of authorisation already has, and no new invalidation path is introduced.
- **The `auto` group and cross-group retry** → Auto resolution goes through `IsUserSelectableGroup`, so it inherits the filter. The verification must still exercise a key on `auto` while an admin-only group sits in the global order, because that path picks groups without the user naming any.
- **Two ways to hide a group now exist** (`GroupSpecialUsableGroup` with `-:`, and this flag) → They compose rather than conflict: `-:` removes per user group, the flag removes per role, and the flag runs last. The settings UI should not present them as alternatives.

## Migration Plan

None required. The new option defaults to an empty list, so a deployment that has never touched the feature resolves groups exactly as before. There is no schema change, no `AutoMigrate` change, and no change to the shape of an existing option, so the three-database verification matrix in `AGENTS.md` is not triggered by this change; the option is written and read through the existing options path.

Rollback is reverting the build. A configuration left behind by a rolled-back deployment is an unread option row and changes nothing.
