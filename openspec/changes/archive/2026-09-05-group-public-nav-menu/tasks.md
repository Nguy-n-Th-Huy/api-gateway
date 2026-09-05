## 1. Grouping logic

- [x] 1.1 Add types for the grouped navigation to `web/src/components/layout/types.ts`: a discriminated structure whose members are either a plain link or a group carrying a label, an optional highlight, and its child links.
- [x] 1.2 Add a React-free module under `web/src/components/layout/lib/` exporting a pure function that takes the link list from `useTopNavLinks` and returns the grouped structure. Match children to groups by destination route, not by title — titles arrive already translated.
- [x] 1.3 Implement the collapse rules in that function: zero enabled children drops the group entirely; exactly one enabled child yields a plain link to that child; two or more yield a group. ← (verify: every enable-flag combination behaves per spec; an emptied group leaves no residue in the output)
- [x] 1.4 Add a table test in `__tests__/` beside the module covering: all modules enabled, both models children disabled, only one models child enabled, both resources children disabled, only one resources child enabled, and an external docs link. Import only the pure module so the test pulls no components. ← (verify: assertions are on exact returned structures and would fail if a collapse rule were removed)

## 2. Desktop panel

- [x] 2.1 Add a panel component under `web/src/components/layout/components/` built on the existing `web/src/components/ui/navigation-menu.tsx` exports. Do not hand-roll open/close, focus or key handling — the primitive owns them.
- [x] 2.2 Lay the panel out per `design.md` and the approved sketch: centred, roughly 880px wide, card background, soft shadow, hairline `--border`, 20px padding, a three-column grid of equal fractions with 20px gaps. Use Tailwind classes and project tokens only — no hardcoded colour values.
- [x] 2.3 Render each child row with an icon tile, a title and a description, using `@hugeicons/react` with `@hugeicons/core-free-icons` — the set the layout already uses. No hand-drawn SVG.
- [x] 2.4 Render the highlight cell on `--accent` with display-font heading and a link to the model catalogue. It must carry no model count or comparable figure. ← (verify: no numeric claim about the catalogue appears anywhere in the panel)
- [x] 2.5 Wire the panel's children through the header's existing `handleNavLinkClick` so `requiresAuth` still raises the sign-in prompt and disabled entries stay inert; closing on prompt is required. External docs links render as `<a target="_blank" rel="noopener noreferrer">`; the current page keeps `aria-current="page"`. ← (verify: sign-in prompt still fires from inside the panel and the panel closes when it does)

## 3. Header integration

- [x] 3.1 Replace the inline flat desktop navigation in `web/src/components/layout/components/public-header.tsx` with the grouped structure, keeping the header shell untouched — both scroll states, logo, utility cluster and auth controls stay exactly as they are.
- [x] 3.2 Update `web/src/components/layout/components/public-navigation.tsx` to consume the same grouping function so the two renderers cannot drift. If it proves to be genuinely unused, report that rather than leaving it inconsistent.
- [x] 3.3 Replace the mobile sheet's flat list with the existing `web/src/components/ui/accordion.tsx` for groups, keeping the sheet's staggered reveal. Plain links stay plain rows. Every activatable row is at least 44px tall. ← (verify: no desktop panel appears at phone width; groups expand in place)

## 4. Translations

- [x] 4.1 Add every new string — group labels, child descriptions, highlight-cell copy — to both `web/src/i18n/locales/en.json` and `web/src/i18n/locales/vi.json`, keyed by the English source string, rendered via `t()`. No literal UI text in the new components. ← (verify: each new key exists in both locales and nothing is hardcoded)

## 5. Verification

- [x] 5.1 Run typecheck, lint scoped to the touched files, and the Vitest suite for the layout module from `web/`; report the real results without suppressing failures.
- [x] 5.2 VERIFIED. Browser-exercised (pointer open/close, keyboard Enter-to-open, tap emulation via a same-origin iframe at 390px, and Escape closing the panel) all worked as expected, confirmed repeatedly. Two further items were confirmed by reading the installed `@base-ui/react` source rather than by driving the browser: (a) focus returning to the trigger after Escape — `navigation-menu/root/NavigationMenuRoot.js:134-143` restores focus to `prevTriggerElementRef.current` whenever the active element is the document body or is contained by the popup element; in this wrapper `Viewport` renders inside `Popup` (`web/src/components/ui/navigation-menu.tsx`), so a focused content link is contained by the popup and the restore path runs, and Escape's close reason is not in `blockedReturnFocusReasons`, so no extra handling is needed; (b) arrow-key movement among a panel's own children — `content/NavigationMenuContent.js` renders a `CompositeRoot` with orientation `'both'` and `loopFocus`, `link/NavigationMenuLink.js` is a `CompositeItem`, and `useCompositeRoot.js` handles the arrow keys and moves focus between items; `NavigationMenuLink` also sets `tabIndex: undefined`, overriding the roving `-1`, so Tab/Shift+Tab reaches every item too. No changes to `web/src/components/ui/navigation-menu.tsx` were needed for either item.
- [x] 5.3 Checked at 390px width (via a same-origin iframe, since the browser window would not resize through the automation tool) in both Vietnamese and English: the desktop panel never appears (only the hamburger sheet), tapping a group expands it in place with no floating panel, and measured row heights are Home 48px / Models trigger 50px / Model Square 44px / Rankings 44px / Resources trigger 50px / Console 48px — all ≥44px. No wrapping or overflow observed in either language.
