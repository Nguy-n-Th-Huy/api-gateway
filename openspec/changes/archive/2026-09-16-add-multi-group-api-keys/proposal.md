# Add multi-group API keys

## Why

An API key can only target one group from the console. Multi-group routing already exists internally — a key with `group = "auto"` plus a stored `auto_groups` snapshot walks those groups in order — but the console only reaches it by selecting the `auto` pseudo-group and editing a separate "Auto group order" control, and the whole path requires the operator to have added `auto` to the user's usable groups. When `auto` is not usable, the key form can only ever hold one group, so a user who wants "try `vip`, then `default`" has no way to express it.

The change replaces that indirect path with one ordered group picker on the key form: the groups a key may use are chosen directly, in order, and the storage and routing that already exist are reused unchanged.

## What Changes

- The key form's single-select "Group" field becomes one always-visible ordered group picker (add/remove/reorder groups, capped at the existing per-key group limit). Selecting one group keeps producing today's single-group key; selecting several produces a multi-group key that walks the chosen groups in the chosen order.
- Keys that follow the administrator's global Auto order remain expressible, as an explicit "Use the global Auto order" toggle offered only when `GET /api/user/self/groups` offers `auto`.
- The stored representation is unchanged and gained no schema: one group → `group = <g>`, empty snapshot, cross-group retry off; two or more groups → `group = "auto"` with an ordered `auto_groups` snapshot and cross-group retry from the form switch; global Auto order → `group = "auto"` with an empty snapshot. No database column is added and no migration is written.
- **BREAKING** (backend behaviour, one gate): token authentication SHALL accept a key whose group is `auto` when the key carries a non-empty group snapshot, even if `auto` is not in the owner's usable groups. Today that combination is refused with 403, which is what makes a multi-group key impossible to use without the `auto` operator step. A key whose group is `auto` with no snapshot is still refused when `auto` is not usable, and a malformed snapshot fails closed.
- The keys table's Group column SHALL show the key's real groups: a multi-group key renders its ordered groups with their ratios and the cross-group indicator, instead of the single "Cross-group" badge it shows today.
- The frontend form model becomes an ordered `groups: string[]` plus a `use_global_auto` toggle and the existing `cross_group_retry` switch; the single `group` field, the `auto_groups_mode` flag and the `auto_groups` form field are removed outright, with no compatibility alias.
- The HTTP contract is untouched: `POST /api/token/` and `PUT /api/token/` keep the `{group, auto_groups, cross_group_retry}` payload shape, `GET /api/token/auto-groups` keeps `{groups, max_count}`, and the group limit still caps a key's group count.

## Capabilities

### New Capabilities

- `api-key-groups`: how an API key's groups are chosen in the console (ordered, one or many, or the global Auto order), how that choice is stored on the key, how it is authorised at write time and at request time, and what the keys table shows for it.

### Modified Capabilities

None. The behaviour of admin-only groups (including their exclusion from automatic group selection and their refusal at request time) is preserved as specified in `openspec/specs/admin-only-groups/spec.md`, and no requirement of that capability changes. The interface language requirements follow `openspec/specs/interface-locales/spec.md` unchanged.

## Impact

- Backend, one gate: `middleware/auth.go` (`TokenAuth` group/gate block) plus its `model/token.go` helper `Token.HasAutoGroupsSnapshot()`, the single definition of "carries a usable snapshot". Every other backend surface already handles `group = "auto"` with a snapshot: `service/channel_select.go` `CacheGetRandomSatisfiedChannel` walks the ordered candidate list and records the serving group, `controller/model.go` `getModelListGroups` unions the models of the snapshot's groups, and `relay/helper/price.go` `HandleGroupRatio` together with `service/quota.go` and `service/task_billing.go` bill with the group that actually served the request (the logs and task rows record that same group). Per-group authorisation is already enforced twice: at write time by `service.IsUserSelectableGroup` inside `controller/token.go` `setTokenAutoGroups`, and at request time by `service.FilterUserTokenAutoGroups` inside `service/group.go` `GetRequestAutoGroups`.
- Frontend: `web/src/features/keys/` — `lib/api-key-form.ts` (schema, defaults, both transforms), `constants.ts` (`DEFAULT_GROUP` removed), `components/api-keys-mutate-drawer.tsx` (picker wiring), `components/auto-group-order-editor.tsx` (reused as the picker), `components/api-key-group-cell.tsx` and `components/api-keys-columns.tsx` (table rendering).
- Tests: `middleware/token_multi_group_auth_test.go` (new), `web/src/features/keys/components/__tests__/{api-keys-mutate-drawer,api-key-group-cell,auto-group-order-editor}.test.tsx`, `web/src/features/keys/lib/__tests__/auto-group-form.test.ts`.
- Translations: `web/src/i18n/locales/en.json` and `web/src/i18n/locales/vi.json`, following `docs/translation-glossary.md`.
- Untouched by design: the database schema, `relaykit/`, the key-check report, the setup script, the Telegram bot payloads (they keep reporting the stored `group`), the playground's group selector, channel-side group configuration, and the fine-grained retry/affinity behaviour of `auto`.
