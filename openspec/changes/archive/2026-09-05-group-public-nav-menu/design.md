## Context

See `proposal.md` — Why, for motivation. Requirements live in `specs/public-nav-grouping/spec.md`.

Constraints established by reading the code before planning:

- **The navigation is a fixed module set, not free-form links.** `web/src/hooks/use-top-nav-links.ts` builds the list from exactly six modules — `home`, `console`, `pricing`, `rankings`, `docs`, `about`. `/api/status` carries only enable flags and `requireAuth` per module, parsed by `web/src/lib/nav-modules.ts` (`DEFAULT_HEADER_NAV_MODULES`). Titles and routes are frontend-owned; only `docs` can point outside, via `status.docs_link`. Grouping therefore needs nothing from the server.
- **A navigation-menu primitive already exists and is unused.** `web/src/components/ui/navigation-menu.tsx` wraps `@base-ui/react/navigation-menu` and exports `NavigationMenu`, `NavigationMenuList`, `NavigationMenuItem`, `NavigationMenuTrigger`, `NavigationMenuContent`, `NavigationMenuPositioner`, `NavigationMenuLink`, `NavigationMenuIndicator` and `navigationMenuTriggerStyle`. A repo-wide search finds no importer.
- **The header owns behaviour the panel must not break.** `public-header.tsx` intercepts clicks through `handleNavLinkClick`, which raises a timed sign-in prompt for `requiresAuth` entries and closes the mobile sheet. The mobile sheet is a full-screen overlay driven by `mobileOpen`, with a staggered reveal keyed on each item's index.
- **`/api/status` has no model count.** Confirmed in `controller/misc.go`.
- **Vitest has a known ESM hazard in this repo**: importing modules that transitively pull `@lobehub/fluent-emoji` fails on a directory import.

## Goals / Non-Goals

**Goals:**

- Give the model-discovery surfaces a place of their own without adding destinations.
- Get the accessibility surface — pointer, touch, keyboard, Escape, focus return, arrow keys, `aria-expanded` — from the existing primitive rather than hand-rolling it.
- Keep the arrangement correct under every combination of module enable flags, including the ones that empty a group.
- Keep the grouping decision testable without rendering anything.

**Non-Goals:**

- No backend work of any kind.
- No redesign of the header shell — its two scroll states, logo, utility cluster and auth controls stay as they are.
- No animation work beyond what the primitive and the existing mobile stagger already provide.

## Decisions

### Group in the frontend, from the module list, with no server involvement

The alternative — extending `/api/status` with a group structure — was considered and rejected on evidence. The module set is closed and frontend-owned; a server-side grouping schema would add a migration, a three-database verification matrix and a backward-compatibility burden to express something the client already knows statically. The server's job here is to say which modules are on, and it keeps doing exactly that.

### A pure grouping function, separate from the components

The arrangement rule (which children belong to which group, and what to do when a group has zero or one enabled child) lives in a React-free module under `web/src/components/layout/lib/`. It takes the already-filtered link list from `useTopNavLinks` and returns a discriminated structure of plain links and groups.

Two reasons. First, the collapse rules are the part most likely to be wrong and are pure input/output — a table test covers every enable-flag combination with no DOM. Second, it sidesteps the `@lobehub/fluent-emoji` ESM hazard entirely: the test imports a module that pulls no components.

Matching children to groups keys off the destination route rather than the translated title, since titles arrive already translated and vary by language.

### Build the panel on the existing Base UI wrapper

`navigation-menu.tsx` exists, wraps a mature primitive, and is unused. Using it delivers the whole spec requirement "opens by pointer, touch and keyboard" plus Escape, focus return, arrow keys and collision-aware positioning without a line of our own event handling — hand-rolling that is where mega menus usually fail for keyboard and touch users.

Being its first consumer carries a real risk: the wrapper has never been exercised, so its styling and positioning defaults are unproven. The mitigation is to verify the interactions in the running dev server rather than trusting the wrapper's appearance, and to fix the wrapper rather than work around it if a default is wrong.

### Children route through the header's existing click handler

The panel's children call the same `handleNavLinkClick` the flat links call today, so the sign-in prompt, the disabled state and the mobile-sheet close all keep working from inside the panel. When the handler raises the prompt, the panel closes — a modal appearing behind an open menu is the failure mode to avoid.

### The highlight cell carries copy, not a count

The design sketch showed "75 models". That number came from a competitor's site. `/api/status` does not carry a model count, and fetching the full model list on every public page load to render one number in a header is a poor trade. The cell keeps its role as the panel's selling surface through copy and a link to the catalogue. Adding a real count later is a separate change with a real data source behind it.

### Group and child descriptions are authored strings

Child descriptions ("the full catalogue with prices", "which models are used most") do not exist anywhere in the module metadata, so they are new translated strings keyed by their English source, in both locale files. They are copy, not data.

## Risks / Trade-offs

- **First use of an unexercised wrapper** → Verify the real interactions in the browser at the running dev server, in both languages and at phone width, before calling the change done. If a wrapper default is wrong, fix the wrapper; do not paper over it in the consumer.
- **Vietnamese labels run ~20% longer** → The panel is a three-column grid of equal fractions rather than fixed columns, and the check at phone width and in Vietnamese is a task, not an afterthought.
- **Fewer top-level entries means one more click to Docs and About** → Accepted deliberately: those two are reference destinations, while the catalogue and rankings are what the visitor is evaluating. The trade buys the catalogue a described, weighted position instead of a bare word.
- **Grouping is static while module flags are dynamic** → The collapse rules exist precisely for this, and their table test is the guard. An operator disabling both `pricing` and `rankings` must see the group disappear, not an empty panel.
- **Two navigation renderers exist** (`public-header.tsx` inline, and `public-navigation.tsx`) → They must not drift. Both consume the same grouping function; if `public-navigation.tsx` turns out to be genuinely unused, say so in the report rather than silently leaving it behind.

## Migration Plan

None. The change is presentational and frontend-only: no schema, no persisted state, no API surface. Deployment is a normal frontend build; rollback is reverting the build. Nothing on the server is aware the arrangement changed.
