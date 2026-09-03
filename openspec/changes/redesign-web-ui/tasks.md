## 1. Token foundation

- [ ] 1.1 Add `@fontsource-variable/outfit` and `@fontsource-variable/plus-jakarta-sans` with `bun add`; confirm variable builds cover Outfit 400–800 and Plus Jakarta Sans 400–700, else fall back to static `@fontsource/*` packages (design.md D5)
- [ ] 1.2 Import both faces in `web/src/styles/index.css` next to the existing font imports; point `--font-sans` at Plus Jakarta Sans and add a display token for Outfit
- [ ] 1.3 Convert the D1 palette table from hex to `oklch` and rewrite the light token block in `web/src/styles/theme.css`, keeping every existing token name and recording each `oklch` triple in a comment
- [ ] 1.4 Author the dark token block per design.md D2 — three-surface hierarchy preserved, `--primary` lightened, tints as low-alpha `color-mix` against the dark ground
- [ ] 1.5 Re-point the derived radius steps in the `@theme inline` block so the scale lands on 12 / 16 / 24 / 32 px with `--radius` unchanged at `1rem` (design.md D4)
- [ ] 1.6 Set `--muted-foreground` to the raised-contrast body value, not the mockups' meta grey; add a comment recording that this deviates from `giaodienmau/` on purpose
- [ ] 1.7 Measure contrast for every text and control token pair in both modes; record the ratios and fix any pair below 4.5:1 body / 3:1 large-and-controls ← (verify: every pair in the recorded table clears its floor in BOTH modes; no pair is left unmeasured)
- [ ] 1.8 Walk every route in both modes with only tokens changed; confirm no screen has unreadable text or lost surface hierarchy before any component work begins ← (verify: token swap alone leaves all 36 screens legible; layout is untouched at this point)

## 2. Remove the theme variant layer

- [ ] 2.1 Delete `web/src/styles/theme-presets.css` and its import
- [ ] 2.2 Delete `web/src/lib/theme-customization.ts` and `web/src/context/theme-customization-provider.tsx`
- [ ] 2.3 Delete `web/src/components/config-drawer.tsx` and `web/src/components/theme-quick-switcher.tsx`
- [ ] 2.4 Unmount the customization provider in `web/src/routes/__root.tsx`
- [ ] 2.5 Remove the `ConfigDrawer` mount and the `showConfigDrawer` prop from `web/src/components/layout/components/app-header.tsx`, and the mount in `web/src/routes/_authenticated/errors/$error.tsx`
- [ ] 2.6 Add a one-shot boot cleanup that expires the cookies previously written by `THEME_COOKIE_KEYS`
- [ ] 2.7 Remove `@fontsource-variable/lora` and `@fontsource-variable/public-sans` once nothing references them; drop the `serif` font axis from the token layer
- [ ] 2.8 Grep for every remaining reference to `theme-preset`, `themePreset`, `data-theme-font`, `data-theme-radius`, `data-theme-width`, `theme-customization` and `ConfigDrawer`; run `bun run typecheck` and lint the touched files ← (verify: zero references survive anywhere in `web/src`; typecheck clean; light/dark toggle still works and persists across reload)
- [ ] 2.9 Add a regression test asserting no appearance-customization control is reachable and that a stale appearance cookie does not alter rendering (spec `web-theming`)

## 3. Shared primitives

- [ ] 3.1 Restyle the button component to the `giaodienmau/styleguide.html` anatomy — primary gradient pill, solid dark, outline, ghost and disabled variants
- [ ] 3.2 Restyle badge / status pill variants: primary, success, warning, destructive, neutral
- [ ] 3.3 Restyle text input, select and textarea including idle, focused (ring), invalid and disabled states
- [ ] 3.4 Restyle card, panel and section surfaces — radii, hairline, soft and floating elevation
- [ ] 3.5 Restyle segmented control, switch and checkbox
- [ ] 3.6 Restyle navigation item (sidebar and top nav) with active, hover and focus treatments
- [ ] 3.7 Restyle data-table header, row, hover, selected and empty states in `web/src/components/data-table/*`
- [ ] 3.8 Ensure every restyled primitive defines default, hover, active, focus-visible, disabled and — where applicable — loading and selected states, none conveyed by color alone (spec `web-design-system`)
- [ ] 3.9 Add primitive-level regression tests under each module's `__tests__/` covering focus visibility, disabled inertness, loading double-submit prevention and the 44px touch-target floor ← (verify: tests assert behavior contracts and accessible state, not Tailwind class strings; all primitive states render correctly in both modes)

## 4. Application shell

- [ ] 4.1 Restyle the public header and footer (`components/layout/components/public-header.tsx`, `public-navigation.tsx`, `footer.tsx`) per `giaodienmau/main.html`
- [ ] 4.2 Restyle the authenticated sidebar and its four nav groups (`app-sidebar.tsx`, `nav-group.tsx`, `nav-link-item.tsx`) per `giaodienmau/overview.html`
- [ ] 4.3 Restyle the authenticated header (`app-header.tsx`) — breadcrumb, page title, search, notifications, profile
- [ ] 4.4 Restyle the system-settings drill-in sidebar (`sidebar-view-header.tsx` and the settings nav view) per `giaodienmau/settings-site.html`
- [ ] 4.5 Restyle the auth layout (`features/auth/auth-layout.tsx`) to the split brand-panel / form-card composition per `giaodienmau/sign-in.html`
- [ ] 4.6 Verify the shell's existing small-viewport behavior — mobile drawer, collapsed sidebar, stacked header — still triggers at the same breakpoints, restyled ← (verify: no breakpoint moved; drawer and collapse behave as before; no page-level horizontal scroll at any supported width)
- [ ] 4.7 Add shell regression tests for sidebar collapse, mobile drawer open/close and focus return

## 5. Public and auth pages

- [ ] 5.1 Home — `routes/index.tsx` + `features/home/**` per `giaodienmau/main.html` (hero, stats band, bento features, how-it-works, CTA, footer)
- [ ] 5.2 Pricing list — `routes/pricing` + `features/pricing/**` per `giaodienmau/pricing.html` (filter sidebar, toolbar, model card grid, table view)
- [ ] 5.3 Model detail — `routes/pricing/$modelId` per `giaodienmau/model-detail.html`
- [ ] 5.4 Rankings — `routes/rankings` + `features/rankings/**` per `giaodienmau/rankings.html`
- [ ] 5.5 About — `routes/about` + `features/about/**` per `giaodienmau/about.html`
- [ ] 5.6 Legal — `routes/user-agreement` and `routes/privacy-policy` + `features/legal/**` per `giaodienmau/legal.html`
- [ ] 5.7 Sign in — `routes/(auth)/sign-in` per `giaodienmau/sign-in.html`
- [ ] 5.8 Sign up — `routes/(auth)/sign-up` per `giaodienmau/sign-up.html`
- [ ] 5.9 Forgot password and reset — `routes/(auth)/forgot-password`, `routes/(auth)/reset`, `routes/(auth)/user/reset` per `giaodienmau/forgot-password.html`
- [ ] 5.10 OTP / two-factor — `routes/(auth)/otp` per `giaodienmau/otp.html`
- [ ] 5.11 OAuth callback — `routes/(auth)/oauth`, `routes/oauth/$provider` reusing the auth layout treatment
- [ ] 5.12 Setup wizard — `routes/setup` + `features/setup/**` per `giaodienmau/setup-wizard.html` (stepper, admin account step)
- [ ] 5.13 Error pages — `routes/(errors)/{401,403,404,500,503}` + `features/errors/**` per `giaodienmau/error-pages.html`
- [ ] 5.14 Check every page in this group at a small viewport and in one long-form locale; fix wrapping, clipping and overlap ← (verify: all 13 public/auth screens hold up at small viewport and in a non-English locale; every string still resolves through i18next with no hardcoded literal)

## 6. Application pages

- [ ] 6.1 Overview — `routes/_authenticated/dashboard/index.tsx` + `features/dashboard/**` per `giaodienmau/overview.html` (stat cards, traffic chart, top models, recent requests)
- [ ] 6.2 Dashboard by model — `routes/_authenticated/dashboard/$section.tsx` per `giaodienmau/dashboard-models.html` (cost chart, provider distribution, per-model table)
- [ ] 6.3 API keys — `routes/_authenticated/keys` + `features/keys/**` per `giaodienmau/keys.html`
- [ ] 6.4 Usage logs — `routes/_authenticated/usage-logs/**` + `features/usage-logs/**` per `giaodienmau/usage-logs.html` (filter bar, stat row, table, pagination)
- [ ] 6.5 Task logs — the task and drawing sections of `routes/_authenticated/usage-logs/$section.tsx` per `giaodienmau/task-logs.html` (progress cells, platform badges)
- [ ] 6.6 Wallet — `routes/_authenticated/wallet` + `features/wallet/**` per `giaodienmau/wallet.html` (balance card, redeem, top-up amounts, transaction history)
- [ ] 6.7 Profile — `routes/_authenticated/profile` + `features/profile/**` per `giaodienmau/profile.html` (identity card, linked accounts, personal info, security)
- [ ] 6.8 Playground — `routes/_authenticated/playground` + `features/playground/**` per `giaodienmau/playground.html` (conversation pane, parameter rail)
- [ ] 6.9 Chat — `routes/_authenticated/chat/$chatId`, `chat2link` + `features/chat/**` per `giaodienmau/chat-view.html` (conversation list, message bubbles, composer)
- [ ] 6.10 Channels — `routes/_authenticated/channels` + `features/channels/**` per `giaodienmau/channels.html`
- [ ] 6.11 Models admin — `routes/_authenticated/models/**` + `features/models/**` per `giaodienmau/models-admin.html`
- [ ] 6.12 Users — `routes/_authenticated/users` + `features/users/**` per `giaodienmau/users-admin.html` (avatar cell, role, quota)
- [ ] 6.13 Redemption codes — `routes/_authenticated/redemption-codes` + `features/redemption-codes/**` per `giaodienmau/redemption-codes.html`
- [ ] 6.14 Subscriptions — `routes/_authenticated/subscriptions` + `features/subscriptions/**` per `giaodienmau/subscriptions.html` (plan cards, subscriber table)
- [ ] 6.15 System info — `routes/_authenticated/system-info` + `features/system-info/**` per `giaodienmau/system-info.html` (version table, service health, log console)
- [ ] 6.16 Task plugins — `routes/_authenticated/task-plugins` + `features/task-plugins/**` per `giaodienmau/task-plugins.html`
- [ ] 6.17 Authenticated error route — `routes/_authenticated/errors/$error.tsx` aligned with the public error treatment
- [ ] 6.18 Check every page in this group at a small viewport; confirm existing card-list and drawer fallbacks still trigger and are restyled ← (verify: all 16 application screens keep their current responsive adaptations; tables scroll within their own container, never the page body)
- [ ] 6.19 Add regression tests for the state-bearing surfaces touched here — table empty/loading/error states, wallet redeem form validation, playground parameter controls

## 7. System settings pages

- [ ] 7.1 Site & branding — `routes/_authenticated/system-settings/site/**` per `giaodienmau/settings-site.html`
- [ ] 7.2 Authentication — `routes/_authenticated/system-settings/auth/**` per `giaodienmau/settings-auth.html`
- [ ] 7.3 Billing & payment — `routes/_authenticated/system-settings/billing/**` per `giaodienmau/settings-billing.html`
- [ ] 7.4 Models & routing — `routes/_authenticated/system-settings/models/**` per `giaodienmau/settings-models.html`
- [ ] 7.5 Security & limits — `routes/_authenticated/system-settings/security/**` per `giaodienmau/settings-security.html`
- [ ] 7.6 Console content — `routes/_authenticated/system-settings/content/**` per `giaodienmau/settings-content.html`
- [ ] 7.7 Operations — `routes/_authenticated/system-settings/operations/**` per `giaodienmau/settings-operations.html`
- [ ] 7.8 Confirm every settings section registered in each `section-registry.tsx` is restyled, not only the section shown in the mockup ← (verify: all sections across the seven groups are covered; form dirty-state, validation errors and the navigation guard still behave as before)

## 8. Charts

- [ ] 8.1 Rework `web/src/lib/use-chart-theme.ts` to resolve `--chart-1..5` plus surface and text tokens from computed style and re-resolve when the light/dark class changes (design.md D6)
- [ ] 8.2 Apply the resolved theme in `features/dashboard/components/models/model-charts.tsx` and `consumption-distribution-chart.tsx`
- [ ] 8.3 Apply the resolved theme in `features/pricing/components/model-details-charts.tsx` and the uptime sparkline
- [ ] 8.4 Verify categorical series stay distinguishable and axes, gridlines and labels stay legible in both modes ← (verify: charts re-render on mode switch with no light-mode colors left on a dark surface; adjacent series remain distinguishable)

## 9. Final verification

- [ ] 9.1 Full pass over all 36 screens in light mode and dark mode at desktop width
- [ ] 9.2 Full pass at small viewport width; confirm no page-level horizontal scroll anywhere
- [ ] 9.3 Keyboard-only pass over the primary flows — sign in, create API key, top up wallet, edit a system setting — confirming visible focus throughout
- [ ] 9.4 Confirm no hex literal or inline `style` attribute introduced by this change remains in `web/src` outside the documented dynamic cases (`web/AGENTS.md` §3.10)
- [ ] 9.5 Run `bun run typecheck`, lint the touched files, run the affected test files and the frontend test suite
- [ ] 9.6 Run `bun run build`; compare bundle size against the pre-change baseline and account for the font swap ← (verify: build succeeds, typecheck and lint clean, all tests pass, bundle size change is explained by the two added minus two removed font families)
