## Context

See `proposal.md` — Why. Constraints that shape the approach:

- **Token layer already exists and is well-shaped.** `web/src/styles/theme.css` declares a semantic token set (`--background`, `--foreground`, `--card`, `--primary`, `--secondary`, `--muted`, `--accent`, `--destructive`, `--success`, `--warning`, `--info`, `--neutral`, `--border`, `--input`, `--ring`, `--chart-1..5`, `--overview-accent-1..3`, `--sidebar-*`) in `oklch`, bridged into Tailwind v4 via `@theme inline`. Components already consume these names.
- **The variant layer sits on top of it.** `theme-presets.css` (742 lines) overrides those same tokens under `[data-theme-preset]`, `[data-theme-font]`, `[data-theme-radius]`, `[data-theme-scale]` selectors; `theme-customization.ts` + `theme-customization-provider.tsx` write the attributes and persist them in cookies; `config-drawer.tsx` is the UI. Only three non-style files mount or read this layer: `routes/__root.tsx`, `components/layout/components/app-header.tsx`, `routes/_authenticated/errors/$error.tsx`.
- **Light/dark is a separate, independent mechanism** — `context/theme-provider.tsx` plus the `.dark` class and the `@custom-variant dark` declaration. It is unaffected by removing presets.
- **Fonts are self-hosted** through `@fontsource-variable/*` packages imported in `styles/index.css`. There is no font CDN in production and no `<link>` to one in `index.html`.
- **`web/AGENTS.md` §3.10** requires Tailwind utilities with `cn()` and CSS variables, and discourages inline styles outside dynamic cases. The `giaodienmau/` mockups are inline-styled by construction; they are a visual reference, not a source to paste from.
- **`web/AGENTS.md` §3.14** requires regression tests whenever layout, sizing, focus management, selected/disabled/loading/empty/error states or responsive behavior change — which is precisely what this change touches.

## Goals / Non-Goals

**Goals:**

- Land the new visual language by changing token *values* and component *classes*, keeping token *names* stable, so the blast radius stays in the style layer.
- Delete the variant layer entirely, in one step, with no dead flags or half-removed code paths left behind.
- Produce a dark token set that is designed, not mechanically inverted.
- Give the restyle a deterministic order so half-migrated states are still coherent to look at.

**Non-Goals:**

- No new breakpoints, no new responsive strategy, no fixed-width production layout. The mockups' 1440px frame is a drawing surface, not a layout rule.
- No information-architecture change: no route added, removed or renamed; no section moved between pages; no new copy.
- No component-library migration. Base UI stays; this change restyles what is built on it.
- No re-theming of the mockups themselves. `giaodienmau/` is frozen reference; it is not kept in sync afterwards.

## Decisions

### D1. Keep semantic token names, replace their values

The new palette is expressed by overwriting the values of the existing tokens in `theme.css`, not by introducing a parallel naming scheme.

Source palette (from `giaodienmau/styleguide.html`, authored there in hex) and its token home:

| Design role | Hex | Token |
| --- | --- | --- |
| Page ground | `#F7F6FC` | `--background` |
| Surface / card | `#FFFFFF` | `--card`, `--popover` |
| Recessed surface | `#FAF9FE` | `--muted` |
| Hairline | `#EDEAF9` | `--border`, `--input` |
| Ink | `#191932` | `--foreground`, `--card-foreground` |
| Body text | `#56546F` | `--muted-foreground` (raised contrast, see D4) |
| Primary | `#6C5CF7` | `--primary`, `--ring` |
| Primary deep (text on tint) | `#4A38D4` | `--primary-foreground` inverse pairing |
| Primary tint | `#EFEBFF` | `--accent` |
| Warm accent | `#FF7A59` | `--overview-accent-2` |
| Success | `#12B77E` / text `#12996C` / tint `#E7FBF3` | `--success` family |
| Warning | `#F0A72A` / text `#B87A0E` / tint `#FFF7E0` | `--warning` family |
| Destructive | text `#C9402A` / tint `#FFEAE4` | `--destructive` family |

**Rationale.** Every existing component already reads these names. Renaming would touch hundreds of files for no behavioral gain; revaluing touches one. It also keeps the diff reviewable — a reviewer can read the token table and know what changed everywhere.

*Alternative rejected:* introduce `--brand-*` tokens alongside the existing set and migrate components gradually. Rejected because it guarantees a long window where two palettes are live, which is exactly the ambiguity this change exists to remove.

**Conversion rule:** the hex values above are the design intent; the committed values MUST be `oklch`, matching the existing file's convention. Convert once, record the `oklch` triple next to each token in a comment, and treat the `oklch` value as canonical from then on — not the hex.

### D2. Dark mode is authored, not inverted

The dark block keeps each token's hue and its *relative* role, and re-picks lightness and chroma for a dark ground:

- Ground `oklch(0.17 0.02 285)`, card one step lighter, hairlines a further step — preserving the light mode's three-surface hierarchy (ground / card / recessed) rather than flattening it.
- `--primary` moves *lighter* on dark so it clears 4.5:1 against the card; the light-mode value would fail there.
- Tint surfaces (`--accent`, and the success/warning/destructive tints) become low-alpha `color-mix` of their hue against the dark ground, instead of the light pastel, which would glare.
- The brand gradient keeps its hue sweep (`#6C5CF7 → #C044E8 → #FF7A59`) and drops in lightness.

**Rationale.** A programmatic lightness flip produces muddy tints and unreadable status colors — the failure mode the contrast requirement in `specs/web-theming/spec.md` is written to catch.

*Alternative rejected:* ship light-only and hide the toggle. Rejected by the user in this change's scoping discussion.

### D3. Remove the variant layer in one commit, and clear its cookies once

Delete `theme-presets.css`, `theme-customization.ts`, `theme-customization-provider.tsx`, `config-drawer.tsx` and `theme-quick-switcher.tsx` (the last is already unmounted). Unmount the provider in `routes/__root.tsx` and the drawer in `app-header.tsx` and `routes/_authenticated/errors/$error.tsx`.

Stored cookies are handled by a one-shot cleanup on boot that expires the keys previously written by `THEME_COOKIE_KEYS`. Absent cleanup the cookies simply linger unread — harmless but confusing during debugging.

**Rationale.** There are no production users, so a staged deprecation buys nothing and leaves two code paths to reason about. One commit, one revert point.

### D4. Radius and type scale are re-derived, not re-based

`--radius` stays `1rem`. The *derived* steps in the `@theme inline` block are re-pointed so the scale lands on the design's actual values (12 / 16 / 24 / 32 px) instead of the current multipliers (which yield 12.8 / 16 / 22.4 / 28.8). Pill shapes use Tailwind's `rounded-full`, not a token.

Body text moves to `#56546F` on `#F7F6FC`; that pair is ~6.2:1, comfortably above the floor. The mockups also use `#8F8CAB` for meta text at 11–12px — that pairing is ~3.4:1 and **fails** the 4.5:1 body-text floor. Resolution: `--muted-foreground` is set to the darker `#56546F`, and the lighter grey is admitted only for non-essential decorative text at large sizes. This is a deliberate divergence from the mockups; the spec's contrast requirement wins.

**Rationale.** Changing the base would shift every consumer of `--radius-*` unpredictably; changing the derivation targets exactly the steps the design names.

### D5. Fonts self-hosted via fontsource, matching the existing mechanism

Add `@fontsource-variable/outfit` (display) and `@fontsource-variable/plus-jakarta-sans` (body) as dependencies, import them in `styles/index.css` next to the existing imports, and point `--font-sans` / a new display token at them. Remove the `serif` font axis along with the rest of the variant layer; `@fontsource-variable/lora` and `@fontsource-variable/public-sans` are dropped once nothing references them.

**Rationale.** The mockups load fonts from the Google CDN only because they are standalone files. Production already self-hosts, which keeps the app functional offline and behind restrictive networks; introducing a CDN dependency would be a regression.

*Verify before committing:* both packages exist as variable fonts on fontsource with the weights the design uses (400–800 for Outfit, 400–700 for Plus Jakarta Sans). If a variable build is unavailable, fall back to the static `@fontsource/*` package for the needed weights rather than the CDN.

### D6. Chart colors come from tokens, read at runtime

`lib/use-chart-theme.ts` resolves `--chart-1..5` (and the surface/text tokens it needs) from computed style and hands VChart literal color values, re-resolving when the light/dark class changes. VChart cannot consume CSS variables directly.

**Rationale.** This is the only place where literal colors legitimately reach a component, and it must be re-derived per mode or dark-mode charts keep light-mode series colors.

### D7. Restyle order: tokens → primitives → shell → pages

1. **Tokens** (`theme.css`, `index.css`) — the whole app shifts palette at once; layout untouched.
2. **Primitives** (`components/ui/*`) — button, badge, input, card, select, toggle, table row. Every screen inherits these.
3. **Shell** (`components/layout/*`) — public header/footer, app sidebar and header, auth layout.
4. **Pages**, grouped as the canvas is: public → application → system settings.

**Rationale.** Steps 1–3 are ~80% of the visible change for ~20% of the files, and each step leaves the app in a coherent state. Starting at page level would mean restyling the same button twenty times.

### D8. Mockups are read, never imported

`giaodienmau/*.html` supplies geometry, hierarchy and state treatments. Implementation re-expresses them as Tailwind utilities against tokens. No inline `style` attribute and no hex literal is copied into `web/src/`, per `web/AGENTS.md` §3.10.

## Risks / Trade-offs

- **Token revaluation silently breaks a screen that relied on the old hue** (e.g. text placed on a surface that used to be light and is now tinted) → Step 1 lands alone and every route is walked in both modes before step 2 begins; contrast checks run against the token pairs, not per screen.
- **Removing the preset layer while restyling makes one large, hard-to-review diff** → the two are separate commits: removal (deletions plus three unmount sites) precedes restyling, so each is reviewable on its own.
- **Mockups are light-only, so dark mode has no reference to check against** → the spec's contrast scenarios are the acceptance test for dark; every screen is reviewed in dark mode at the end of its group rather than at the end of the change.
- **The mockups' meta-text grey fails the contrast floor** (see D4) → resolved in favor of the spec; the design deviation is recorded here so a later reviewer does not "fix" it back toward the mockup.
- **Fixed-1440px mockups tempt fixed-width production layout** → no `width:` value from a mockup is carried over; small-viewport behavior of each restyled screen is checked against its current behavior before the screen is considered done.
- **Long translations in 6 non-English locales overflow tighter new spacing** → each restyled screen is checked in at least one long-form locale; wrap/truncate rules are made explicit rather than left to chance.
- **`bun run build` output grows from two new font families** → the two retired families are removed in the same change; bundle size is compared before and after.

## Migration Plan

1. Add font dependencies; land the token rewrite (light + dark) in `theme.css` / `index.css`.
2. Delete the variant layer and its three mount points; add the one-shot cookie cleanup.
3. Restyle primitives, then shell, then pages by group (D7), with tests per `web/AGENTS.md` §3.14 accompanying each group.
4. Full pass in both modes at desktop and small viewport; `bun run typecheck`, lint on touched files, `bun run build`.

**Rollback:** `git revert` of the range. No data migration, no persisted state to unwind — the only stored artifacts are the appearance cookies, which are already inert after step 2.
