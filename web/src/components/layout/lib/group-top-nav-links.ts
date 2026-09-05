/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { TFunction } from 'i18next'

import type { GroupedNavEntry, NavGroupHighlight, TopNavLink } from '../types'

const HOME_ROUTE = '/'
const CONSOLE_ROUTE = '/dashboard'
const PRICING_ROUTE = '/pricing'
const RANKINGS_ROUTE = '/rankings'
const DOCS_ROUTE = '/docs'
const ABOUT_ROUTE = '/about'

function isModelsChild(link: TopNavLink): boolean {
  return link.href === PRICING_ROUTE || link.href === RANKINGS_ROUTE
}

function isDocsChild(link: TopNavLink): boolean {
  // Docs is the only module that can point outside the app (via
  // `status.docs_link`), so an external link is always the docs entry —
  // matching by href alone would miss it once it points off-site.
  return link.href === DOCS_ROUTE || !!link.external
}

function isResourcesChild(link: TopNavLink): boolean {
  return isDocsChild(link) || link.href === ABOUT_ROUTE
}

function describeModelsChild(link: TopNavLink, t: TFunction): string {
  return link.href === PRICING_ROUTE
    ? t('The full catalogue with prices')
    : t('Which models are used most')
}

function describeResourcesChild(link: TopNavLink, t: TFunction): string {
  return isDocsChild(link)
    ? t('Guides and API reference')
    : t('Who we are and what we offer')
}

function buildModelsHighlight(t: TFunction): NavGroupHighlight {
  return {
    title: t('Explore the full catalogue'),
    description: t('One endpoint, one key. Prices shown in your currency.'),
    linkLabel: t('View Pricing'),
    href: PRICING_ROUTE,
  }
}

/**
 * Collapses a group's already-filtered children per the spec: zero enabled
 * children drops the group entirely, exactly one yields a plain link to
 * that child (no disclosure, no panel), and two or more yield a group entry
 * whose panel lists exactly those children.
 */
function buildGroup(
  id: string,
  label: string,
  children: TopNavLink[],
  describe: (link: TopNavLink, t: TFunction) => string,
  t: TFunction,
  highlight?: NavGroupHighlight
): GroupedNavEntry | undefined {
  if (children.length === 0) return undefined

  if (children.length === 1) {
    const [only] = children
    return { kind: 'link', ...only }
  }

  return {
    kind: 'group',
    id,
    label,
    highlight,
    children: children.map((child) => ({
      ...child,
      description: describe(child, t),
    })),
  }
}

/**
 * Turns the flat, already-filtered link list from `useTopNavLinks` into the
 * public header's grouped structure: a home entry, a models group, a
 * resources group, and a console entry, in that order.
 *
 * A module the backend has disabled never appears in `links`, so a group
 * whose children are all disabled naturally has zero candidates here and is
 * dropped; a group left with exactly one enabled child collapses to a plain
 * link. Children are matched to their group by destination route, never by
 * title, since titles arrive already translated and vary by language.
 *
 * Kept free of React and hooks so it can be unit-tested by passing a plain
 * translator function — no rendering, no DOM.
 */
export function groupTopNavLinks(
  links: TopNavLink[],
  t: TFunction
): GroupedNavEntry[] {
  const entries: GroupedNavEntry[] = []

  const home = links.find((link) => link.href === HOME_ROUTE)
  if (home) entries.push({ kind: 'link', ...home })

  const modelsChildren = links.filter(isModelsChild)
  const modelsGroup = buildGroup(
    'models',
    t('Models'),
    modelsChildren,
    describeModelsChild,
    t,
    modelsChildren.length >= 2 ? buildModelsHighlight(t) : undefined
  )
  if (modelsGroup) entries.push(modelsGroup)

  const resourcesChildren = links.filter(isResourcesChild)
  const resourcesGroup = buildGroup(
    'resources',
    t('Resources'),
    resourcesChildren,
    describeResourcesChild,
    t
  )
  if (resourcesGroup) entries.push(resourcesGroup)

  const consoleLink = links.find((link) => link.href === CONSOLE_ROUTE)
  if (consoleLink) entries.push({ kind: 'link', ...consoleLink })

  return entries
}
