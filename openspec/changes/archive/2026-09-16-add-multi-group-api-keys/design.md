# Design: multi-group API keys

## Context

See `proposal.md` — Why. The mechanism this change puts on the console already exists end to end:

- `model/token.go` stores `Token.Group string`, `Token.CrossGroupRetry bool` and `Token.AutoGroups string` (a JSON array, `json:"-"`, `gorm:"type:text"`, read through `GetAutoGroups()` / written through `SetAutoGroups()`).
- `middleware/auth.go` `TokenAuth` (the `if tokenGroup != ""` block, ~455-490) currently requires a token's group to be present in `service.GetUserUsableGroups(userGroup, userCache.Role)` and then requires `ratio_setting.ContainsGroupRatio(group)` unless the group is `auto`; it then sets `constant.ContextKeyUsingGroup`. `SetupContextForToken` (~498) sets `ContextKeyTokenGroup`, `ContextKeyTokenCrossGroupRetry`, and — only when `token.AutoGroups != ""` — `ContextKeyTokenAutoGroups`. A snapshot that fails to parse is recorded as a present, empty slice, so routing fails closed instead of silently inheriting the global order.
- `service/group.go`: `GetRequestAutoGroups(c, userGroup)` returns the token snapshot filtered by `FilterUserTokenAutoGroups` when the context value is present, and the global Auto list otherwise; `IsUserSelectableGroup` rejects `""` and `"auto"` and requires usable ∧ `ContainsGroupRatio`; `FilterUserTokenAutoGroups` drops groups that are no longer selectable and truncates to `setting.GetMaxTokenAutoGroups()`.
- `service/channel_select.go` `CacheGetRandomSatisfiedChannel` (~116-170): for `param.TokenGroup == "auto"` it walks the candidate list in order, skips a group with no expected channel for the model, honours `ContextKeyTokenCrossGroupRetry` to advance past an exhausted group, and records the group that served the request in `ContextKeyAutoGroup`.
- `controller/model.go` `getModelListGroups`: for `tokenGroup == "auto"` the owner's groups are `service.GetRequestAutoGroups`, i.e. the union of the snapshot's models. `relay/helper/price.go` `HandleGroupRatio` overwrites `relayInfo.UsingGroup` with `ContextKeyAutoGroup`, and `service/quota.go` (`PreWssConsumeQuota`, ~110, which re-reads `ContextKeyAutoGroup` and overrides `relayInfo.UsingGroup`) together with `service/task_billing.go` (line ~77, which stores `info.UsingGroup` as the task's `Group`) price and record from that value.
- `controller/token.go`: `setTokenAutoGroups` validates a snapshot (count ≤ `setting.GetMaxTokenAutoGroups()`, no duplicates, `service.IsUserSelectableGroup` per group) and is invoked only when `token.Group == "auto"`; `AddToken` and `UpdateToken` otherwise clear `CrossGroupRetry` and the snapshot. `buildMaskedTokenResponse` exposes `auto_groups` as `[]string`. `GetTokenAutoGroups` serves `GET /api/token/auto-groups` → `{groups: service.GetUserAutoGroup(userGroup, role), max_count: setting.GetMaxTokenAutoGroups()}`. `controller/group.go` `GetUserGroups` serves `GET /api/user/self/groups` with the requester's usable groups plus `auto` only when `auto` is usable.
- `web/src/features/keys/` holds the current console: the form model is `group: string` / `auto_groups_mode` / `auto_groups` / `cross_group_retry` in `lib/api-key-form.ts`, the drawer renders a single-select `ApiKeyGroupCombobox` and only for `auto` an `AutoGroupOrderEditor` plus the cross-group switch, and the table's group column renders `ApiKeyGroupCell`.

Because everything downstream already keys off `group == "auto"` + snapshot, the only backend obstacle is the authentication gate: it refuses `auto` for an owner whose usable groups do not contain `auto`, and `service.IsUserSelectableGroup` deliberately rejects `"auto"` (so it can never be inside a snapshot). Multi-group selection is therefore unreachable for such an owner today.

## Goals / Non-Goals

**Goals**

- One console control for a key's groups, usable by any owner regardless of whether `auto` is in their usable groups.
- Zero schema change: the existing `group` / `auto_groups` / `cross_group_retry` columns keep their meaning and every existing consumer keeps working.
- Keep per-group authorisation exactly as strong as it is today, at both write time and request time.

**Non-Goals**

- No new pseudo-group, no change to the `auto` pseudo-group's own meaning, and no change to fine-grained retry/affinity inside a single group.
- No change to `GET /api/token/auto-groups`, `POST /api/token/`, `PUT /api/token/` payload shapes, or to `GET /api/user/self/groups`.
- Backend work is limited to the single authentication gate. Channel selection, model-list union, billing, logging and task rows are **not** modified by this change — they already implement multi-group routing; see Decisions D3.

## Decisions

### D1 — Storage stays as-is; the console maps selection onto it

The form's ordered selection is mapped onto the existing columns, with no new column and no migration:

| Form state | `group` | `auto_groups` | `cross_group_retry` |
| --- | --- | --- | --- |
| one group | that group | `""` | `false` |
| two or more groups | `"auto"` | ordered snapshot | form switch (default `true`) |
| global Auto order | `"auto"` | `""` | form switch |
| none | `""` | `""` | `false` |

Rationale: `group = "auto"` with a snapshot is exactly what `service/channel_select.go` already walks, so multi-group needs no new routing code and no new persistence. Alternatives considered: a dedicated `groups` column (requires a migration, a new cache shape in `model/token.go`, and a parallel path through `service/group.go` — rejected as strictly more risk for no capability gain); reusing `group = "auto"` for a single group (rejected: it would make every existing single-group key flow through the Auto path and change its billing/retry surface for no reason).

### D2 — The authentication gate accepts `auto` + a valid snapshot

`middleware/auth.go` `TokenAuth` remains the only production backend change. In the `if tokenGroup != ""` block, the usable-group check is relaxed for exactly one case:

```go
if _, ok := service.GetUserUsableGroups(userGroup, userCache.Role)[tokenGroup]; !ok {
    // A multi-group key is stored as group == "auto" plus an ordered snapshot.
    // The snapshot names only groups the owner may select (validated at write
    // time by controller/token.go setTokenAutoGroups and re-filtered at request
    // time by service.FilterUserTokenAutoGroups), so it authorises the key's own
    // groups; the "auto" placeholder itself needs no authorization in that case.
    if tokenGroup != "auto" || !token.HasAutoGroupsSnapshot() {
        abortWithOpenAiMessage(c, http.StatusForbidden, fmt.Sprintf("无权访问 %s 分组", tokenGroup))
        return
    }
}
```

`Token.HasAutoGroupsSnapshot()` (new, `model/token.go`) is the single definition of "carries a usable snapshot": it parses `AutoGroups` with `common.Unmarshal` (repo rule: no direct `encoding/json`) and reports `len(groups) > 0`; it returns `false` on a parse error, on an empty string, and on an empty array. Rationale for a method rather than an inline parse: the gate and `SetupContextForToken` must agree on what a usable snapshot is, and the same predicate is what the design's tests can drive directly.

Everything else in the block is untouched: the `ratio_setting.ContainsGroupRatio` check still runs for the resolved group (its `auto` exception already exists), the admin-only check on the effective group still runs, and the request-time re-filtering in `GetRequestAutoGroups` still removes any snapshot group the owner may no longer select. Alternatives considered: allowing `auto` unconditionally when a snapshot is stored (rejected: an unparsable or empty snapshot would then become a bypass); authorising the snapshot at the gate by iterating it (rejected: duplicates `FilterUserTokenAutoGroups`, which already runs on the request path with the same role).

Net effect: a global-Auto key (`auto`, no snapshot) for an owner without `auto` usable is still refused with the same 403 as today, and a malformed snapshot is treated as no snapshot, also 403.

### D3 — The rest of the relay path is already correct and is not touched

Recorded explicitly so it is not mistaken for missing coverage:

- **Channel selection** — `service/channel_select.go` `CacheGetRandomSatisfiedChannel`: for `TokenGroup == "auto"` it resolves the candidates via `GetRequestAutoGroups`, walks them in order, skips a group with no expected channel for the model, advances to the next group only when `ContextKeyTokenCrossGroupRetry` is set and the current group has exhausted `common.RetryTimes`, and stores the winner in `ContextKeyAutoGroup`. Nothing in this change alters it.
- **Model list** — `controller/model.go` `getModelListGroups` uses `service.GetRequestAutoGroups` for `tokenGroup == "auto"`, so the model list offered for a multi-group key is the union over its snapshot, in order.
- **Billing and records** — `relay/helper/price.go` `HandleGroupRatio` sets `relayInfo.UsingGroup` from `ContextKeyAutoGroup`, `service/quota.go` `PreWssConsumeQuota` (~110) applies the same override before pricing, and `service/task_billing.go` (~77) stores `info.UsingGroup` as the task's group, so the request is priced by, and its log/task rows name, the group that actually served it.
- **Per-group authorisation** — already enforced twice: at write time by `service.IsUserSelectableGroup` in `controller/token.go` `setTokenAutoGroups`, and at request time by `service.FilterUserTokenAutoGroups` inside `service.GetRequestAutoGroups`. The gate relaxation in D2 does not weaken either.

### D4 — The frontend form model is replaced outright

In `web/src/features/keys/lib/api-key-form.ts`:

- Schema (`getApiKeyFormSchema(t, maxAutoGroups)`): `group`, `auto_groups_mode` and `auto_groups` are removed; `groups: z.array(z.string())` and `use_global_auto: z.boolean()` are added; `cross_group_retry: z.boolean()` stays. The `superRefine` block for `auto` is replaced by: when `use_global_auto` is true, the list checks are skipped (the list is empty by construction); otherwise the list must not exceed `maxAutoGroups` and must not contain duplicates. The existing "select at least one group" rule is dropped because an empty selection is now a legal, meaningful state (the key follows its owner's group).
- Defaults: `API_KEY_FORM_DEFAULT_VALUES` becomes `groups: []`, `use_global_auto: false`, `cross_group_retry: false`; `getApiKeyFormDefaultValues(defaultUseAutoGroup)` returns those with `use_global_auto: defaultUseAutoGroup` and `cross_group_retry: defaultUseAutoGroup`, which preserves today's behaviour where the `auto` default also turns retry on. `DEFAULT_GROUP` is removed from `constants.ts`, so no single-group default is injected — matching today's behaviour, where the default form value is also the empty group.
- `transformFormDataToPayload(data)`:
  - `use_global_auto` → `{ group: 'auto', auto_groups: [], cross_group_retry: !!data.cross_group_retry }`;
  - 0 groups → `{ group: '', auto_groups: [], cross_group_retry: false }`;
  - 1 group → `{ group: groups[0], auto_groups: [], cross_group_retry: false }`;
  - ≥2 groups → `{ group: 'auto', auto_groups: groups, cross_group_retry: !!data.cross_group_retry }`.
- `transformApiKeyToFormDefaults(apiKey, availableGroups = [], maxAutoGroups = 5)` (the second parameter is renamed from `availableAutoGroups` because it is now the selectable group list, and its source is unchanged: `GET /api/user/self/groups` minus `auto`, as computed in the drawer):
  - `group === 'auto'` with a non-empty stored snapshot → `use_global_auto: false`, `groups` = the snapshot filtered against `availableGroups` and truncated to `maxAutoGroups`;
  - `group === 'auto'` with an empty or absent snapshot → `use_global_auto: true`, `groups: []`;
  - any other non-empty group → `use_global_auto: false`, `groups` = that group filtered against `availableGroups`;
  - `group === ''` → `use_global_auto: false`, `groups: []`.
- `ApiKeyFormData` in `web/src/features/keys/types.ts` keeps its `group` / `auto_groups` / `cross_group_retry` members unchanged — the HTTP contract does not move.

Rationale for deleting rather than aliasing: `group` and `groups` cannot both be authoritative, and a compatibility path would leave two write paths for the same payload. The payload shape itself is unchanged, so the backend needs no coordinated change.

### D5 — The picker is the existing ordered editor; the global Auto order becomes a toggle

`web/src/features/keys/components/auto-group-order-editor.tsx` keeps its ordered-list machinery (add via `ApiKeyGroupCombobox`, `Reorder.Group` drag, up/down, remove, counter, limit) and loses its two-mode branching: props become `{ value: string[], options: ApiKeyGroupOption[], maxCount: number, onChange: (groups: string[]) => void }` plus the existing passthrough DOM props (drop `mode`, `globalOptions`, and the `{groups, mode}` wrapper of `onChange`). Its inherit affordances — the "Restore global Auto" button and the read-only global order list — move to a second export in the same module, `GlobalAutoOrderPreview`, which renders the same `data-slot="global-auto-order"` / `global-auto-order-name` markup from `globalOptions` (the order and ratio chips already styled there).

The drawer (`components/api-keys-mutate-drawer.tsx`) then owns the mode:

- A `use_global_auto` `FormField` (a `Switch`, label "Use the global Auto order") rendered only when `backendHasAuto` (already computed from the `user-groups` query).
- A `groups` `FormField` rendering `AutoGroupOrderEditor` — the picker is rendered for every key, not only for `auto`, and is what the "Group" field was.
- When `use_global_auto` is on, the picker is replaced by `GlobalAutoOrderPreview` fed by `globalAutoGroupOptions` (already computed from `GET /api/token/auto-groups` filtered to the usable groups), so the operator still sees which groups the key will try; turning the toggle on clears `groups`, and while it is on the picker is not rendered at all, so no picker edit can occur — turning the toggle off restores the picker with its (emptied) selection.
- The `cross_group_retry` `FormField` is shown when `groups.length >= 2` or `use_global_auto` is on (today: only when the group is `auto`), and the switch is turned on when the selection becomes able to span more than one group — the same thing today's `auto` selection does — with the requester free to turn it off.

Rationale: one control with two modes beats two competing controls, and it is what makes the "offered only when `auto` is usable" rule expressible — inside the old editor the "Restore global Auto" affordance was unconditional, so it could not honour that rule. Alternatives considered: keeping `mode` inside the editor (rejected: the editor would then need a "global Auto allowed" prop to hide an affordance, and the form would carry both `use_global_auto` and `mode`); hiding the picker entirely when the toggle is on (rejected: the operator would lose sight of the order the key will follow).

### D6 — The keys table renders the key's real groups

`components/api-key-group-cell.tsx`: props become `{ group: string, groups: string[], ratio?: GroupRatio, groupRatios?: Record<string, GroupRatio>, shouldReduceMotion: boolean }` — the currently-dead `crossGroupRetry` prop is removed.

- `group !== 'auto'` → unchanged (`TruncatedCell` + `GroupBadge`).
- `group === 'auto'` with `groups.length > 0` → inside the existing `BadgeCell`, the cross-group `StatusBadge` (`t('Cross-group')`) plus one `GroupBadge` chip per snapshot group in stored order, each carrying that group's ratio from `groupRatios`; the tooltip content lists the stored order (`g1 → g2 → ...`), so the order is readable without opening the form. The Auto ratio badge is not rendered here, because the ratio is now per group.
- `group === 'auto'` with no snapshot → today's rendering (cross-group badge + `GroupRatioBadge isAuto` + the existing tooltip copy).

`components/api-keys-columns.tsx`'s group column passes `groups={apiKey.auto_groups ?? []}` and `groupRatios={groupRatios}` (both already available in `useApiKeysColumns` via `useGroupRatios()`), and no longer passes `crossGroupRetry`.

### D7 — Translations

New or reworded strings live in `web/src/i18n/locales/en.json` and `web/src/i18n/locales/vi.json` (flat keys under `translation`, English source strings as keys, per `AGENTS.md`), and follow `docs/translation-glossary.md`. The set this change needs, with existing keys reused where the wording still fits:

- reused: `Groups`, `Add group`, `{{count}} / {{max}} groups selected`, `Maximum {{max}} groups selected`, `Cross-group`, `Cross-group retry`, its description, `Ratio`, `Automatically selects the best available group with circuit breaker mechanism`;
- new: the picker's description ("Choose the groups this API key will try, in order."), the global-Auto toggle label and its description, the empty-picker copy that states what saving an empty selection means, the per-group-order tooltip copy, and the validation messages for "too many groups" and "duplicate groups" if their existing `Auto`-specific wording is not reused.

No existing locale key is deleted: the locale files are shared across features and an unreferenced key is harmless, whereas a wrongly removed one breaks another surface. The implementer confirms each final string exists in both files (e.g. `jq -r '.translation | keys[]'` per file) rather than assuming.

## Risks / Trade-offs

- **The gate relaxation widens what authenticates.** → It only bypasses the `auto` placeholder's own usable-group check, and only when `Token.HasAutoGroupsSnapshot()` is true; the snapshot's names are validated at write time and re-filtered at request time by `FilterUserTokenAutoGroups`, and an unparsable/empty snapshot is treated as absent, so a global-Auto key for such an owner still 403s. Tests must cover all four cases (valid snapshot without `auto` usable, no snapshot without `auto` usable, malformed snapshot without `auto` usable, existing single-group and ungrouped keys unchanged).
- **A key whose snapshot no longer contains any selectable group becomes uneditable into a working state by accident.** → Such a key is already refused at request time, and the form's empty-picker copy states that saving then leaves the key on its owner's group; the global-Auto toggle or adding a group recovers it. The behaviour mirrors the existing snapshot filtering, so it introduces no new class of loss.
- **A single-group key naming a group that is no longer selectable renders with an empty picker and would be saved with no group.** → Same trade-off as above and the same mitigation; the alternative (passing an unknown value through) would let the form save a group the owner may not select, which the write path cannot reject for non-`auto` groups today.
- **Two limits could drift** (the picker's `maxCount` and `setting.GetMaxTokenAutoGroups()`). → The picker reads `max_count` from `GET /api/token/auto-groups`; the write path keeps enforcing the setting and returns the existing too-many-groups error, so the backend is authoritative.
- **Chip overflow in a 220px column.** → The cell reuses `BadgeCell`/`TruncatedCell` and `GroupBadge`'s truncation; the order stays fully readable in the tooltip.
- **The editor's public props change** (`mode`/`globalOptions`/`onChange` shape), touching `auto-group-order-editor.test.tsx`. → Rewrite those tests against the new props rather than deleting them; they pin real behaviour (limit, remove, reorder, add, candidate exclusion).
- **A stale stored `group` for a key created before this change is still reported as-is** by the key-check report, the setup script and the Telegram payloads. → Deliberate non-goal; those surfaces keep reporting `group`, which remains a faithful summary (`auto` means "several or the global order").

## Migration Plan

- No database migration, no data backfill: the columns and their meaning are unchanged, and every existing key keeps working (`group = 'auto'` + snapshot, `group = 'auto'` alone, a single group, and an empty group all retain their current behaviour).
- Deployment order: the backend gate change is additive and safe to ship first or together; the new console is only useful with it in place, so ship them in the same release. No frontend/backend payload change means a mixed-version window is harmless (an old console keeps producing the old payloads, which the new backend accepts unchanged).
- Rollback: revert the form model, the picker and the table cell to the previous console, and restore the gate. Keys stored as `group = 'auto'` with a snapshot keep working for owners who may select `auto`; for an owner without `auto` usable they would be refused again (403) by the restored gate — that is the only user-visible cost, and it does not affect any key that existed before this change.
- Not to be touched: `relaykit/`, the database schema and its migrations, and any file outside `web/src/features/keys/`, `web/src/i18n/locales/`, `middleware/auth.go` and `model/token.go`.

## Open Questions

- The final wording of the new interface strings is left to implementation, constrained by the glossary and by the requirement that every string exists in both `en.json` and `vi.json`.
- Whether the empty-picker copy should also be shown when the selection is empty because a stored group was filtered out (rather than because the operator cleared it) is a copy-level choice; the required behaviour (an explicit statement of what saving means) is the same either way.
