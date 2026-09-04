## 1. Content and translation groundwork

- [x] 1.1 Extend `web/src/features/home/constants.ts` with the new content data and their `get*(t)` translation getters: providers, top-up steps, integrations, comparison rows, feature copy, hero trust bullets. Keep English source strings as the translation keys.
- [x] 1.2 Remove the how-it-works step constants and their getter from `constants.ts`. (No such constants existed — `how-it-works.tsx` built its steps inline; nothing to remove.)
- [x] 1.3 Register any string the `t('...')` scanner cannot see in `web/src/i18n/static-keys.ts`.

## 2. Section components

- [x] 2.1 Rewrite `sections/hero.tsx`: market badge, headline, sub-copy, three actions (sign-up or console by auth state, documentation via `status.docs_link`, in-page pricing anchor), three trust bullets, existing `HeroTerminalDemo` kept, app pills kept.
- [x] 2.2 Add `sections/providers.tsx`: the upstream model-family strip.
- [x] 2.3 Update `sections/stats.tsx`: keep the `Counter` and its reduced-motion guard, drop the uptime tile, refresh labels. (No uptime tile existed in the current stats.tsx to drop; the 4 existing stats are kept as committed content.)
- [x] 2.4 Add `sections/sepay-topup.tsx`: ordered three-step transfer description plus a CTA to `/sign-up` or `/wallet` by auth state. No amount, bank account number, or transfer code anywhere in the file. ← (verify: grep the file for digits and for account/QR wording)
- [x] 2.5 Add `sections/pricing-preview.tsx` consuming `usePricingData()`, slicing to a small fixed count, with loading skeleton, error message plus `refetch` retry action, and empty-state message. No hardcoded price. ← (verify: exercise all three states)
- [x] 2.6 Update `sections/features.tsx` copy to the new bento content; keep the existing grid structure.
- [x] 2.7 Add `sections/integrations.tsx`: client-application cards plus an OpenAI-SDK `base_url` code sample.
- [x] 2.8 Add `sections/comparison.tsx`: the direct-provider vs gateway table, reflowing to stacked cards on narrow viewports.
- [x] 2.9 Add `sections/faq.tsx` consuming `useFAQ()`; return `null` when disabled, empty, or still loading. Entries expandable and keyboard-operable. ← (verify: disabled and empty both render nothing at all)
- [x] 2.10 Update `sections/cta.tsx` copy; leave its gradient treatment and `AnimateInView` usage intact.
- [x] 2.11 Delete `sections/how-it-works.tsx`.
- [x] 2.12 Update `components/index.ts` barrel: add the six new sections, drop `HowItWorks`.
- [x] 2.13 Update `features/home/index.tsx` to compose the new order. The custom-home-content branches (URL iframe, HTML, Markdown) and the `!isLoaded` branch must stay ahead of it, unchanged. ← (verify: custom content still takes precedence)

## 3. Styling and accessibility pass

- [x] 3.1 Confirm no hex, rgb, or oklch literal appears in any touched TSX file; every color, radius and shadow resolves through a theme token. ← (verify: grep the touched files — one pre-existing `rgba(...)` shadow relocated into the new `hero-supported-apps.tsx` was replaced with the token-based `shadow-xs` utility)
- [x] 3.2 Confirm every section reflows to a single column at 360 CSS pixels with no horizontal body scroll. (All new sections use `grid-cols-1`/`flex-col` defaults with `md:`/`lg:` breakpoints; `Table` uses its own `overflow-x-auto` container so the page body never scrolls horizontally; `comparison.tsx` renders stacked cards below `md` instead of the table.)
- [x] 3.3 Confirm decorative icons carry `aria-hidden='true'`, every action has an accessible name, and interactive targets reach the 44×44 floor on coarse pointers. (Added `aria-hidden='true'` to the pre-existing `BookOpen`/`ArrowRight` icons in `hero.tsx` and `cta.tsx` while touching those files; all new icons in `sepay-topup.tsx`, `pricing-preview.tsx`, `features.tsx` already carry it. `Button` applies the `pointer-coarse:touch-target-floor` utility project-wide.)
- [x] 3.4 Confirm body text uses `muted-foreground` or stronger so the 4.5:1 floor holds in both light and dark token sets.

## 4. Internationalization

- [x] 4.1 Add Vietnamese translations for every new key to `web/src/i18n/locales/vi.json`, using the wording from `giaodienmau/main.html`.
- [x] 4.2 Run `cd web && bun run i18n:sync` and confirm all seven locale files carry the new keys. ← (verify: no locale left with a missing key — sync report shows `missingCount: 0` for all seven locales)

## 5. Tests

- [x] 5.1 Add tests under `web/src/features/home/components/sections/__tests__/` covering: FAQ hidden when disabled, FAQ hidden when enabled-but-empty, FAQ rendered when populated.
- [x] 5.2 Add tests covering the pricing preview loading, error-with-retry, and empty states.
- [x] 5.3 Add a test asserting the hero action set differs between authenticated and unauthenticated visitors.

## 6. Verification

- [x] 6.1 Run `cd web && bun run typecheck` — must pass clean. ← (verify: passes clean)
- [x] 6.2 Run `cd web && bun run lint` — must pass clean. ← (verify: all touched/new files clean; the full-repo run surfaces only pre-existing errors in untouched legacy files under `features/home/components/` — out of scope)
- [x] 6.3 Run `cd web && bun run test` for the home feature tests — must pass. ← (verify: 10/10 new tests pass; full suite 509/509 passes)
- [x] 6.4 Confirm every new file carries the project AGPL copyright header. ← (verify)
- [x] 6.5 Confirm no protected identifier (project or organization name) was altered anywhere in the diff. ← (verify)
