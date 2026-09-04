## 1. Language-driven currency

- [x] 1.1 In `web/src/lib/currency.ts`, make `getDisplayMeta()` resolve the currency from the active i18next language: `vi` → `{ kind: 'currency', symbol: '₫', currencyCode: 'VND', exchangeRate: <status.price> }`; anything else → US dollars at rate 1. Keep `TOKENS` as the only non-currency branch; treat `USD`, `CNY` and `CUSTOM` all as currency mode.
- [x] 1.2 Carry `status.price` into the currency config (`web/src/hooks/use-system-config.ts`, `web/src/stores/system-config-store.ts`) so the helper reads it from the store it already uses. Guard against zero or missing with the same defensive pattern used for `usdExchangeRate`. Do not introduce a literal `1000` anywhere. ← (verify: grep the touched files for `1000`)
- [x] 1.3 Render Dong with `maximumFractionDigits: 0` and grouped digits, consistent with `formatVND()` in `web/src/features/wallet/lib/format.ts`; keep the existing per-thousand/per-million presentation for sub-unit prices.
- [x] 1.4 Update `web/src/features/pricing/components/dynamic-pricing-breakdown.tsx:253-256` to stop branching on `CNY`/`CUSTOM` and go through the shared helper. ← (verify: no consumer outside `currency.ts` inspects `quotaDisplayType` for a currency name)
- [x] 1.5 Confirm the pricing page, `features/home/components/sections/pricing-preview.tsx`, and the wallet all change currency when the language switches, with no reload. ← (verify)

## 2. Administrator display-mode selector

- [x] 2.1 In `web/src/features/system-settings/general/pricing-section.tsx`, reduce the selector to two modes — currency following the interface language (stores `USD`) and tokens (stores `TOKENS`). Remove the `CNY`/`CUSTOM` choices and the custom-symbol/custom-rate inputs from the form.
- [x] 2.2 Keep the zod schema accepting legacy `CNY`/`CUSTOM` on read so an existing instance loads its settings without a validation error, while the form only offers the two modes. ← (verify)
- [x] 2.3 Update `web/src/features/system-settings/billing/section-registry.tsx` and `parseCurrencyDisplayType` callers accordingly. Do not change any backend option name or value set.

## 3. Interface locales — frontend

- [x] 3.1 Delete `web/src/i18n/locales/{zh,zh-TW,fr,ru,ja}.json`.
- [x] 3.2 In `web/src/i18n/config.ts`, reduce `resources` and `supportedLngs` to `en` and `vi`; keep `fallbackLng: 'en'` and the detection order.
- [x] 3.3 In `web/src/i18n/languages.ts`, reduce `INTERFACE_LANGUAGE_OPTIONS` to Vietnamese and English; simplify `normalizeInterfaceLanguage`, `convertDetectedLanguage` (Vietnamese variants → `vi`, everything else passes through to i18next matching) and `toIntlLocale` (drop the `zhCN`/`zhTW` cases, keep the guard).
- [x] 3.4 Remove the Chinese interface-language branches in `web/src/lib/localized-text.ts` (the `zhcn`/`zhtw` → `zh` mapping for the *interface* language) and `web/src/features/channels/lib/channel-utils.ts:460`. Leave model-description locale matching for `zh` content intact — that is data, not interface language.
- [x] 3.5 Update `web/scripts/sync-i18n.mjs`: locale discovery, and the per-locale heuristics at lines ~229-234 that name `ja`/`zh`/`ru`/`fr`. Run `bun run i18n:sync` and confirm it reports only `en` and `vi`. ← (verify)
- [x] 3.6 Confirm the protected attribution key handled by the sync script (`footer.` + `newapi` + `.projectAttributionSuffix`) is untouched in `en.json` and `vi.json`. ← (verify)
- [x] 3.7 Update tests that enumerate locale codes (`web/src/features/auth/lib/auth-redirect.test.ts`, `web/src/lib/__tests__/localized-text.test.ts`) to the two-locale set, keeping the behaviors they assert.

## 4. Interface locales — backend

- [x] 4.1 Add `i18n/locales/vi.yaml` with a hand-written Vietnamese translation of every key in `i18n/locales/en.yaml` (237 keys). Keep placeholders such as `{{.Provider}}` byte-identical.
- [x] 4.2 Delete `i18n/locales/zh-CN.yaml` and `i18n/locales/zh-TW.yaml`.
- [x] 4.3 In `i18n/i18n.go`: bundle base `language.English`; load `en.yaml` and `vi.yaml`; constants `LangEn`, `LangVi`; `normalizeLang` maps `vi*` → `LangVi`, `en*` → `LangEn`, default `DefaultLang` (English); `SupportedLanguages()` returns the two.
- [x] 4.4 Search the Go tree for any reference to `LangZhCN`, `LangZhTW`, `"zh-CN"` or `"zh-TW"` in the i18n context and update or remove it. Leave Chinese literals passed directly to `errors.New` / `SysLog` alone. ← (verify: `go build ./...`)
- [x] 4.5 Add a Go test asserting the key set of `vi.yaml` equals the key set of `en.yaml`, so future upstream additions to `en.yaml` fail until translated.

## 5. Documentation

- [x] 5.1 Update `AGENTS.md` lines 46 and 50 to the new language lists (backend: en, vi; frontend: en (base/fallback), vi).
- [x] 5.2 Delete `docs/translation-glossary.fr.md` and `docs/translation-glossary.ru.md`. Leave `docs/translation-glossary.md`.

## 6. Tests

- [x] 6.1 Frontend test: with language `vi`, `getDisplayMeta()` yields `₫` at `status.price`; with `en`, `$` at 1; with `TOKENS` mode, tokens regardless of language; with legacy `CNY`/`CUSTOM` stored, currency mode following language.
- [x] 6.2 Frontend test: a Dong amount renders with no fractional part and grouped digits.
- [x] 6.3 Frontend test: `normalizeInterfaceLanguage('zhCN')` and `('fr')` return `'en'`; `convertDetectedLanguage('vi-VN')` resolves to `vi`; `convertDetectedLanguage('zh-CN')` no longer maps to a Chinese code.
- [x] 6.4 Backend test: `normalizeLang` for `vi`, `vi-VN`, `en-US`, `zh-CN`, `fr`, and empty.
- [x] 6.5 Use `stretchr/testify` `require`/`assert` in Go tests per `AGENTS.md`; keep frontend tests in module-local `__tests__/`.

## 7. Verification

- [x] 7.1 `go build ./...` clean. ← (verify)
- [x] 7.2 `go test ./i18n/... ./controller/...` pass. ← (verify)
- [x] 7.3 `cd relaykit && GOWORK=off go build ./...` clean. ← (verify)
- [x] 7.4 `cd web && bun run typecheck`, scoped lint on touched files, `bun run test`, and `bun run build` pass — the build matters here because locale imports were removed. ← (verify)
- [x] 7.5 Confirm no database change, no option rename, no change to the `GET /api/status` field set. ← (verify)
- [x] 7.6 Confirm no protected identifier (project or organization name) was altered anywhere in the diff, including inside the deleted locale files' replacements. ← (verify)
- [x] 7.7 Confirm no Chinese comment or hardcoded log string in Go source was touched — that is out of scope. ← (verify: `git diff --stat` shows no unrelated Go files)
