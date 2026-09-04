## Why

The public home page is a generic developer landing page: hero, stats, features, three how-it-works steps, CTA. It says nothing about how a Vietnamese team pays for the service, what a request costs, or how to plug the gateway into the tools they already use — the three questions that decide whether a visitor from that market signs up. A static mockup for a Vietnam-oriented home page has already been designed and accepted (`giaodienmau/main.html`); this change applies that design to the real frontend.

## What Changes

- Rewrite the hero: market-specific badge and headline, three primary actions, three trust bullets, existing terminal demo kept, plus a card describing the SePay bank-transfer top-up flow.
- Add a provider strip naming the upstream model families the gateway aggregates.
- Keep the animated stats block but drop the uptime tile, which has no data source.
- Add a SePay top-up section: a numberless three-step description of the transfer flow and a call to action into sign-up or the wallet. No amounts, bank account, or QR data — those are only available behind authentication.
- Add a pricing preview fed by the existing public `GET /api/pricing` endpoint, with explicit loading, error, and empty states.
- Refresh the features bento copy.
- Replace the how-it-works section with an integrations section: the client applications the gateway is used from, plus an OpenAI-SDK `base_url` code sample.
- Add a comparison table contrasting direct provider access with the gateway.
- Add an FAQ section rendering the admin-configured FAQ from `GET /api/status`, hidden entirely when FAQ is disabled or empty.
- Update the closing CTA copy. Footer is unchanged.
- All new user-facing text goes through i18n with English source keys and Vietnamese translations.
- **Out of scope**: customer testimonials and an uptime/latency block — no data source exists for either and the project forbids fabricated figures. No backend changes and no new endpoints.

## Capabilities

### New Capabilities
- `web-home-page`: what the public home page must present to an unauthenticated visitor, which data each block is sourced from, and how each block behaves when its data is unavailable.

### Modified Capabilities
<!-- None. The design-system and theming specs already govern tokens, component vocabulary and dark mode; this change consumes them without altering their requirements. -->

## Impact

- `web/src/features/home/index.tsx` — section composition.
- `web/src/features/home/components/sections/` — hero, stats, features and CTA rewritten; how-it-works replaced; providers, SePay, pricing preview, integrations, comparison and FAQ added.
- `web/src/features/home/components/index.ts` — barrel exports.
- `web/src/features/home/constants.ts` — content constants and their translation getters.
- `web/src/i18n/locales/*.json` — new keys, Vietnamese translations, other locales synced.
- Consumes existing public endpoints only: `GET /api/pricing` and `GET /api/status` (`docs_link`, `faq_enabled`, `faq`). No server-side change.
- Design tokens in `web/src/styles/theme.css` are consumed as-is; no new token is introduced and no color is hardcoded.
