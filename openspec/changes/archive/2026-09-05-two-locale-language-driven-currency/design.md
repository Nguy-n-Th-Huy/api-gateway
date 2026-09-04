## Context

See `proposal.md` — Why.

Currency today is a single administrator setting. `quota_display_type` (`USD` | `CNY` | `TOKENS` | `CUSTOM`) is stored as an option, published on `GET /api/status`, read into `web/src/stores/system-config-store.ts`, and resolved in one place — `getDisplayMeta()` in `web/src/lib/currency.ts:188` — into a symbol and an exchange rate that every formatter then uses. One consumer, `dynamic-pricing-breakdown.tsx:253-256`, branches on the type directly instead of going through the helper.

The Dong-per-dollar top-up rate is `operation_setting.Price` (`setting/operation_setting/payment_setting_old.go:12`, default `1000`), already published as `status.price` and already consumed by `usePricingData()` (`features/pricing/hooks/use-pricing-data.ts:38`) and by the wallet as `TopupInfo.price`.

Interface language is i18next with seven locales (`web/src/i18n/config.ts`), `fallbackLng: 'en'`, detection order `localStorage` then `navigator`, and a custom `convertDetectedLanguage` that maps browser `zh-*` tags onto the project's non-standard `zhCN`/`zhTW` codes. `INTERFACE_LANGUAGE_OPTIONS` in `languages.ts` drives both the switcher and the profile preference. The backend bundle (`i18n/i18n.go`) is go-i18n with `language.Chinese` as bundle base, three YAML files, 237 keys, and `DefaultLang = en`.

## Goals / Non-Goals

**Goals:**
- One resolution point for currency, still `getDisplayMeta()`, now taking the language into account — no consumer branches on currency type itself.
- No backend contract change for currency: `quota_display_type` keeps its name, its stored values, and its place on `/api/status`.
- Locale removal that is complete — resources, options, detection, tooling, docs — so nothing references a language that no longer exists.

**Non-Goals:**
- No renaming of the `zhCN`/`zhTW` code convention elsewhere in the codebase beyond deleting the branches that produce them.
- No purge of Chinese comments or log strings from Go source.
- No change to model-description locale data, which is about the model, not the interface.

## Decisions

**Resolve currency inside `getDisplayMeta()` from `i18n.language`, not from a new store field.** The helper is already the single choke point every formatter goes through, and the language is already global state. Adding a second store field that mirrors the language would create two sources of truth that can drift. Consumers stay untouched — they keep calling the same formatters. Alternative considered: a React hook returning a currency context; rejected because many formatters are called outside React (constants, table cell renderers), and the helper already reads the store imperatively for that reason.

**Dong rate comes from `status.price`, never from a literal.** `1000` is the current default, but it is an administrator setting that can change, and the wallet already charges by it. Displaying at a different number than the top-up charges would recreate the exact mismatch this change exists to remove. The helper reads it from the same system-config store that already carries `usdExchangeRate`.

**Keep `quota_display_type` and reinterpret it, rather than replacing it.** The option name, the `/api/status` field, and the admin API all stay. The frontend collapses the four values to two behaviors: `TOKENS` is tokens; everything else is currency-following-language. Stored `CNY` or `CUSTOM` on an existing instance therefore keeps working without a migration. The admin selector writes `USD` for currency mode so the stored value stays one the backend already validates. Alternative considered: a new option `currency_mode`; rejected as a backend change with a migration for no user-visible gain.

**`₫` symbol, whole units, `Intl.NumberFormat` with `maximumFractionDigits: 0`.** This matches `formatVND()` in `features/wallet/lib/format.ts:42`, which already encodes "SePay settles exclusively in whole Dong". Sub-unit per-token prices keep the existing per-thousand/per-million presentation that dollar prices already use, so distinct prices remain distinguishable.

**Delete the five locales rather than hide them.** Hiding them in the switcher while shipping the JSON keeps a bundle cost and a maintenance surface nobody reviews; the sync tool would keep writing untranslated English into them forever. Deleting is the honest state. The cost — they reappear on upstream merges — is accepted and recorded below.

**A stored preference for a removed language is treated as absent.** `normalizeInterfaceLanguage()` already returns `'en'` for anything not in `INTERFACE_LANGUAGE_OPTIONS`; the profile preference path goes through it. For the i18next detector, `supportedLngs` shrinks to `['en', 'vi']` and `convertDetectedLanguage` is simplified so a stored `zhCN` no longer maps to anything, letting i18next fall to `fallbackLng`. The detector then caches the resolved language, replacing the stale value.

**Backend: add `vi.yaml` translated by hand, switch the bundle base to English, drop the Chinese files.** 237 keys is a bounded, one-time translation. go-i18n's bundle base language is the fallback for missing keys, so it must become English once Chinese leaves. `normalizeLang` maps `vi*` → `vi`, `en*` → `en`, default `en`. Hardcoded Chinese strings passed directly to `errors.New` or `SysLog` are outside the bundle and are left alone.

**`toIntlLocale()` stays.** With `zhCN`/`zhTW` gone it reduces to `Intl.getCanonicalLocales`, but it still guards every `Intl.*` constructor against an unexpected value, and `vi`/`en` pass through it unchanged.

## Risks / Trade-offs

- **Deleted locale files return on every upstream merge** → each merge drops them again along with their `config.ts` imports. This is a recurring, mechanical cost; it is the price of the smaller surface and should be part of the merge checklist.
- **Chinese-speaking operators of this fork lose their interface language** → intentional; this fork's audience is Vietnamese and English speakers. Upstream remains available to anyone who needs Chinese.
- **A price shown in Dong is only correct if `Price` is set correctly** → this was already true of the wallet. The go-live checklist in the market research report already flags the default `1000` as a retail rate the operator must confirm.
- **`getDisplayMeta()` now depends on i18next being initialized** → it already depends on the system-config store being populated, and the same defensive default (US dollars) applies when the language is not yet resolved.
- **Backend `vi.yaml` drifts as upstream adds keys to `en.yaml`** → a test asserts key-set equality between the two files, so a merge that adds an English key fails until the Vietnamese one is added.

## Migration Plan

Frontend-only bundle change plus a backend message bundle change; no schema, no option rename, no endpoint change. Rollback is a revert. An instance with `CNY` or `CUSTOM` stored continues to work in currency mode; an operator who wants tokens selects tokens as before. A visitor with a stored removed language sees the interface in their browser language on next load and nothing else.
