## Why

The frontend currently ships ten user-selectable theme presets plus font, radius, scale and content-width axes, which multiplies the visual surface area without a single authoritative look. A complete visual design for all 36 screens now exists as static reference pages in `giaodienmau/`, and the product has no production users yet — so this is the cheapest moment to collapse the theme matrix into one deliberate, documented design language.

## What Changes

- Adopt one canonical visual language across the whole frontend, derived from the reference pages in `giaodienmau/` (see `giaodienmau/styleguide.html` for the authoritative palette, type ramp, radii, shadows and component anatomy).
- **BREAKING** Remove the theme preset system and the per-user appearance axes (preset, font, radius, scale, content layout) together with the Config Drawer UI that exposes them. Stored appearance cookies/preferences become inert and are cleared.
- Keep light/dark mode. Author a dark token set derived from the new light palette so both modes express the same design language.
- Replace the current token values in `web/src/styles/theme.css` with the new palette, expressed in `oklch` to match the existing token convention; retire `web/src/styles/theme-presets.css`.
- Restyle every route to the new language against the existing responsive breakpoints — mobile-first Tailwind utilities, existing mobile card lists and drawers keep working. No new breakpoints, no fixed 1440px layout in production code.
- Chart theming (`@visactor/vchart`) reads from the single token set instead of per-preset values.

## Capabilities

### New Capabilities
- `web-theming`: the single application theme — token contract, light/dark parity, what appearance choices a viewer does and does not get, and how previously stored appearance preferences are handled.
- `web-design-system`: the shared visual vocabulary every screen must use — button/badge/input/card/navigation/table-row anatomy and states, plus the accessibility floor those states must clear.

### Modified Capabilities
<!-- None: openspec/specs/ is empty, so both capabilities above are new. -->

## Impact

**Removed**
- `web/src/components/config-drawer.tsx` (745 lines) — the appearance drawer.
- `web/src/context/theme-customization-provider.tsx` (262 lines), `web/src/lib/theme-customization.ts` (197 lines).
- `web/src/styles/theme-presets.css` (742 lines).
- `web/src/components/theme-quick-switcher.tsx` — currently defined but not mounted anywhere; removed with the rest of the preset surface.

**Modified**
- `web/src/styles/theme.css`, `web/src/styles/index.css` — new token values, dark block, no preset selectors.
- `web/src/routes/__root.tsx`, `web/src/components/layout/components/app-header.tsx`, `web/src/routes/_authenticated/errors/$error.tsx` — drop the customization provider and drawer mount points.
- `web/src/lib/use-chart-theme.ts`, `web/src/features/dashboard/components/models/*`, `web/src/features/pricing/components/model-details-charts.tsx` — chart colors from the single token set.
- All 36 screens across `web/src/features/**` and `web/src/routes/**` — restyled to the new language.

**Unchanged**
- `web/src/context/theme-provider.tsx` and the light/dark switch remain; only the preset/axis layer is removed.
- Every user-facing string keeps going through i18next with English source keys (`web/src/i18n/locales/*.json`, 7 locales). This change adds no new copy.
- No backend, API, routing or data-shape change.

**Reference-only**
- `giaodienmau/` holds 36 static HTML mockups plus `index.html`. They are inline-styled, fixed at 1440px and light-only — a visual reference, never shipped and never imported. Production code uses Tailwind utilities and CSS variables per `web/AGENTS.md` §3.10.
