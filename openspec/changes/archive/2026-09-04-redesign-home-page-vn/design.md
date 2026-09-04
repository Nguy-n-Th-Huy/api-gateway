## Context

See `proposal.md` — Why. The current home page is composed in `web/src/features/home/index.tsx` from five sections under `components/sections/` plus the shared footer. Content constants and their translation getters already live in `features/home/constants.ts`; animation helpers (`AnimateInView`, `landing-animate-fade-up`) and the design tokens in `web/src/styles/theme.css` already exist and are consumed by the current sections.

An accepted static mockup for the new layout exists at `giaodienmau/main.html`. It is a design reference only — it uses the mockup folder palette and carries bracketed placeholders for every real figure. The implementation takes its structure and copy, not its colors or its placeholders.

Two public data sources already have frontend access paths:
- `getPricing()` / `usePricingData()` in `features/pricing/` wrap the public `GET /api/pricing`.
- `useFAQ()` in `features/dashboard/hooks/use-status-data.ts` reads `faq_enabled` + `faq` off `useStatus()`, which is fed by the public `GET /api/status`.

## Goals / Non-Goals

**Goals:**
- Reuse the existing data hooks rather than adding new API calls for the same data.
- Keep every section file under the 200-line ceiling from `web/AGENTS.md`, splitting content data out into constants.
- Keep the page renderable when either public data source is unavailable.

**Non-Goals:**
- No change to `PublicLayout`, the footer, the header, or routing.
- No new design token, no new shared UI component in `components/ui/`.
- No change to `giaodienmau/`; the mockup stays as the reference artifact it is.

## Decisions

**Reuse `usePricingData()` instead of calling `getPricing()` directly.** It already normalizes vendors onto models and derives the currency rate from `status`, and it shares the `['pricing']` query key with the pricing page, so a visitor who follows the "see full pricing" link hits a warm cache. The alternative — a home-local `useQuery` — would duplicate the vendor join and split the cache. The home page passes a small `enabled` window and slices the result down to a handful of models client-side; introducing a new server-side "featured models" endpoint is out of scope and would need admin configuration to be meaningful.

**Reuse `useFAQ()` from `features/dashboard/hooks/`.** Cross-feature imports are already established in this codebase (`features/pricing/` imports from `features/performance-metrics/`). Duplicating the `status.faq_enabled` / `status.faq` read in `features/home/` would be a DRY violation on a shape that can change.

**FAQ section returns `null` rather than rendering an empty shell.** The spec requires the whole block to disappear when disabled or empty. Rendering a heading with an empty list would leave a dead region on the page and imply the operator forgot to fill it in. While `useStatus()` is still loading, the section renders nothing — a skeleton would flash a heading that may never gain content.

**Pricing preview owns its own loading / error / empty treatments rather than hiding on failure.** Unlike the FAQ, the pricing block is a promised part of the page: silently removing it changes the page structure between loads. It renders a fixed-height skeleton while loading, a message plus a retry action wired to the hook's `refetch` on error, and an empty message when the model list is empty.

**Content lives in `constants.ts` as `get*(t)` functions.** This matches the existing `getGatewayFeatures(t)` / `getDefaultStats(t)` pattern, keeps the English source strings visible to the `t('...')` scanner used by `bun run i18n:sync`, and keeps each section component focused on layout.

**Section files split by block, one file per block.** `sections/` gains `providers.tsx`, `sepay-topup.tsx`, `pricing-preview.tsx`, `integrations.tsx`, `comparison.tsx` and `faq.tsx`; `how-it-works.tsx` is removed and its barrel export with it. Any file approaching 200 lines splits its rows/cards into a sibling presentational component rather than growing.

**Vietnamese copy is a translation, not a hardcoded string.** Components call `t('English source string')`; `vi.json` carries the Vietnamese wording taken from the mockup; the remaining six locales are filled by `bun run i18n:sync`. Strings that the scanner cannot see (built from constants) are registered in `web/src/i18n/static-keys.ts`.

**The uptime tile and the testimonials block are dropped, not stubbed.** The mockup carries `[UPTIME]`, `[HỌ TÊN]`, `[CÔNG TY]` placeholders precisely because no data source exists. Shipping them with invented values would violate the no-fabricated-figures requirement; shipping them with visible brackets would ship a broken page.

## Risks / Trade-offs

- **Page weight and time to interactive grow with eleven blocks** → all below-the-fold blocks keep the existing `AnimateInView` intersection-observer pattern, so their animations do not run until scrolled into view; no new library is added.
- **`/api/pricing` can be large on instances with hundreds of models** → the preview slices to a small fixed count client-side after the hook's existing memoized transform; the full list is already fetched by the pricing page with the same cache entry, so no extra network cost is introduced for a visitor who continues there.
- **FAQ content is admin-authored and rendered on a public page** → answers are rendered through the same path the dashboard FAQ panel already uses, so no new rendering surface is introduced.
- **A longer page increases the surface for token drift** → no color, radius, shadow, or font value is written literally in TSX; everything resolves through the theme tokens, which keeps dark mode correct by construction.
- **Seven locale files gain many keys at once** → `bun run i18n:sync` fills non-Vietnamese locales with the English source, which is the project's established fallback behavior; no locale is left with missing keys.

## Migration Plan

Frontend-only, no data migration. The change is a single deploy of the web bundle. Rollback is a revert of the commit; no persisted state, schema, or API contract is touched, and instances relying on custom home page content are unaffected because that branch is preserved ahead of the default composition.
