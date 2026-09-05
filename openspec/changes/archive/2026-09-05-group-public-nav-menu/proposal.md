## Why

The public header lists six items side by side: Home, Console, Model Square, Rankings, Docs, About. Two of those — Model Square (the full model catalogue with prices) and Rankings — are the surfaces a prospective customer actually comes to evaluate, yet they sit at the same weight as About. A visitor deciding whether this gateway carries the model they need has to guess which single word hides the catalogue.

The header is also the last flat list left in a console that has otherwise moved to grouped navigation, and it is close to full: Vietnamese labels run roughly 20% longer than their English equivalents, so six items already crowd the bar at the widths this audience browses on.

## What Changes

- Group the six navigation modules into four top-level entries: `Home`, a **Models** group, a **Resources** group, and `Console`.
- The two group entries open a panel listing their children with a short description each, plus a highlight cell pointing to the model catalogue.
- The panel opens on pointer, touch, and keyboard, and closes on Escape — it is not hover-only.
- On small screens the same groups render as an accordion inside the existing mobile sheet, never as a panel.
- A group whose children are all disabled by the backend does not render at all; a group left with exactly one enabled child renders as a plain link instead of a panel.
- Children keep the behaviour they have today: an entry marked `requireAuth` still raises the existing sign-in prompt, and an externally configured docs link still opens in a new tab.

Not in scope, deliberately:

- No backend change. The navigation modules, their enable flags and their `requireAuth` flags stay exactly as they are; no schema, no migration, no option.
- No new navigation destinations. The same routes are reachable before and after; only their arrangement changes.
- No model count or other figure in the panel that the frontend cannot source from real data.

## Capabilities

### New Capabilities

- `public-nav-grouping`: how the public header's fixed navigation modules are arranged into groups, how a group behaves when its children are disabled, and how the grouped navigation is presented and operated on pointer, touch, keyboard and small screens.

### Modified Capabilities

<!-- None. No existing capability's requirements change. -->

## Impact

**Frontend only.**

- `web/src/components/layout/lib/` — new pure module that turns the enabled navigation links into the grouped structure. Kept free of React so it can be unit-tested directly.
- `web/src/components/layout/components/public-header.tsx` — desktop navigation and the mobile sheet both consume the grouped structure.
- `web/src/components/layout/components/public-navigation.tsx` — the reusable navigation must stay consistent with the header.
- `web/src/components/layout/types.ts` — types for the grouped structure.
- `web/src/i18n/locales/en.json`, `web/src/i18n/locales/vi.json` — group labels, child descriptions, highlight-cell copy.

**Reused, not written**

- `web/src/components/ui/navigation-menu.tsx` already wraps `@base-ui/react/navigation-menu` and is currently unused anywhere in the codebase. It supplies pointer/touch/keyboard opening, Escape, arrow-key movement, focus handling, `aria-expanded` and collision-aware positioning. Writing a second mega-menu implementation beside it would be the wrong move.
- `web/src/components/ui/accordion.tsx` supplies the mobile disclosure.
- Icons come from `@hugeicons/react` with `@hugeicons/core-free-icons`, the set the rest of the layout already uses.

**Data source, unchanged**

`web/src/hooks/use-top-nav-links.ts` builds the list from six fixed modules — `home`, `console`, `pricing`, `rankings`, `docs`, `about` — whose enable and `requireAuth` flags arrive on `/api/status` and are parsed by `web/src/lib/nav-modules.ts`. Titles and routes are owned by the frontend; only `docs` may point outside via `status.docs_link`. Grouping is therefore a presentation concern and needs nothing from the server.
