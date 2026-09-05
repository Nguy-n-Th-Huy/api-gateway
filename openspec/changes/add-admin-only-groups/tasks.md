## 1. The setting

- [ ] 1.1 Add an admin-only groups option in `setting/`, following the shape of `setting/auto_group.go`: a JSON-string updater, a JSON-string serializer, a getter returning the list, and a membership check for a single group name. Malformed or absent JSON must yield an empty list, never an error that blocks option loading.
- [ ] 1.2 Register the option in `model/option.go` — add it to `common.OptionMap` beside `AutoGroups`, and add its case to the option-update switch so a settings save persists and applies it. ← (verify: saving the option from the settings screen persists it and takes effect without a restart; an unset option loads as empty)

## 2. Role on the request context

- [ ] 2.1 Add `ContextKeyUserRole` to `constant/context_key.go` with the underlying key `"role"`, matching the key `setDashboardAuthContext` already sets.
- [ ] 2.2 Set the role in `model/user_cache.go` `UserBase.WriteContext`, so both the dashboard path (`setDashboardAuthContext`) and the relay path (`TokenAuth`) populate it from one place. Leave the existing `c.Set("role", …)` alone. ← (verify: an authenticated dashboard request and a relayed API-key request both read back the owner's role; an anonymous request reads 0)

## 3. Role-aware group resolution

- [ ] 3.1 Change `service.GetUserUsableGroups` in `service/group.go` to take the requester's role, and apply the admin-only filter as the last step — after the base map, after the `GroupSpecialUsableGroup` overrides, and after the rule that grants the requester their own user group. Do not keep a role-blind variant.
- [ ] 3.2 Thread the role through `GroupInUserUsableGroups`, `IsUserSelectableGroup`, `GetUserAutoGroup` and `FilterUserTokenAutoGroups`, and have `GetRequestAutoGroups` read the role from the request context.
- [ ] 3.3 Add table tests beside `service/group.go` covering: administrator sees admin-only groups; ordinary user does not; role 0 does not; a `+:` override naming an admin-only group does not restore it; the requester's own user group being admin-only does not restore it; an admin-only name that is not in the registry changes nothing; an empty configuration leaves resolution byte-identical to today. Use `testify` `require`/`assert` and restore global settings in the fixture teardown, as the neighbouring group tests do. ← (verify: each test fails if the filter is moved earlier in the resolver or removed)

## 4. Call sites

- [ ] 4.1 `controller/group.go` — pass the requester's role. The unauthenticated route must resolve as role 0 rather than assume a session.
- [ ] 4.2 `controller/pricing.go` — pass the requester's role, so the filtered model rows, the `group_ratio` map, the `usable_group` map and the `auto_groups` list are all resolved for that role. An anonymous request resolves as role 0.
- [ ] 4.3 `controller/user.go` `GetUserModels` — pass the role of the user whose models are being listed.
- [ ] 4.4 `controller/token.go` — pass the role in `setTokenAutoGroups` and `GetTokenAutoGroups`, so an ordinary user is neither offered nor allowed to store an admin-only group in a key's automatic-selection order.
- [ ] 4.5 `middleware/auth.go` `TokenAuth` — make the existing token-group check use the key owner's role, keeping the existing forbidden-group response unchanged.
- [ ] 4.6 `middleware/distributor.go` — make the playground group override use the requester's role, keeping the existing access-denied response unchanged.
- [ ] 4.7 Confirm `GetGroups` (the administrator registry behind `AdminAuth`) is untouched and still returns every group. ← (verify: the build has no remaining call to the old arity anywhere, and an administrator can still attach an admin-only group to a channel and to a user)

## 5. Group editor

- [ ] 5.1 Add the admin-only flag to the group row model in `web/src/features/system-settings/models/group-ratio-visual-editor.tsx`, reading it in `buildGroupPricingRows` and writing it in `serializeGroupPricingRows` through the same `onChange(field, value)` mechanism the other options use.
- [ ] 5.2 Add the checkbox column beside the existing `User selectable` column, with an accessible label, and show the flag's state in `GroupDetailSheet`.
- [ ] 5.3 Wire the new option value into the editor from its parent form/section so it round-trips on save, and confirm the group signature comparison used to detect unsaved changes accounts for the new field. ← (verify: marking a group admin-only, saving, and reloading the settings screen shows the flag still set; the unsaved-changes indicator reacts to toggling it)
- [ ] 5.4 State at the control that marking a group admin-only makes it unusable and invisible to every non-administrator, including a user whose own user group is that group. No user count, no database lookup.
- [ ] 5.5 Add every new string to `web/src/i18n/locales/en.json` and `web/src/i18n/locales/vi.json`, keyed by the English source string and rendered through `t()`. No literal UI text in the component. ← (verify: each new key exists in both locales and the editor is fully translated in Vietnamese)

## 6. Verification

- [ ] 6.1 Run the Go build and the touched backend test packages; run typecheck, lint scoped to the touched frontend files, and the frontend test suite from `web/`. Report the real results without suppressing failures.
- [ ] 6.2 Exercise the six surfaces against a running instance with one group marked admin-only, as an ordinary user and then as an administrator: the group list behind the API-key form, the public and authenticated model catalogue, the per-user model list, the automatic-selection order editor, a relayed request with a key on that group, and a playground request naming that group. ← (verify: for the ordinary user, no response contains the group's name, description or ratio anywhere, and the relayed request is refused; for the administrator, every surface behaves as before)
- [ ] 6.3 Exercise the two time-shifted cases: a key created on a group that is marked admin-only afterwards must start being refused, and a key on `auto` belonging to an ordinary user must skip an admin-only group that sits in the global automatic-selection order. ← (verify: neither case depends on the user re-saving anything)
- [ ] 6.4 Confirm the no-op case: with the option empty, every one of those surfaces returns what it returned before the change, for both roles.
