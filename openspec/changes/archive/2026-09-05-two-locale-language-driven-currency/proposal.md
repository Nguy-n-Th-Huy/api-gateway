## Why

A Vietnamese visitor reading the pricing page today sees dollars, then pays in Dong at the SePay top-up rate — two numbers they have to reconcile themselves. The gateway already knows both the visitor's interface language and the Dong-per-dollar rate it charges (`operation_setting.Price`, exposed as `status.price`), so the price shown should follow the language: Dong for Vietnamese, dollars for English, at exactly the rate the top-up uses.

This fork also serves two audiences, not seven. It still ships Simplified Chinese, Traditional Chinese, French, Russian and Japanese interface locales inherited from upstream, plus a backend message bundle in Chinese only. Each locale is a maintenance surface that nobody here reviews, and every fork-specific string added so far has landed in those five locales as untranslated English. Keeping exactly Vietnamese and English — in both the interface and the backend messages — makes the translation surface match the product.

## What Changes

- Currency presentation follows the interface language. With Vietnamese active, every price, balance and cost renders in Dong (`₫`, whole units) converted at `status.price`; with English active, in US dollars. The pricing page, the home-page pricing preview, wallet balances and usage costs all use the one shared conversion path.
- **BREAKING**: the administrator's currency selector loses the `CNY` and `CUSTOM` choices and offers two modes: *currency, following the interface language* and *tokens*. The stored `quota_display_type` option and the backend keep their existing values and shape; legacy `CNY`/`CUSTOM` values already stored are interpreted as currency mode.
- **BREAKING**: interface languages reduce to Vietnamese and English. The zh, zh-TW, fr, ru and ja locale files, their entries in the language switcher and profile preference, and the browser-language mapping for Chinese are removed. English stays the fallback.
- **BREAKING**: backend messages reduce to English and Vietnamese. A `vi.yaml` covering every key in `en.yaml` is added; `zh-CN.yaml` and `zh-TW.yaml` are removed; `Accept-Language` normalization maps `vi*` to Vietnamese, `en*` to English, everything else to the English default.
- Documentation of the supported languages (`AGENTS.md`, the i18n sync tooling, the French and Russian glossaries) is brought in line.
- **Out of scope**: Chinese-language comments and hardcoded log strings in Go source are left alone — they are not user-facing and purging them would put the fork in permanent conflict with upstream. Model-description locale data (`localized-text.ts`, the `Chinese` option in model metadata) is content about models, not an interface locale, and is unchanged. No change to the top-up flow or to how `Price` is set.

## Capabilities

### New Capabilities
- `web-currency-display`: which currency the interface presents amounts in, at what rate, and how the administrator's display mode interacts with it.
- `interface-locales`: which languages the interface and backend messages are available in, how a visitor's language is detected and stored, and what happens to a previously stored language that is no longer offered.

### Modified Capabilities
<!-- None. web-home-page consumes the pricing display through the shared helper and its own requirements do not change. -->

## Impact

- `web/src/lib/currency.ts`, `web/src/stores/system-config-store.ts`, `web/src/hooks/use-system-config.ts` — the language-driven display resolution.
- `web/src/features/pricing/components/dynamic-pricing-breakdown.tsx` — the one consumer that branches on `CNY`/`CUSTOM` directly.
- `web/src/features/system-settings/general/pricing-section.tsx` and the billing section registry — the reduced administrator selector.
- `web/src/i18n/config.ts`, `web/src/i18n/languages.ts`, `web/src/i18n/locales/` — the two-locale interface; `web/src/lib/localized-text.ts` and `web/src/features/channels/lib/channel-utils.ts` — remove the Chinese interface-language branches.
- `web/scripts/sync-i18n.mjs` — locale list and per-locale heuristics.
- `i18n/i18n.go`, `i18n/locales/` — the two-language backend bundle.
- `AGENTS.md` language lists; `docs/translation-glossary.fr.md` and `docs/translation-glossary.ru.md` removed.
- **No database change**, no new endpoint, no change to the shape of `GET /api/status`.
- Merge cost: removed locale files will reappear in upstream merges and must be dropped again each time. This is the price of the reduced surface and is called out in the design notes.
